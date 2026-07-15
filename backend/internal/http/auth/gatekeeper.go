package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	gg "github.com/Pashteto/lia/protocols/gateguard"
)

// GatekeeperConfig is the subset of configuration needed to reach GateGuard.
type GatekeeperConfig struct {
	Address string
	Timeout time.Duration
}

// ggClient is the slice of the generated GateguardServiceClient used here
// (satisfied by gg.NewGateguardServiceClient). Kept small so tests can fake it.
type ggClient interface {
	CheckAuth(ctx context.Context, in *gg.TokenRequest, opts ...grpc.CallOption) (*gg.User, error)
	SignInOAuth(ctx context.Context, in *gg.User, opts ...grpc.CallOption) (*gg.TokenResponse, error)
	SignUpWithPassword(ctx context.Context, in *gg.SignUpRequest, opts ...grpc.CallOption) (*gg.TokenResponse, error)
	SignInWithPassword(ctx context.Context, in *gg.PasswordSignInRequest, opts ...grpc.CallOption) (*gg.TokenResponse, error)
}

// gatekeeperValidator validates bearer tokens against GateGuard's CheckAuth RPC.
// The token stays opaque to Lia — GateGuard validates the JWT and returns the user.
type gatekeeperValidator struct {
	cfg    GatekeeperConfig
	client ggClient
}

// NewGatekeeperValidator dials GateGuard and returns a TokenValidator. The gRPC
// connection is lazy (grpc.NewClient connects on first call).
//
// Transport is insecure: GateGuard runs on the box's internal/loopback network
// alongside Lia. Switch to TLS credentials if it ever moves off-box (spec §7).
func NewGatekeeperValidator(cfg GatekeeperConfig) (TokenValidator, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gatekeeper %q: %w", cfg.Address, err)
	}
	return &gatekeeperValidator{cfg: cfg, client: gg.NewGateguardServiceClient(conn)}, nil
}

// newValidatorWithClient injects a client (used by tests).
func newValidatorWithClient(c ggClient) *gatekeeperValidator {
	return &gatekeeperValidator{client: c}
}

func (g *gatekeeperValidator) Validate(ctx context.Context, token string) (*Claims, error) {
	if g.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.cfg.Timeout)
		defer cancel()
	}

	u, err := g.client.CheckAuth(ctx, &gg.TokenRequest{Token: []byte(token)})
	if err != nil {
		return nil, fmt.Errorf("gatekeeper checkauth: %w", err)
	}

	subject := ""
	if id, err := uuid.FromBytes(u.Uuid); err == nil {
		subject = id.String()
	}

	return &Claims{Subject: subject, Email: u.Email, Name: u.Name, Role: u.Role.String(), EmailVerified: u.EmailVerified}, nil
}
