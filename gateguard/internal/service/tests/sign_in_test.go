package service_test

import (
	sessions "github.com/andskur/gatekeeper"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"

	"gateguard/internal/models"
	"gateguard/internal/pkg/converters/ipconv"
	"gateguard/internal/pkg/tests/fake"
	"gateguard/internal/repository"
)

func (s *UseCaseSuite) Test_SignInOAuth_Success() {
	s.Run("sign_in", func() {
		tokenExpected := sessions.Token("some_token")
		someUuid, err := uuid.NewV7()
		s.Require().NoError(err)

		someIP, err := ipconv.IpToUint32(fake.GenerateRandomIP())
		s.Require().NoError(err)

		user := &models.User{
			UUID:   someUuid,
			Status: models.UserActive,
			IP:     someIP,
		}

		s.repo.EXPECT().GetUser(mock.Anything, user, repository.Email).Return(nil).Once()
		s.repo.EXPECT().UpdateUserBy(mock.Anything, user, repository.Email, ipKey).Return(nil).Once()
		s.sessions.EXPECT().Create(mock.Anything, user.ToJWT()).Return(tokenExpected, nil)

		tokenGot, user, err := s.service.SignInOAuth(s.ctx, user)
		s.Require().NoError(err)
		s.Require().NotNil(user)
		s.Require().EqualValues(tokenExpected, tokenGot, "tokens must be equal")
	})

	s.Run("register", func() {
		tokenExpected := sessions.Token("some_token")
		someUuid, err := uuid.NewV7()
		s.Require().NoError(err)

		someIP, err := ipconv.IpToUint32(fake.GenerateRandomIP())
		s.Require().NoError(err)

		user := &models.User{
			UUID:   someUuid,
			Status: models.UserActive,
			IP:     someIP,
		}

		s.repo.EXPECT().GetUser(mock.Anything, user, repository.Email).Return(repository.ErrUserNotFound).Once()
		s.repo.EXPECT().CreateUser(mock.Anything, user).Return(nil).Once()
		s.repo.EXPECT().UpdateUserBy(mock.Anything, user, repository.Email, ipKey).Return(nil).Once()
		s.sessions.EXPECT().Create(mock.Anything, user.ToJWT()).Return(tokenExpected, nil)

		tokenGot, user, err := s.service.SignInOAuth(s.ctx, user)
		s.Require().NoError(err)
		s.Require().NotNil(user)
		s.Require().EqualValues(tokenExpected, tokenGot, "tokens must be equal")
	})
}
