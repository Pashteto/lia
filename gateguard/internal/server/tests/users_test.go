package handler_test

import (
	"github.com/stretchr/testify/mock"

	"gateguard/protocols/gateguard"
)

func (s *ServerSuite) TestGateguardHandlers_DeleteUser() {
	s.Run("delete_user_success", func() {
		s.usersServiceMock.EXPECT().DeleteUser(mock.Anything, []byte(bearerToken)).Return(nil).Once()

		_, err := s.handlers.DeleteUser(s.ctx, &gateguard.TokenRequest{Token: []byte(bearerToken)})

		s.Require().NoError(err)
	})

	s.Run("service_error", func() {
		s.usersServiceMock.EXPECT().DeleteUser(mock.Anything, []byte(bearerToken)).Return(errInternal).Once()

		_, err := s.handlers.DeleteUser(s.ctx, &gateguard.TokenRequest{Token: []byte(bearerToken)})

		s.Require().ErrorIs(err, errInternal)
	})
}
