package auth

import (
	"context"
	"fmt"
	"time"
)

// GatekeeperConfig is the subset of configuration the validator needs to reach
// gateway.fm's Gatekeeper service.
type GatekeeperConfig struct {
	Address string
	Timeout time.Duration
}

// gatekeeperValidator validates bearer tokens against Gatekeeper.
//
// TODO(gatekeeper): wire the real gRPC client once the Gatekeeper proto/client
// module and a reachable instance are confirmed (design spec §9). Until then
// Validate returns an error, so non-mock auth SAFELY DENIES rather than guessing
// the token contract. This is the single seam to fill to go live.
type gatekeeperValidator struct {
	cfg GatekeeperConfig
}

// NewGatekeeperValidator builds a TokenValidator backed by Gatekeeper.
func NewGatekeeperValidator(cfg GatekeeperConfig) TokenValidator {
	return &gatekeeperValidator{cfg: cfg}
}

func (g *gatekeeperValidator) Validate(_ context.Context, _ string) (*Claims, error) {
	return nil, fmt.Errorf(
		"gatekeeper validation not yet wired (address=%q): confirm client/proto and a reachable instance (spec §9)",
		g.cfg.Address,
	)
}
