package server

import (
	"github.com/gateway-fm/scriptorium/clog"

	"gateguard/internal/service"
	proto "gateguard/protocols/gateguard"
)

type GateguardHandlers struct {
	log *clog.CustomLogger
	srv service.IUsersService

	*proto.UnimplementedGateguardServiceServer
}

// NewGateguardHandlers creates a new GateguardHandlers instance
func NewGateguardHandlers(srv service.IUsersService, log *clog.CustomLogger) *GateguardHandlers {
	return &GateguardHandlers{
		log: log,
		srv: srv,
	}
}
