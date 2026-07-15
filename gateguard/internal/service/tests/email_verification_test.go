package service_test

import (
	"github.com/stretchr/testify/mock"

	"gateguard/internal/models"
	"gateguard/internal/repository"
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
