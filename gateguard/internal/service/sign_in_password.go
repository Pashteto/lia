package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gateguard/internal/models"
	"gateguard/internal/pkg/password"
	"gateguard/internal/repository"
)

var (
	// ErrUserAlreadyExists is returned when registering an email that already has a password.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrInvalidCredentials is returned for an unknown email or a wrong password.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// SignUpWithPassword creates a credentialed account and returns a session JWT.
// If the email already exists without a password (e.g. a demo-login user), the
// password is attached to that account instead of erroring.
func (u *UsersService) SignUpWithPassword(ctx context.Context, email, name, plain string) ([]byte, *models.User, error) {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"user_email": email})

	existing := &models.User{Email: email}
	err := u.repository.GetUser(ctx, existing, repository.Email)
	if err == nil && existing.PasswordHash != "" {
		return nil, nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("lookup user %s: %w", email, err)
	}

	hash, hErr := password.Hash(plain)
	if hErr != nil {
		return nil, nil, fmt.Errorf("hash password: %w", hErr)
	}

	code := newVerificationCode()
	user := &models.User{
		Email:                   email,
		Name:                    name,
		Status:                  models.UserActive,
		Role:                    models.UserRoleCommon,
		PasswordHash:            hash,
		EmailVerified:           false,
		EmailVerificationToken:  code,
		EmailVerificationSentAt: time.Now(),
	}

	if errors.Is(err, repository.ErrUserNotFound) {
		if cErr := u.repository.CreateUser(ctx, user); cErr != nil {
			u.log.ErrorCtx(ctx, cErr, fmt.Sprintf("create user %s", email))
			return nil, nil, fmt.Errorf("create user %s: %w", email, cErr)
		}
		user.CreatedOrRestored = true
	} else {
		// Pre-existing passwordless account: attach credentials to it.
		user.UUID = existing.UUID
		if uErr := u.repository.UpdateUserBy(ctx, user, repository.Email,
			"password_hash", "email_verification_token", "email_verification_sent_at", "email_verified", "name"); uErr != nil {
			u.log.ErrorCtx(ctx, uErr, fmt.Sprintf("attach credentials %s", email))
			return nil, nil, fmt.Errorf("attach credentials %s: %w", email, uErr)
		}
	}

	if sErr := u.notificator.SendEmailVerification(ctx, email, code); sErr != nil {
		// Do not fail signup if the email send fails; the user can request a resend.
		u.log.ErrorCtx(ctx, sErr, fmt.Sprintf("send verification email %s", email))
	}

	jwt, jErr := u.createJWT(user)
	if jErr != nil {
		u.log.ErrorCtx(ctx, jErr, "create user session token")
		return nil, nil, fmt.Errorf("create session token: %w", jErr)
	}
	return jwt, user, nil
}

// SignInWithPassword verifies the password for an email and returns a session JWT.
func (u *UsersService) SignInWithPassword(ctx context.Context, email, plain string) ([]byte, *models.User, error) {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"user_email": email})

	user := &models.User{Email: email}
	if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("lookup user %s: %w", email, err)
	}

	if user.PasswordHash == "" || password.Compare(user.PasswordHash, plain) != nil {
		return nil, nil, ErrInvalidCredentials
	}

	jwt, err := u.createJWT(user)
	if err != nil {
		u.log.ErrorCtx(ctx, err, "create user session token")
		return nil, nil, fmt.Errorf("create session token: %w", err)
	}
	return jwt, user, nil
}
