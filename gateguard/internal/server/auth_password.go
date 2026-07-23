package server

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"gateguard/internal/service"
	proto "gateguard/protocols/gateguard"
)

func (h *GateguardHandlers) SignUpWithPassword(ctx context.Context, req *proto.SignUpRequest) (*proto.TokenResponse, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"request_email": req.Email})
	h.log.DebugCtx(ctx, "SignUpWithPassword handler called")

	token, user, err := h.srv.SignUpWithPassword(ctx, req.Email, req.Name, req.Password)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to sign up user")
		return nil, fmt.Errorf("sign up user: %w", err)
	}

	return &proto.TokenResponse{Token: token, UserCreatedOrRestored: user.CreatedOrRestored}, nil
}

func (h *GateguardHandlers) SignInWithPassword(ctx context.Context, req *proto.PasswordSignInRequest) (*proto.TokenResponse, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"request_email": req.Email})
	h.log.DebugCtx(ctx, "SignInWithPassword handler called")

	token, _, err := h.srv.SignInWithPassword(ctx, req.Email, req.Password)
	if err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to sign in user")
		return nil, fmt.Errorf("sign in user: %w", err)
	}

	return &proto.TokenResponse{Token: token}, nil
}

func (h *GateguardHandlers) RequestEmailVerification(ctx context.Context, req *proto.EmailRequest) (*proto.Empty, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"request_email": req.Email})
	h.log.DebugCtx(ctx, "RequestEmailVerification handler called")

	if err := h.srv.RequestEmailVerification(ctx, req.Email); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to request email verification")
		return nil, fmt.Errorf("request verification: %w", err)
	}

	return &proto.Empty{}, nil
}

func (h *GateguardHandlers) MarkEmailVerified(ctx context.Context, req *proto.EmailRequest) (*proto.Empty, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"request_email": req.Email})
	h.log.DebugCtx(ctx, "MarkEmailVerified handler called")

	if err := h.srv.MarkEmailVerified(ctx, req.Email); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to mark email verified")
		return nil, fmt.Errorf("mark email verified: %w", err)
	}

	return &proto.Empty{}, nil
}

func (h *GateguardHandlers) VerifyEmail(ctx context.Context, req *proto.VerifyEmailRequest) (*proto.Empty, error) {
	ctx = h.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"request_email": req.Email})
	h.log.DebugCtx(ctx, "VerifyEmail handler called")

	if err := h.srv.VerifyEmail(ctx, req.Email, req.Token); err != nil {
		h.log.ErrorCtx(ctx, err, "Failed to verify email")
		switch {
		case errors.Is(err, service.ErrVerificationTooManyAttempts):
			return nil, status.Error(codes.ResourceExhausted, "verification attempts exceeded")
		case errors.Is(err, service.ErrVerificationCodeExpired):
			return nil, status.Error(codes.DeadlineExceeded, "verification code expired")
		case errors.Is(err, service.ErrVerificationTokenInvalid):
			return nil, status.Error(codes.InvalidArgument, "verification token invalid")
		default:
			return nil, fmt.Errorf("verify email: %w", err)
		}
	}

	return &proto.Empty{}, nil
}
