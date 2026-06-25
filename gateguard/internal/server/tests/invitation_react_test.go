package handler_test

import (
	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/stretchr/testify/mock"

	"gateguard/internal/models"
	"gateguard/internal/service"
	"gateguard/protocols/gateguard"
)

const refCode = "test-ref-code"

func (s *ServerSuite) TestGateguardHandlers_ReactToInvitation() {
	s.Run("react_to_invitation_success", func() {
		status := models.Accepted

		s.usersServiceMock.EXPECT().ReactToInvitation(mock.Anything, service.ReactToInvitationIn{
			InviteeEmail: inviteeEmail,
			Status:       status,
			RefCode:      refCode,
		}).Return(nil).Once()

		req := &gateguard.ReactToInvitationRequest{
			Invitee: inviteeEmail,
			Status:  gateguard.InvitationStatus(status),
			RefCode: refCode,
		}

		resp, err := s.handlers.ReactToInvitation(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().NotNil(resp.Success)
		s.Require().Nil(resp.Error)
	})

	s.Run("service_error_user_not_found", func() {
		status := models.Accepted

		s.usersServiceMock.EXPECT().ReactToInvitation(mock.Anything, service.ReactToInvitationIn{
			InviteeEmail: inviteeEmail,
			Status:       status,
			RefCode:      refCode,
		}).Return(service.ErrUserNotFound).Once()

		req := &gateguard.ReactToInvitationRequest{
			Invitee: inviteeEmail,
			Status:  gateguard.InvitationStatus(status),
			RefCode: refCode,
		}

		resp, err := s.handlers.ReactToInvitation(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.ReactToInvitationError_RtiValidationError), resp.Error)
	})

	s.Run("service_error_user_deleted", func() {
		status := models.Accepted

		s.usersServiceMock.EXPECT().ReactToInvitation(mock.Anything, service.ReactToInvitationIn{
			InviteeEmail: inviteeEmail,
			Status:       status,
			RefCode:      refCode,
		}).Return(service.ErrUserDeleted).Once()

		req := &gateguard.ReactToInvitationRequest{
			Invitee: inviteeEmail,
			Status:  gateguard.InvitationStatus(status),
			RefCode: refCode,
		}

		resp, err := s.handlers.ReactToInvitation(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.ReactToInvitationError_RtiValidationError), resp.Error)
	})

	s.Run("service_error_organization_not_found", func() {
		status := models.Accepted

		s.usersServiceMock.EXPECT().ReactToInvitation(mock.Anything, service.ReactToInvitationIn{
			InviteeEmail: inviteeEmail,
			Status:       status,
			RefCode:      refCode,
		}).Return(service.ErrOrganizationNotFound).Once()

		req := &gateguard.ReactToInvitationRequest{
			Invitee: inviteeEmail,
			Status:  gateguard.InvitationStatus(status),
			RefCode: refCode,
		}

		resp, err := s.handlers.ReactToInvitation(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.ReactToInvitationError_RtiOrganizationConflict), resp.Error)
	})

	s.Run("service_error_organization_deleted", func() {
		status := models.Accepted

		s.usersServiceMock.EXPECT().ReactToInvitation(mock.Anything, service.ReactToInvitationIn{
			InviteeEmail: inviteeEmail,
			Status:       status,
			RefCode:      refCode,
		}).Return(service.ErrOrganizationNotValid).Once()

		req := &gateguard.ReactToInvitationRequest{
			Invitee: inviteeEmail,
			Status:  gateguard.InvitationStatus(status),
			RefCode: refCode,
		}

		resp, err := s.handlers.ReactToInvitation(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.ReactToInvitationError_RtiOrganizationConflict), resp.Error)
	})

	s.Run("service_error_invite_status_not_valid", func() {
		status := models.Accepted

		s.usersServiceMock.EXPECT().ReactToInvitation(mock.Anything, service.ReactToInvitationIn{
			InviteeEmail: inviteeEmail,
			Status:       status,
			RefCode:      refCode,
		}).Return(service.ErrInviteStatusNotValid).Once()

		req := &gateguard.ReactToInvitationRequest{
			Invitee: inviteeEmail,
			Status:  gateguard.InvitationStatus(status),
			RefCode: refCode,
		}

		resp, err := s.handlers.ReactToInvitation(s.ctx, req)

		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(pointer.Ref(gateguard.ReactToInvitationError_RtiInviteStatusConflict), resp.Error)
	})

	s.Run("service_internal_error", func() {
		status := models.Accepted

		s.usersServiceMock.EXPECT().ReactToInvitation(mock.Anything, service.ReactToInvitationIn{
			InviteeEmail: inviteeEmail,
			Status:       status,
			RefCode:      refCode,
		}).Return(errInternal).Once()

		req := &gateguard.ReactToInvitationRequest{
			Invitee: inviteeEmail,
			Status:  gateguard.InvitationStatus(status),
			RefCode: refCode,
		}

		resp, err := s.handlers.ReactToInvitation(s.ctx, req)

		s.Require().ErrorIs(err, errInternal)
		s.Require().Nil(resp)
	})
}
