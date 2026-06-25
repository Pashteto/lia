package server

import (
	"fmt"
	"net"
	"time"

	"github.com/gateway-fm/scriptorium/clog"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	mwRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcSentry "github.com/johnbellone/grpc-middleware-sentry"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"gateguard/config"
)

// IGrpc is basic GRPC server interface
type IGrpc interface {
	Listen()
	Close()

	GetServer() *grpc.Server
}

// Server is Grpc Server instance
type Server struct {
	addr     string
	listener net.Listener

	*grpc.Server
	logger *clog.CustomLogger
}

// NewServer create new GRPC server instance
func NewServer(cfg *config.Server, logger *clog.CustomLogger) (IGrpc, error) {
	server := &Server{
		addr:   fmt.Sprintf("0.0.0.0:%d", cfg.Port),
		logger: logger,
	}

	if err := server.initListener(); err != nil {
		return nil, fmt.Errorf("GRPC server initializing: %w", err)
	}

	if err := server.initServer(cfg.Timeout); err != nil {
		return nil, fmt.Errorf("GRPC server initializing: %w", err)
	}

	return server, nil
}

// Listen open and listening incoming Tcp connections to Grpc Server port
func (s *Server) Listen() {
	s.logger.Info(fmt.Sprintf("listen and serve GRPC on %s", s.addr))

	if err := s.Serve(s.listener); err != nil {
		s.logger.Error(fmt.Errorf("errored listening for grpc connections: %s", err).Error())
	}
}

// GetServer return Grpc Server core instance
func (s *Server) GetServer() *grpc.Server {
	return s.Server
}

// Close closing Grpc Server listener and resetting connections
func (s *Server) Close() {
	if s.Server != nil {
		s.GracefulStop()
	}
}

// initListener initialize GRPC server connections listener
func (s *Server) initListener() (err error) {
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen GRPC on addr %s: %w", s.addr, err)
	}
	return nil
}

// initServer initialize GRPC server core
func (s *Server) initServer(timeout string) error {
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		return fmt.Errorf("initialize GRPC server: %w", err)
	}

	middlewares := middleware.ChainUnaryServer(
		grpcSentry.UnaryServerInterceptor(),
		mwRecovery.UnaryServerInterceptor(),
	)

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}

	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     360 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
		MaxConnectionAgeGrace: 5 * time.Second,   // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
		Time:                  5 * time.Second,   // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout:               1 * time.Second,   // Wait 1 second for the ping ack before assuming the connection is dead
	}

	if workers := viper.GetInt("grpcworkers"); workers > 0 {
		s.Server = grpc.NewServer(
			grpc.KeepaliveEnforcementPolicy(kaep),
			grpc.KeepaliveParams(kasp),
			grpc.ConnectionTimeout(duration),
			grpc.UnaryInterceptor(middlewares),
			grpc.MaxSendMsgSize(40*1024*1024),
			grpc.MaxRecvMsgSize(40*1024*1024),
			grpc.NumStreamWorkers(uint32(workers)),
		)
	} else {
		s.Server = grpc.NewServer(
			grpc.KeepaliveEnforcementPolicy(kaep),
			grpc.KeepaliveParams(kasp),
			grpc.ConnectionTimeout(duration),
			grpc.UnaryInterceptor(middlewares),
			grpc.MaxSendMsgSize(60*1024*1024),
			grpc.MaxRecvMsgSize(60*1024*1024),
		)
	}

	return nil
}
