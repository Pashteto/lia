package server

import (
	"context"
	"errors"

	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"

	"gateguard/internal/service"
	proto "gateguard/protocols/gateguard"
)

func (h *GateguardHandlers) InviteUserByEmail(ctx context.Context, req *proto.InviteUserByEmailRequest) (*proto.InviteUserByEmailResponse, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_email":             req.Email,
		"request_inviter_uuid":      uuid.FromBytesOrNil(req.InviterUuid).String(),
		"request_inviter_email":     req.InviterEmail,
		"request_organization_uuid": uuid.FromBytesOrNil(req.OrganizationUuid).String(),
		"request_invitee_role":      req.GetInviteeRole(),
	})

	h.log.DebugCtx(ctx, "InviteUserByEmail handler called")
	inviterUUID, err := uuid.FromBytes(req.InviterUuid)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse inviter UUID")
		return &proto.InviteUserByEmailResponse{
			Error: pointer.Ref(proto.InviteUserByEmailError_IueValidationError),
		}, nil
	}

	var organizationUUID *uuid.UUID
	if req.OrganizationUuid != nil {
		var orgUUID uuid.UUID
		orgUUID, err = uuid.FromBytes(req.OrganizationUuid)
		if err != nil {
			h.log.ErrorCtx(ctx, err, "Failed to parse organization UUID")
			return &proto.InviteUserByEmailResponse{
				Error: pointer.Ref(proto.InviteUserByEmailError_IueValidationError),
			}, nil
		}
		organizationUUID = &orgUUID
	}

	err = h.srv.InviteUserByEmail(ctx, service.InviteUserByEmailIn{
		InviterUUID:      inviterUUID,
		InviterEmail:     req.InviterEmail,
		InviteeEmail:     req.Email,
		OrganizationUUID: organizationUUID,
		InviteeRole:      req.InviteeRole,
	})

	if err != nil {
		var code proto.InviteUserByEmailError

		switch {
		case errors.Is(err, service.ErrOrganizationNotFound):
			code = proto.InviteUserByEmailError_IueOrganizationNotFound
		case errors.Is(err, service.ErrInviteNotAllowed), errors.Is(err, service.ErrOrganizationNotValid):
			code = proto.InviteUserByEmailError_IueOrganizationInviteNotAllowed
		case errors.Is(err, service.ErrSuchUserAlreadyExists):
			code = proto.InviteUserByEmailError_IueUserAlreadyExists
		case errors.Is(err, service.ErrInvitesRateLimitReached):
			code = proto.InviteUserByEmailError_IueInvitesRateLimitReached
		default:
			h.log.ErrorCtx(ctx, err, "Failed to invite user by email")
			return nil, err
		}

		h.log.WarnCtx(ctx, "Invite user by email failed")
		return &proto.InviteUserByEmailResponse{
			Error: pointer.Ref(code),
		}, nil
	}

	return &proto.InviteUserByEmailResponse{
		Success: &proto.Empty{},
	}, nil
}
