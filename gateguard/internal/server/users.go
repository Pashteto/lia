package server

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"gateguard/internal/models"
	proto "gateguard/protocols/gateguard"
)

func (h *GateguardHandlers) UserByUUID(ctx context.Context, req *proto.UUIDRequest) (*proto.User, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_uuid": string(req.Uuid),
	})
	h.log.DebugCtx(ctx, "UserByUUID handler called")
	userUUID, err := uuid.FromBytes(req.Uuid)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse UUID")
		return nil, fmt.Errorf("parse uuid: %w", err)
	}
	user, err := h.srv.UserByUUID(ctx, userUUID)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to select user by UUID")
		return nil, fmt.Errorf("select user: %w", err)
	}
	return user.Proto(), nil
}

func (h *GateguardHandlers) UserByEmail(ctx context.Context, req *proto.EmailRequest) (*proto.User, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_email": req.Email,
	})
	h.log.DebugCtx(ctx, "UserByEmail handler called")
	user, err := h.srv.UserByEmail(ctx, req.Email)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to select user by email")
		return nil, fmt.Errorf("select user: %w", err)
	}
	return user.Proto(), nil
}

func (h *GateguardHandlers) DeleteUser(ctx context.Context, req *proto.TokenRequest) (*proto.Empty, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_token": string(req.Token),
	})
	h.log.DebugCtx(ctx, "DeleteUser handler called")
	if err := h.srv.DeleteUser(ctx, req.Token); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to delete user")
		return nil, fmt.Errorf("delete user: %w", err)
	}
	return &proto.Empty{}, nil
}

func (h *GateguardHandlers) Users(ctx context.Context, _ *proto.Empty) (*proto.Users, error) {
	h.log.DebugCtx(ctx, "Users handler called")
	users, err := h.srv.AllUsers(ctx)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to select users")
		return nil, fmt.Errorf("select users: %w", err)
	}
	return &proto.Users{
		User: models.UsersToProto(users),
	}, nil
}

func (h *GateguardHandlers) UpdateUserRole(ctx context.Context, req *proto.UpdateUserRoleRequest) (*proto.Empty, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_user_uuid": string(req.UserUuid),
		"request_role":      req.Role.String(),
	})

	h.log.DebugCtx(ctx, "UpdateUserRole handler called")

	userUUID, err := uuid.FromBytes(req.UserUuid)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse UUID")
		return nil, fmt.Errorf("parse uuid: %w", err)
	}

	role := models.UserRoleFromProto(req.Role)
	if role == models.UserRoleUnsupported {
		h.log.WarnCtx(ctx, "Unsupported user role")
		return nil, fmt.Errorf("user role is unsupported")
	}

	if err = h.srv.UpdateRole(ctx, userUUID, role); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to update user role")
		return nil, fmt.Errorf("update user role: %w", err)
	}

	return &proto.Empty{}, nil
}

func (h *GateguardHandlers) SetUserPreferredStacks(ctx context.Context, req *proto.SetUserPreferredStacksRequest) (*proto.Empty, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_user_uuid": string(req.UserUuid),
		"stacks":            req.PreferredStacks,
	})

	h.log.DebugCtx(ctx, "SetUserPreferredStacks handler called")

	userUUID, err := uuid.FromBytes(req.UserUuid)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse UUID")
		return nil, fmt.Errorf("parse uuid: %w", err)
	}

	if err := h.srv.SetUsersPreferredStack(ctx, userUUID, req.PreferredStacks); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to set user preferred stacks")
		return nil, fmt.Errorf("set user preferred stacks: %w", err)
	}

	return &proto.Empty{}, nil
}

func (h *GateguardHandlers) SetUserTrialUsed(ctx context.Context, req *proto.SetUserTrialUsedRequest) (*proto.Empty, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_user_uuid": string(req.UserUuid),
		"trial_used":        req.TrialUsed,
	})

	h.log.DebugCtx(ctx, "SetUserTrialUsed handler called")

	userUUID, err := uuid.FromBytes(req.UserUuid)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to parse UUID")
		return nil, fmt.Errorf("parse uuid: %w", err)
	}

	if err = h.srv.SetUsersTrialUsed(ctx, userUUID, req.TrialUsed); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to set user used trial")
		return nil, fmt.Errorf("set user preferred stacks: %w", err)
	}

	return &proto.Empty{}, nil
}
