package handler_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/gateway-fm/scriptorium/clog"
	"github.com/stretchr/testify/suite"

	"gateguard/internal/server"
	"gateguard/internal/service/mocks"
)

var errInternal = errors.New("internal error")

const (
	bearerToken = "somebearertoken"
)

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}

type ServerSuite struct {
	suite.Suite

	ctx              context.Context
	handlers         *server.GateguardHandlers
	usersServiceMock *mocks.IUsersService
	log              *clog.CustomLogger
}

func (s *ServerSuite) SetupTest() {
	s.ctx = context.Background()
	s.usersServiceMock = mocks.NewIUsersService(s.T())
	s.log = clog.NewCustomLogger(os.Stdout, clog.LevelDebug, false)

	s.handlers = server.NewGateguardHandlers(s.usersServiceMock, s.log)
}
