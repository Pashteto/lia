package server

import (
	"context"
	"errors"

	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"

	"gateguard/internal/service"
	proto "gateguard/protocols/gateguard"
)

func (h *GateguardHandlers) AddOrganizationToUser(ctx context.Context, req *proto.AddOrganizationRequest) (*proto.AddOrganizationResponse, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_user_uuid":         string(req.GetUserUuid()),
		"request_organization_uuid": string(req.GetOrganizationUuid()),
	})
	h.log.DebugCtx(ctx, "AddOrganizationToUser handler called")

	userUUID, err := uuid.FromBytes(req.GetUserUuid())
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse user UUID")
		return &proto.AddOrganizationResponse{
			Error: pointer.Ref(proto.AddOrganizationError_AoeValidationError),
		}, nil
	}

	orgUUID, err := uuid.FromBytes(req.GetOrganizationUuid())
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse organization UUID")
		return &proto.AddOrganizationResponse{
			Error: pointer.Ref(proto.AddOrganizationError_AoeValidationError),
		}, nil
	}

	err = h.srv.AddOrganizationToUser(ctx, userUUID, orgUUID)
	if err != nil {
		var code proto.AddOrganizationError
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			code = proto.AddOrganizationError_AoeUserNotFound
		case errors.Is(err, service.ErrOrganizationAlreadyExists):
			code = proto.AddOrganizationError_AoeOrganizationAlreadyExists
		default:
			h.log.ErrorCtx(ctx, err, "Failed to add organization to user")
			return nil, err
		}
		h.log.WarnCtx(ctx, "Add organization to user failed")
		return &proto.AddOrganizationResponse{
			Error: pointer.Ref(code),
		}, nil
	}

	return &proto.AddOrganizationResponse{
		Success: &proto.Empty{},
	}, nil
}

func (h *GateguardHandlers) RemoveOrganizationFromUser(ctx context.Context, req *proto.RemoveOrganizationRequest) (*proto.RemoveOrganizationResponse, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_user_uuid":         string(req.GetUserUuid()),
		"request_organization_uuid": string(req.GetOrganizationUuid()),
	})
	h.log.DebugCtx(ctx, "RemoveOrganizationFromUser handler called")

	userUUID, err := uuid.FromBytes(req.GetUserUuid())
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse user UUID")
		return &proto.RemoveOrganizationResponse{
			Error: pointer.Ref(proto.RemoveOrganizationError_RoeValidationError),
		}, nil
	}

	orgUUID, err := uuid.FromBytes(req.GetOrganizationUuid())
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse organization UUID")
		return &proto.RemoveOrganizationResponse{
			Error: pointer.Ref(proto.RemoveOrganizationError_RoeValidationError),
		}, nil
	}

	err = h.srv.RemoveOrganizationFromUser(ctx, userUUID, orgUUID)
	if err != nil {
		var code proto.RemoveOrganizationError
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			code = proto.RemoveOrganizationError_RoeUserNotFound
		case errors.Is(err, service.ErrOrganizationNotFound):
			code = proto.RemoveOrganizationError_RoeOrganizationNotFound
		default:
			h.log.ErrorCtx(ctx, err, "Failed to remove organization from user")
			return nil, err
		}
		h.log.WarnCtx(ctx, "Remove organization from user failed")
		return &proto.RemoveOrganizationResponse{
			Error: pointer.Ref(code),
		}, nil
	}

	return &proto.RemoveOrganizationResponse{
		Success: &proto.Empty{},
	}, nil
}
