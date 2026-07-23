package service_test

import (
	"context"
	"errors"
	"time"

	"github.com/stretchr/testify/mock"

	"gateguard/internal/models"
	"gateguard/internal/repository"
	"gateguard/internal/service"
)

func (s *UseCaseSuite) Test_RequestEmailVerification_SendsCode() {
	email := "user@example.com"

	// GetUser loads a user with no prior send (zero sent-at → no cooldown).
	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Return(nil).Once()

	// Persist must include both the token and the sent-at columns.
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything, mock.Anything, repository.Email,
			"email_verification_token", "email_verification_sent_at", "email_verification_attempts").
		Return(nil).Once()

	// A real 6-digit code must be sent to the address.
	s.nMock.EXPECT().
		SendEmailVerification(mock.Anything, email, mock.MatchedBy(func(code string) bool {
			return len(code) == 6
		})).
		Return(nil).Once()

	err := s.service.RequestEmailVerification(s.ctx, email)
	s.Require().NoError(err)
}

func (s *UseCaseSuite) Test_RequestEmailVerification_Cooldown() {
	email := "user@example.com"

	// GetUser returns a user whose last send was 5 seconds ago → cooldown active.
	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationSentAt = time.Now().Add(-5 * time.Second)
		}).
		Return(nil).Once()

	err := s.service.RequestEmailVerification(s.ctx, email)
	s.Require().ErrorIs(err, service.ErrVerificationCooldown)
}

func (s *UseCaseSuite) Test_VerifyEmail_Expired() {
	email := "user@example.com"
	code := "123456"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = code
			u.EmailVerificationSentAt = time.Now().Add(-25 * time.Hour) // older than the 24h TTL
		}).
		Return(nil).Once()

	err := s.service.VerifyEmail(s.ctx, email, code)
	s.Require().ErrorIs(err, service.ErrVerificationCodeExpired)
}

func (s *UseCaseSuite) Test_VerifyEmail_LockoutAfterFiveWrongAttempts() {
	email := "user@example.com"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = "123456"
			u.EmailVerificationAttempts = 4 // one guess left
			u.EmailVerificationSentAt = time.Now()
		}).
		Return(nil).Once()

	// The 5th wrong guess trips the cap: attempts hits 5 AND the code is burned.
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool {
				return u.EmailVerificationAttempts == 5 && u.EmailVerificationToken == ""
			}),
			repository.Email,
			"email_verification_attempts", "email_verification_token").
		Return(nil).Once()

	err := s.service.VerifyEmail(s.ctx, email, "000000")
	s.Require().ErrorIs(err, service.ErrVerificationTooManyAttempts)
}

// PINS THE CHECK ORDER. Fails if the attempts check is placed after the token
// comparison: lockout clears the token, so a later check would fall into the
// mismatch branch and report ErrVerificationTokenInvalid — stranding the user
// with "wrong code" forever and no hint that resending is the way out.
func (s *UseCaseSuite) Test_VerifyEmail_LockedOutRejectsEvenCorrectCode() {
	email := "user@example.com"
	code := "123456"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = code
			u.EmailVerificationAttempts = 5 // already locked out
			u.EmailVerificationSentAt = time.Now()
		}).
		Return(nil).Once()

	// No UpdateUserBy expectation: the guard returns before any write. If the
	// implementation writes here, the mock fails the test.
	err := s.service.VerifyEmail(s.ctx, email, code) // CORRECT code
	s.Require().ErrorIs(err, service.ErrVerificationTooManyAttempts)
}

func (s *UseCaseSuite) Test_VerifyEmail_WrongCodeIncrementsAttempts() {
	email := "user@example.com"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = "123456"
			u.EmailVerificationSentAt = time.Now()
		}).
		Return(nil).Once()

	// Below the cap: increment only, token NOT burned.
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool {
				return u.EmailVerificationAttempts == 1 && u.EmailVerificationToken == "123456"
			}),
			repository.Email,
			"email_verification_attempts").
		Return(nil).Once()

	err := s.service.VerifyEmail(s.ctx, email, "999999")
	s.Require().ErrorIs(err, service.ErrVerificationTokenInvalid)
}

func (s *UseCaseSuite) Test_VerifyEmail_WithinTTLSucceedsAndResetsAttempts() {
	email := "user@example.com"
	code := "123456"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationToken = code
			u.EmailVerificationAttempts = 3
			u.EmailVerificationSentAt = time.Now().Add(-23 * time.Hour) // just inside 24h
		}).
		Return(nil).Once()

	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool {
				return u.EmailVerified && u.EmailVerificationAttempts == 0 && u.EmailVerificationToken == ""
			}),
			repository.Email,
			"email_verified", "email_verification_token", "email_verification_attempts").
		Return(nil).Once()

	s.Require().NoError(s.service.VerifyEmail(s.ctx, email, code))
}

// MarkEmailVerified is the trusted path: no code, just flip email_verified=true
// on an existing account. Persist must target only the email_verified column.
func (s *UseCaseSuite) Test_MarkEmailVerified_SetsFlag() {
	email := "user@example.com"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Return(nil).Once()

	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool { return u.EmailVerified }),
			repository.Email,
			"email_verified").
		Return(nil).Once()

	s.Require().NoError(s.service.MarkEmailVerified(s.ctx, email))
}

// An unknown address must surface the lookup error, not silently succeed.
func (s *UseCaseSuite) Test_MarkEmailVerified_UnknownEmail() {
	email := "nobody@example.com"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Return(errors.New("no rows")).Once()

	err := s.service.MarkEmailVerified(s.ctx, email)
	s.Require().Error(err)
}

func (s *UseCaseSuite) Test_RequestEmailVerification_ResetsAttempts() {
	email := "user@example.com"

	s.repo.EXPECT().
		GetUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == email }), repository.Email).
		Run(func(_ context.Context, u *models.User, _ repository.UserGetter) {
			u.EmailVerificationAttempts = 5                              // locked out
			u.EmailVerificationSentAt = time.Now().Add(-2 * time.Minute) // past the 60s cooldown
		}).
		Return(nil).Once()

	// A resend must hand back a fresh guess budget, or lockout is permanent.
	s.repo.EXPECT().
		UpdateUserBy(mock.Anything,
			mock.MatchedBy(func(u *models.User) bool { return u.EmailVerificationAttempts == 0 }),
			repository.Email,
			"email_verification_token", "email_verification_sent_at", "email_verification_attempts").
		Return(nil).Once()

	s.nMock.EXPECT().
		SendEmailVerification(mock.Anything, email, mock.MatchedBy(func(c string) bool { return len(c) == 6 })).
		Return(nil).Once()

	s.Require().NoError(s.service.RequestEmailVerification(s.ctx, email))
}
