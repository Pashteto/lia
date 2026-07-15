package service_test

import (
	"context"
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
			"email_verification_token", "email_verification_sent_at").
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
			u.EmailVerificationSentAt = time.Now().Add(-20 * time.Minute) // older than 15m
		}).
		Return(nil).Once()

	err := s.service.VerifyEmail(s.ctx, email, code)
	s.Require().ErrorIs(err, service.ErrVerificationCodeExpired)
}
