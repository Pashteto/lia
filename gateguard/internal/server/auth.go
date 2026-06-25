package server

import (
	"context"
	"fmt"

	"gateguard/internal/models"
	proto "gateguard/protocols/gateguard"
)

func (h *GateguardHandlers) SignInOAuth(ctx context.Context, req *proto.User) (*proto.TokenResponse, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_email": req.Email,
	})
	h.log.DebugCtx(ctx, "SignInOAuth handler called")

	token, user, err := h.srv.SignInOAuth(ctx, models.UserFromProto(req))
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to sign in user")
		return nil, fmt.Errorf("signIn user: %w", err)
	}

	return &proto.TokenResponse{Token: token, UserCreatedOrRestored: user.CreatedOrRestored}, nil
}

func (h *GateguardHandlers) SignOut(ctx context.Context, req *proto.TokenRequest) (*proto.Empty, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_token": string(req.Token),
	})
	h.log.DebugCtx(ctx, "SignOut handler called")

	if err := h.srv.SignOut(ctx, req.Token); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to sign out user")
		return nil, fmt.Errorf("signOut user: %w", err)
	}

	return &proto.Empty{}, nil
}

func (h *GateguardHandlers) CheckAuth(ctx context.Context, req *proto.TokenRequest) (*proto.User, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"request_token": string(req.Token),
	})
	h.log.DebugCtx(ctx, "CheckAuth handler called")

	user, err := h.srv.CheckAuth(ctx, req.Token)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to check auth user")
		return nil, fmt.Errorf("check auth user: %w", err)
	}

	if user.Status == models.UserDeleted {
		h.log.WarnCtx(ctx, "User is deleted")
		return nil, fmt.Errorf("user %s is deleted: %w", user.Email, err)
	}

	return user.Proto(), nil
}
