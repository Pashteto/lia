package service_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/gateway-fm/scriptorium/clog"
	"github.com/gateway-fm/scriptorium/transactions"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"gateguard/internal/models"
	oMocks "gateguard/internal/pkg/clients/organizations/mocks"
	lMocks "gateguard/internal/pkg/links/mocks"
	nMocks "gateguard/internal/pkg/notificator/mocks"
	"gateguard/internal/repository"
	"gateguard/internal/repository/mocks"
	"gateguard/internal/service"
	serviceMocks "gateguard/internal/service/mocks"
)

var errInternal = errors.New("internal error")

const (
	ipKey = "ip"
)

func TestUsecase(t *testing.T) {
	suite.Run(t, new(UseCaseSuite))
}

type UseCaseSuite struct {
	suite.Suite

	ctx      context.Context
	log      *clog.CustomLogger
	repo     *mocks.IRepository
	sessions *serviceMocks.ISessions
	service  service.IUsersService
	nMock    *nMocks.INotificator
	oMock    *oMocks.IOrganizationsAPI
	lMock    *lMocks.LinkBuilder
	trm      transactions.TransactionManager
}

func (s *UseCaseSuite) SetupTest() {
	s.ctx = context.Background()

	s.log = clog.NewCustomLogger(os.Stdout, clog.LevelDebug, false)
	s.repo = mocks.NewIRepository(s.T())
	s.sessions = serviceMocks.NewISessions(s.T())
	s.nMock = nMocks.NewINotificator(s.T())
	s.oMock = oMocks.NewIOrganizationsAPI(s.T())
	s.lMock = lMocks.NewLinkBuilder(s.T())
	s.trm = transactions.NewTrmStub()

	s.service = service.NewUsersService(s.log, s.repo, s.sessions, s.nMock, s.oMock, s.trm, s.lMock, 0, 0)
}

func (s *UseCaseSuite) Test_AddOrganizationToUser() {
	s.Run("success", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		user := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{},
		}

		userWithOrg := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{orgUUID},
		}

		s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.UserUUID).Return(nil).Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
			*model = *user
		}).Once()

		s.repo.EXPECT().UpdateUserBy(mock.Anything, userWithOrg, repository.UserUUID, "organizations").Return(nil).Once()

		err := s.service.AddOrganizationToUser(s.ctx, userUUID, orgUUID)
		s.Require().NoError(err)
	})

	s.Run("user_not_found", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.UserUUID).Return(repository.ErrUserNotFound).Once()

		err := s.service.AddOrganizationToUser(s.ctx, userUUID, orgUUID)
		s.Require().ErrorIs(err, service.ErrUserNotFound)
	})

	s.Run("organization_already_exists", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		user := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{orgUUID},
		}

		s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.UserUUID).Return(nil).Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
			*model = *user
		}).Once()

		err := s.service.AddOrganizationToUser(s.ctx, userUUID, orgUUID)
		s.Require().ErrorIs(err, service.ErrOrganizationAlreadyExists)
	})

	s.Run("internal_error", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		user := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{},
		}

		userWithOrg := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{orgUUID},
		}

		s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.UserUUID).Return(nil).Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
			*model = *user
		}).Once()

		s.repo.EXPECT().UpdateUserBy(mock.Anything, userWithOrg, repository.UserUUID, "organizations").Return(errInternal).Once()

		err := s.service.AddOrganizationToUser(s.ctx, userUUID, orgUUID)
		s.Require().ErrorIs(err, errInternal)
	})
}

func (s *UseCaseSuite) Test_RemoveOrganizationFromUser() {
	s.Run("success", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		user := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{orgUUID},
		}

		userWithoutOrg := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{},
		}

		s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.UserUUID).Return(nil).Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
			*model = *user
		}).Once()

		s.repo.EXPECT().UpdateUserBy(mock.Anything, userWithoutOrg, repository.UserUUID, "organizations").Return(nil).Once()

		err := s.service.RemoveOrganizationFromUser(s.ctx, userUUID, orgUUID)
		s.Require().NoError(err)
	})

	s.Run("user_not_found", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.UserUUID).Return(service.ErrUserNotFound).Once()

		err := s.service.RemoveOrganizationFromUser(s.ctx, userUUID, orgUUID)
		s.Require().ErrorIs(err, service.ErrUserNotFound)
	})

	s.Run("organization_not_found", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		user := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{},
		}

		s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.UserUUID).Return(nil).Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
			*model = *user
		}).Once()

		err := s.service.RemoveOrganizationFromUser(s.ctx, userUUID, orgUUID)
		s.Require().ErrorIs(err, service.ErrOrganizationNotFound)
	})

	s.Run("internal_error", func() {
		userUUID := uuid.Must(uuid.NewV4())
		orgUUID := uuid.Must(uuid.NewV4())

		user := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{orgUUID},
		}

		userWithoutOrg := &models.User{
			UUID:          userUUID,
			Organizations: []uuid.UUID{},
		}

		s.repo.EXPECT().GetUser(mock.Anything, mock.Anything, repository.UserUUID).Return(nil).Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
			*model = *user
		}).Once()

		s.repo.EXPECT().UpdateUserBy(mock.Anything, userWithoutOrg, repository.UserUUID, "organizations").Return(errInternal).Once()

		err := s.service.RemoveOrganizationFromUser(s.ctx, userUUID, orgUUID)
		s.Require().ErrorIs(err, errInternal)
	})
}
