package auth

import (
	"context"
	"fmt"

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

	resp, err := s.client.SignInOAuth(ctx, &gg.User{Email: email, Name: name})
	if err != nil {
		return "", fmt.Errorf("gateguard signin: %w", err)
	}
	if resp == nil || len(resp.Token) == 0 {
		return "", fmt.Errorf("gateguard returned an empty token")
	}
	return string(resp.Token), nil
}
