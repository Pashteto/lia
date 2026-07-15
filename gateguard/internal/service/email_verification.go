package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"

	"gateguard/internal/models"
	"gateguard/internal/repository"
)

// ErrVerificationTokenInvalid is returned when an email/token pair does not match.
var ErrVerificationTokenInvalid = errors.New("verification token invalid")

// newVerificationToken returns a random URL-safe token.
func newVerificationToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

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

// sendVerificationStub is a NON-PRODUCTION stub: it logs the verification link
// instead of emailing it. Replace with the SMTP notificator before real prod.
func (u *UsersService) sendVerificationStub(ctx context.Context, user *models.User) {
	u.log.WarnCtx(ctx, fmt.Sprintf(
		"[STUB] email verification not sent (no mailer wired). email=%s token=%s",
		user.Email, user.EmailVerificationToken))
}

// RequestEmailVerification regenerates + persists a verification token and
// (stub) "sends" it.
func (u *UsersService) RequestEmailVerification(ctx context.Context, email string) error {
	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		return fmt.Errorf("lookup user %s: %w", email, err)
	}
	user.EmailVerificationToken = newVerificationToken()
	if err := u.repository.UpdateUserBy(ctx, user, repository.Email, "email_verification_token"); err != nil {
		return fmt.Errorf("persist token %s: %w", email, err)
	}
	u.sendVerificationStub(ctx, user)
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
