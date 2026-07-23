package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"gateguard/internal/models"
	"gateguard/internal/repository"
)

// ErrVerificationTokenInvalid is returned when an email/token pair does not match.
var ErrVerificationTokenInvalid = errors.New("verification token invalid")

const verificationResendCooldown = 60 * time.Second

// ErrVerificationCooldown is returned when a code was sent less than the cooldown ago.
var ErrVerificationCooldown = errors.New("verification code recently sent")

// 24h: the code arrives unannounced, so a short clock strands users who don't
// check mail immediately. Brute force is bounded by verificationMaxAttempts
// below, not by this TTL.
const verificationCodeTTL = 24 * time.Hour

// verificationMaxAttempts caps wrong guesses per issued code. A 6-digit code is
// 1,000,000 combinations; without this cap the TTL is the only bound.
const verificationMaxAttempts = 5

// ErrVerificationCodeExpired is returned when a matching code is older than the TTL.
var ErrVerificationCodeExpired = errors.New("verification code expired")

// ErrVerificationTooManyAttempts is returned once a code has been guessed wrong
// verificationMaxAttempts times. The code is dead; the user must resend.
var ErrVerificationTooManyAttempts = errors.New("verification attempts exceeded")

// newVerificationCode returns a cryptographically-random 6-digit numeric code
// as a zero-padded string (e.g. "042173").
func newVerificationCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		// rand.Reader failure is catastrophic; fall back to a non-guessable-enough
		// value derived from the same reader via a smaller read.
		b := make([]byte, 3)
		_, _ = rand.Read(b)
		n = big.NewInt(int64(b[0])<<16 | int64(b[1])<<8 | int64(b[2]))
		n.Mod(n, big.NewInt(1000000))
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// RequestEmailVerification regenerates a 6-digit code, stamps the send time, and
// emails it. Enforces a 60-second resend cooldown (Task 5 adds the check).
func (u *UsersService) RequestEmailVerification(ctx context.Context, email string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}

	if !user.EmailVerificationSentAt.IsZero() &&
		time.Since(user.EmailVerificationSentAt) < verificationResendCooldown {
		return ErrVerificationCooldown
	}

	user.EmailVerificationToken = newVerificationCode()
	user.EmailVerificationSentAt = time.Now()
	user.EmailVerificationAttempts = 0 // a new code gets a fresh guess budget
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email,
		"email_verification_token", "email_verification_sent_at", "email_verification_attempts"); err != nil {
		return fmt.Errorf("persist code %s: %w", email, err)
	}

	if err := u.notificator.SendEmailVerification(ctx, email, user.EmailVerificationToken); err != nil {
		return fmt.Errorf("send verification %s: %w", email, err)
	}
	return nil
}

// MarkEmailVerified flips email_verified=true for an already-existing account.
// Trusted path: the caller (Lia) has proven the user owns the address (invite
// accept). No token/code required.
func (u *UsersService) MarkEmailVerified(ctx context.Context, email string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}
	user.EmailVerified = true
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email, "email_verified"); err != nil {
		return fmt.Errorf("mark verified %s: %w", email, err)
	}
	return nil
}

// VerifyEmail marks the account verified when the email/token pair matches.
func (u *UsersService) VerifyEmail(ctx context.Context, email, token string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}
	// ORDER IS LOAD-BEARING: this MUST precede the token comparison. Lockout
	// clears the token, so a later check would fall into the mismatch branch and
	// report ErrVerificationTokenInvalid — telling a locked-out user "wrong code"
	// forever with no hint that resending is the way out.
	if user.EmailVerificationAttempts >= verificationMaxAttempts {
		return ErrVerificationTooManyAttempts
	}

	if token == "" || user.EmailVerificationToken != token {
		user.EmailVerificationAttempts++
		cols := []string{"email_verification_attempts"}
		if user.EmailVerificationAttempts >= verificationMaxAttempts {
			user.EmailVerificationToken = "" // burn the code
			cols = append(cols, "email_verification_token")
		}
		if err := u.repository.UpdateUserBy(ctx, user, repository.Email, cols...); err != nil {
			return fmt.Errorf("persist attempt %s: %w", email, err)
		}
		if user.EmailVerificationAttempts >= verificationMaxAttempts {
			return ErrVerificationTooManyAttempts
		}
		return ErrVerificationTokenInvalid
	}

	if user.EmailVerificationSentAt.IsZero() ||
		time.Since(user.EmailVerificationSentAt) > verificationCodeTTL {
		return ErrVerificationCodeExpired
	}

	user.EmailVerified = true
	user.EmailVerificationToken = ""
	user.EmailVerificationAttempts = 0
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email,
		"email_verified", "email_verification_token", "email_verification_attempts"); err != nil {
		return fmt.Errorf("mark verified %s: %w", email, err)
	}
	return nil
}
