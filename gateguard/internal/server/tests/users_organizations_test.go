package handler_test

import (
	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"

	"gateguard/internal/service"
	"gateguard/protocols/gateguard"
)

func (s *ServerSuite) TestGateguardHandlers_AddOrganizationToUser() {
	s.Run("add_organization_success", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().AddOrganizationToUser(mock.Anything, userUUID, orgUUID).Return(nil).Once()

		req := &gateguard.AddOrganizationRequest{
			UserUuid:         userUUID.Bytes(),
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.AddOrganizationToUser(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().NotNil(resp.Success)
		s.Require().Nil(resp.Error)
	})

	s.Run("service_error_user_not_found", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().AddOrganizationToUser(mock.Anything, userUUID, orgUUID).Return(service.ErrUserNotFound).Once()

		req := &gateguard.AddOrganizationRequest{
			UserUuid:         userUUID.Bytes(),
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.AddOrganizationToUser(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.AddOrganizationError_AoeUserNotFound), resp.Error)
	})

	s.Run("service_error_organization_already_exists", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().AddOrganizationToUser(mock.Anything, userUUID, orgUUID).Return(service.ErrOrganizationAlreadyExists).Once()

		req := &gateguard.AddOrganizationRequest{
			UserUuid:         userUUID.Bytes(),
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.AddOrganizationToUser(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.AddOrganizationError_AoeOrganizationAlreadyExists), resp.Error)
	})

	s.Run("service_internal_error", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().AddOrganizationToUser(mock.Anything, userUUID, orgUUID).Return(errInternal).Once()

		req := &gateguard.AddOrganizationRequest{
			UserUuid:         userUUID.Bytes(),
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.AddOrganizationToUser(s.ctx, req)

		s.Require().ErrorIs(err, errInternal)
		s.Require().Nil(resp)
	})
}

func (s *ServerSuite) TestGateguardHandlers_RemoveOrganizationFromUser() {
	s.Run("remove_organization_success", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().RemoveOrganizationFromUser(mock.Anything, userUUID, orgUUID).Return(nil).Once()

		req := &gateguard.RemoveOrganizationRequest{
			UserUuid:         userUUID.Bytes(),
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.RemoveOrganizationFromUser(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().NotNil(resp.Success)
		s.Require().Nil(resp.Error)
	})

	s.Run("service_error_user_not_found", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().RemoveOrganizationFromUser(mock.Anything, userUUID, orgUUID).Return(service.ErrUserNotFound).Once()

		req := &gateguard.RemoveOrganizationRequest{
			UserUuid:         userUUID.Bytes(),
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.RemoveOrganizationFromUser(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.RemoveOrganizationError_RoeUserNotFound), resp.Error)
	})

	s.Run("service_error_organization_not_found", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().RemoveOrganizationFromUser(mock.Anything, userUUID, orgUUID).Return(service.ErrOrganizationNotFound).Once()

		req := &gateguard.RemoveOrganizationRequest{
			UserUuid:         userUUID.Bytes(),
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.RemoveOrganizationFromUser(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.RemoveOrganizationError_RoeOrganizationNotFound), resp.Error)
	})

	s.Run("service_internal_error", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().RemoveOrganizationFromUser(mock.Anything, userUUID, orgUUID).Return(errInternal).Once()

		req := &gateguard.RemoveOrganizationRequest{
			UserUuid:         userUUID.Bytes(),
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.RemoveOrganizationFromUser(s.ctx, req)

		s.Require().ErrorIs(err, errInternal)
		s.Require().Nil(resp)
	})
}
