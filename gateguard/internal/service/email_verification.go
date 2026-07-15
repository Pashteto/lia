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
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email,
		"email_verification_token", "email_verification_sent_at"); err != nil {
		return fmt.Errorf("persist code %s: %w", email, err)
	}

	if err := u.notificator.SendEmailVerification(ctx, email, user.EmailVerificationToken); err != nil {
		return fmt.Errorf("send verification %s: %w", email, err)
	}
	return nil
}

// VerifyEmail marks the account verified when the email/token pair matches.
func (u *UsersService) VerifyEmail(ctx context.Context, email, token string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}
	if token == "" || user.EmailVerificationToken != token {
		return ErrVerificationTokenInvalid
	}
	user.EmailVerified = true
	user.EmailVerificationToken = ""
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email,
		"email_verified", "email_verification_token"); err != nil {
		return fmt.Errorf("mark verified %s: %w", email, err)
	}
	return nil
}
