package server

import (
	"context"
	"errors"

	"github.com/gateway-fm/scriptorium/pointer"

	"gateguard/internal/models"
	"gateguard/internal/service"
	proto "gateguard/protocols/gateguard"
)

func (h *GateguardHandlers) ReactToInvitation(ctx context.Context, req *proto.ReactToInvitationRequest) (*proto.ReactToInvitationResponse, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_refCode":       req.GetRefCode(),
		"request_invitee_email": req.GetInvitee(),
		"request_status":        req.GetStatus(),
	})
	h.log.DebugCtx(ctx, "ReactToInvitation handler called")

	err := h.srv.ReactToInvitation(ctx, service.ReactToInvitationIn{
		InviteeEmail: req.GetInvitee(),
		Status:       models.InvitationStatus(req.GetStatus()),
		RefCode:      req.GetRefCode(),
	})
	if err != nil {
		h.log.ErrorCtx(ctx, err, "failed to react to invitation")

		var code proto.ReactToInvitationError

		switch {
		case errors.Is(err, service.ErrUserNotFound), errors.Is(err, service.ErrUserDeleted):
			code = proto.ReactToInvitationError_RtiValidationError
		case errors.Is(err, service.ErrOrganizationNotFound), errors.Is(err, service.ErrOrganizationNotValid):
			code = proto.ReactToInvitationError_RtiOrganizationConflict
		case errors.Is(err, service.ErrInviteStatusNotValid):
			code = proto.ReactToInvitationError_RtiInviteStatusConflict
		case errors.Is(err, service.ErrUserMismatch):
			code = proto.ReactToInvitationError_RtiInviteUserMismatch
		default:
			return nil, err
		}

		return &proto.ReactToInvitationResponse{
			Error: pointer.Ref(code),
		}, nil
	}

	return &proto.ReactToInvitationResponse{
		Success: &proto.Empty{},
	}, nil
}
