package handler_test

import (
	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"

	"gateguard/internal/service"
	"gateguard/protocols/gateguard"
)

const (
	inviterEmail = "inviter@example.com"
	inviteeEmail = "invitee@example.com"
)

func (s *ServerSuite) TestGateguardHandlers_InviteUserByEmail() {
	s.Run("invite_user_success", func() {
		inviterUUID := uuid.Must(uuid.NewV4())

		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().InviteUserByEmail(mock.Anything, service.InviteUserByEmailIn{
			InviterUUID:      inviterUUID,
			InviterEmail:     inviterEmail,
			InviteeEmail:     inviteeEmail,
			OrganizationUUID: &orgUUID,
		}).Return(nil).Once()

		req := &gateguard.InviteUserByEmailRequest{
			InviterUuid:      inviterUUID.Bytes(),
			InviterEmail:     inviterEmail,
			Email:            inviteeEmail,
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.InviteUserByEmail(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().NotNil(resp.Success)
		s.Require().Nil(resp.Error)
	})

	s.Run("service_error_organization_not_found", func() {
		inviterUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().InviteUserByEmail(mock.Anything, service.InviteUserByEmailIn{
			InviterUUID:      inviterUUID,
			InviterEmail:     inviterEmail,
			InviteeEmail:     inviteeEmail,
			OrganizationUUID: &orgUUID,
		}).Return(service.ErrOrganizationNotFound).Once()

		req := &gateguard.InviteUserByEmailRequest{
			InviterUuid:      inviterUUID.Bytes(),
			InviterEmail:     inviterEmail,
			Email:            inviteeEmail,
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.InviteUserByEmail(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.InviteUserByEmailError_IueOrganizationNotFound), resp.Error)
	})

	s.Run("service_error_invite_not_allowed", func() {
		inviterUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().InviteUserByEmail(mock.Anything, service.InviteUserByEmailIn{
			InviterUUID:      inviterUUID,
			InviterEmail:     inviterEmail,
			InviteeEmail:     inviteeEmail,
			OrganizationUUID: &orgUUID,
		}).Return(service.ErrInviteNotAllowed).Once()

		req := &gateguard.InviteUserByEmailRequest{
			InviterUuid:      inviterUUID.Bytes(),
			InviterEmail:     inviterEmail,
			Email:            inviteeEmail,
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.InviteUserByEmail(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.InviteUserByEmailError_IueOrganizationInviteNotAllowed), resp.Error)
	})

	s.Run("service_internal_error", func() {
		inviterUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.usersServiceMock.EXPECT().InviteUserByEmail(mock.Anything, service.InviteUserByEmailIn{
			InviterUUID:      inviterUUID,
			InviterEmail:     inviterEmail,
			InviteeEmail:     inviteeEmail,
			OrganizationUUID: &orgUUID,
		}).Return(errInternal).Once()

		req := &gateguard.InviteUserByEmailRequest{
			InviterUuid:      inviterUUID.Bytes(),
			InviterEmail:     inviterEmail,
			Email:            inviteeEmail,
			OrganizationUuid: orgUUID.Bytes(),
		}

		resp, err := s.handlers.InviteUserByEmail(s.ctx, req)

		s.Require().ErrorIs(err, errInternal)
		s.Require().Nil(resp)
	})
}
