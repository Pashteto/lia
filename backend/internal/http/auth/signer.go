package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	gg "github.com/Pashteto/lia/protocols/gateguard"
)

// Signer mints a GateGuard session token for an identity. Used by the demo-login
// endpoint (no Google): enter email -> GateGuard SignInOAuth -> JWT.
//
// DEMO-ONLY: this trusts the caller to have authenticated the user (it will
// get-or-create any email), so it must never be exposed in real prod — it's a
// known non-production control, like HTTP_MOCK_AUTH.
type Signer interface {
	SignIn(ctx context.Context, email, name string) (string, error)
	// SignUpPassword registers a credentialed account and returns a session JWT.
	SignUpPassword(ctx context.Context, email, name, password string) (string, error)
	// SignInPassword verifies a password and returns a session JWT.
	SignInPassword(ctx context.Context, email, password string) (string, error)
}

// ErrInvalidCredentials / ErrUserExists let handlers map password-auth failures
// to the right HTTP status. GateGuard does not yet return typed gRPC status
// codes, so they are classified from its error message.
var (
	ErrInvalidCredentials = fmt.Errorf("invalid credentials")
	ErrUserExists         = fmt.Errorf("user already exists")
)

func classifyAuthErr(err error) error {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "already exists"):
		return ErrUserExists
	case strings.Contains(msg, "invalid credentials"):
		return ErrInvalidCredentials
	default:
		return err
	}
}

type gatekeeperSigner struct {
	cfg    GatekeeperConfig
	client ggClient
}

// NewSigner dials GateGuard and returns a Signer backed by its SignInOAuth RPC.
func NewSigner(cfg GatekeeperConfig) (Signer, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gatekeeper %q: %w", cfg.Address, err)
	}
	return &gatekeeperSigner{cfg: cfg, client: gg.NewGateguardServiceClient(conn)}, nil
}

func newSignerWithClient(c ggClient) *gatekeeperSigner {
	return &gatekeeperSigner{client: c}
}

func (s *gatekeeperSigner) SignIn(ctx context.Context, email, name string) (string, error) {
	if s.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.Timeout)
		defer cancel()
	}

	// Status and Role MUST be set explicitly. GateGuard maps an unset proto
	// status (0 = Unknown) to an internal "unsupported" sentinel and then panics
	// stringifying it when inserting the user (index out of range). Send a valid,
	// active common user so the GateGuard-side record is well-formed.
	resp, err := s.client.SignInOAuth(ctx, &gg.User{
		Email:  email,
		Name:   name,
		Status: gg.UserStatus_UserActive,
		Role:   gg.UserRole_UserRoleCommon,
	})
	if err != nil {
		return "", fmt.Errorf("gateguard signin: %w", err)
	}
	if resp == nil || len(resp.Token) == 0 {
		return "", fmt.Errorf("gateguard returned an empty token")
	}
	return string(resp.Token), nil
}

func (s *gatekeeperSigner) SignUpPassword(ctx context.Context, email, name, password string) (string, error) {
	if s.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.Timeout)
		defer cancel()
	}
	resp, err := s.client.SignUpWithPassword(ctx, &gg.SignUpRequest{
		Email:    email,
		Name:     name,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("gateguard signup: %w", classifyAuthErr(err))
	}
	if resp == nil || len(resp.Token) == 0 {
		return "", fmt.Errorf("gateguard returned an empty token")
	}
	return string(resp.Token), nil
}

func (s *gatekeeperSigner) SignInPassword(ctx context.Context, email, password string) (string, error) {
	if s.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.Timeout)
		defer cancel()
	}
	resp, err := s.client.SignInWithPassword(ctx, &gg.PasswordSignInRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("gateguard signin: %w", classifyAuthErr(err))
	}
	if resp == nil || len(resp.Token) == 0 {
		return "", fmt.Errorf("gateguard returned an empty token")
	}
	return string(resp.Token), nil
}
