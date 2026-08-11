package service_test

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"gateguard/internal/models"
	"gateguard/internal/pkg/password"
	"gateguard/internal/repository"
	"gateguard/internal/service"
)

const testLoginPassword = "correct horse battery"

// seedUser returns a credentialed active user with the given lockout state.
func (s *UseCaseSuite) seedUser(attempts int, lockedUntil time.Time) *models.User {
	hash, err := password.Hash(testLoginPassword)
	s.Require().NoError(err)
	return &models.User{
		Email:               "login@example.com",
		PasswordHash:        hash,
		Status:              models.UserActive,
		FailedLoginAttempts: attempts,
		LoginLockedUntil:    lockedUntil,
	}
}

// expectGetUser makes the mocked GetUser return a copy of the seeded user.
func (s *UseCaseSuite) expectGetUser(seed *models.User) {
	s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.Email).
		Return(nil).
		Run(func(_ context.Context, model *models.User, _ repository.UserGetter) {
			*model = *seed
		}).Once()
}

func (s *UseCaseSuite) Test_SignInWithPassword_WrongPasswordIncrements() {
	s.expectGetUser(s.seedUser(0, time.Time{}))

	var got *models.User
	s.repo.EXPECT().UpdateUserBy(mock.Anything, mock.Anything, repository.Email, "failed_login_attempts").
		Return(nil).
		Run(func(_ context.Context, model *models.User, _ repository.UserGetter, _ ...string) {
			got = model
		}).Once()

	_, _, err := s.service.SignInWithPassword(s.ctx, "login@example.com", "wrong")
	s.Require().ErrorIs(err, service.ErrInvalidCredentials)
	s.Equal(1, got.FailedLoginAttempts)
	s.True(got.LoginLockedUntil.IsZero(), "must not lock on the first failure")
}

func (s *UseCaseSuite) Test_SignInWithPassword_FifthFailureLocks() {
	s.expectGetUser(s.seedUser(4, time.Time{})) // one away from the cap

	var got *models.User
	s.repo.EXPECT().UpdateUserBy(mock.Anything, mock.Anything, repository.Email,
		"failed_login_attempts", "login_locked_until").
		Return(nil).
		Run(func(_ context.Context, model *models.User, _ repository.UserGetter, _ ...string) {
			got = model
		}).Once()

	_, _, err := s.service.SignInWithPassword(s.ctx, "login@example.com", "wrong")
	s.Require().ErrorIs(err, service.ErrAccountLocked)
	s.Equal(5, got.FailedLoginAttempts)
	s.True(got.LoginLockedUntil.After(time.Now()), "5th failure must set a future lock")
}

func (s *UseCaseSuite) Test_SignInWithPassword_LockedRejectsEvenWithCorrectPassword() {
	// Locked 10 minutes into the future; no UpdateUserBy expected (rejected early).
	s.expectGetUser(s.seedUser(5, time.Now().Add(10*time.Minute)))

	_, _, err := s.service.SignInWithPassword(s.ctx, "login@example.com", testLoginPassword)
	s.Require().ErrorIs(err, service.ErrAccountLocked,
		"a locked account is rejected before the password is even checked")
}

func (s *UseCaseSuite) Test_SignInWithPassword_ExpiredLockResetsOnNextFailure() {
	// Was locked, but the window has passed: the next wrong attempt starts fresh.
	s.expectGetUser(s.seedUser(5, time.Now().Add(-time.Minute)))

	var got *models.User
	s.repo.EXPECT().UpdateUserBy(mock.Anything, mock.Anything, repository.Email,
		"failed_login_attempts", "login_locked_until").
		Return(nil).
		Run(func(_ context.Context, model *models.User, _ repository.UserGetter, _ ...string) {
			got = model
		}).Once()

	_, _, err := s.service.SignInWithPassword(s.ctx, "login@example.com", "wrong")
	s.Require().ErrorIs(err, service.ErrInvalidCredentials)
	s.Equal(1, got.FailedLoginAttempts, "counter resets to 1 after the window elapses")
	s.True(got.LoginLockedUntil.IsZero(), "expired lock is cleared")
}

func (s *UseCaseSuite) Test_SignInWithPassword_SuccessResetsCounter() {
	s.expectGetUser(s.seedUser(3, time.Time{})) // some prior failures

	var got *models.User
	s.repo.EXPECT().UpdateUserBy(mock.Anything, mock.Anything, repository.Email,
		"failed_login_attempts", "login_locked_until").
		Return(nil).
		Run(func(_ context.Context, model *models.User, _ repository.UserGetter, _ ...string) {
			got = model
		}).Once()
	s.sessions.EXPECT().Create(mock.Anything, mock.Anything).Return([]byte("jwt"), nil).Once()

	token, _, err := s.service.SignInWithPassword(s.ctx, "login@example.com", testLoginPassword)
	s.Require().NoError(err)
	s.Equal([]byte("jwt"), token)
	s.Equal(0, got.FailedLoginAttempts, "successful login clears the failure counter")
	s.True(got.LoginLockedUntil.IsZero())
}
