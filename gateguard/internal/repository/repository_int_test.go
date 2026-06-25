//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gateway-fm/scriptorium/transactions"
	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/suite"

	"gateguard/internal/models"
	"gateguard/internal/pkg/tests/fake"
	repoTesting "gateguard/internal/pkg/tests/repository_testing"
	"gateguard/internal/repository"
)

const DBAddressEnvVar = "DB_LOCAL_CONN_STRING"

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

type RepositorySuite struct {
	suite.Suite

	ctx context.Context
	db  *pg.DB

	trf  *transactions.PgTransactionFactory
	trm  transactions.TransactionManager
	repo repository.IRepository
}

func (s *RepositorySuite) SetupTest() {
	s.ctx = context.Background()

	repoTesting.InitTestingConfig(s.ctx, s.T())
	s.db = repoTesting.InitDB(s.ctx, s.T(), repoTesting.MustGetEnv(s.ctx, s.T(), DBAddressEnvVar))

	s.trf = transactions.NewPgTransactionFactory(s.db)
	s.trm = transactions.NewPgTransactionManager(s.trf,
		transactions.Options{AlwaysRollback: true},
	)
	s.repo = repository.NewRepository(s.trf)
}

func (s *RepositorySuite) TestRepository_GetUser() {
	s.Run("success", func() {
		user := fake.User()
		userGot := &models.User{Email: user.Email}

		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			err := s.repo.CreateUser(ctx, user)
			s.Require().NoError(err, "error while trying to create user")

			return s.repo.GetUser(ctx, userGot, repository.Email)
		})

		s.Require().NoError(err, "error while trying to search for created user")
		s.Require().Equal(user, userGot, "should get the same user")
	})

	s.Run("no result", func() {
		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			user := fake.User()
			wrongEmail := gofakeit.Email()

			err := s.repo.CreateUser(ctx, user)
			s.Require().NoError(err, "error while trying to create user")

			return s.repo.GetUser(ctx, &models.User{Email: wrongEmail}, repository.Email)
		})

		s.Require().ErrorIs(err, pg.ErrNoRows, "no error encountered when searching with wrong email")
	})
}

func (s *RepositorySuite) TestRepository_CreateUser() {
	s.Run("success", func() {
		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			user := fake.User()
			err := s.repo.CreateUser(ctx, user)

			return err
		})

		s.Require().NoError(err, "error while trying to create user")
	})
}

func (s *RepositorySuite) TestRepository_CreateInvitation() {
	s.Run("success", func() {
		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			user := fake.User()
			invitation := fake.Invitation()

			invitation.Inviter = user.Email

			err := s.repo.CreateUser(ctx, user)
			s.Require().NoError(err, "error while trying to create user")

			return s.repo.CreateInvitation(ctx, invitation)
		})

		s.Require().NoError(err, "error while trying to create invitation")
	})
}

func (s *RepositorySuite) TestRepository_GetInvitation() {
	s.Run("success", func() {
		user := fake.User()
		invitation := fake.Invitation()

		invitation.Inviter = user.Email

		invitationGot := &models.Invitation{Invitee: invitation.Invitee}

		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			err := s.repo.CreateUser(ctx, user)
			s.Require().NoError(err, "error while trying to create user")

			err = s.repo.CreateInvitation(ctx, invitation)
			s.Require().NoError(err, "error while trying to create invitation")

			return s.repo.GetInvitation(ctx, invitationGot, repository.InvitationByInvitee)
		})

		s.Require().NoError(err, "error while trying to search for created invitation")
		s.Require().Equal(invitation.Invitee, invitationGot.Invitee, "should get the same invitation")
	})

	s.Run("no result", func() {
		user := fake.User()
		invitation := fake.Invitation()
		wrongEmail := gofakeit.Email()

		invitation.Inviter = user.Email

		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			err := s.repo.CreateUser(ctx, user)
			s.Require().NoError(err, "error while trying to create user")

			err = s.repo.CreateInvitation(ctx, invitation)
			s.Require().NoError(err, "error while trying to create invitation")

			return s.repo.GetInvitation(ctx, &models.Invitation{Invitee: wrongEmail}, repository.InvitationByInvitee)
		})

		s.Require().ErrorIs(err, pg.ErrNoRows, "no error encountered when searching with wrong email")
	})
}

func (s *RepositorySuite) TestRepository_UpdateInvitationBy() {
	s.Run("success", func() {
		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			user := fake.User()
			invitation := fake.Invitation()

			invitation.Inviter = user.Email

			err := s.repo.CreateUser(ctx, user)
			s.Require().NoError(err, "error while trying to create user")

			err = s.repo.CreateInvitation(ctx, invitation)
			s.Require().NoError(err, "error while trying to create invitation")

			invitation.Status = models.Accepted
			return s.repo.UpdateInvitationBy(ctx, invitation, repository.InvitationByInvitee, "status")
		})

		s.Require().NoError(err, "error while trying to update invitation")
	})

	s.Run("no result", func() {
		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			invitation := fake.Invitation()
			invitation.Status = models.Accepted

			return s.repo.UpdateInvitationBy(ctx, invitation, repository.InvitationByInvitee, "status")
		})

		s.Require().NoError(err, "no error encountered when updating non-existent invitation")
	})
}

func (s *RepositorySuite) TestRepository_AllInvitations() {
	s.Run("success", func() {
		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			user := fake.User()
			invitation1 := fake.Invitation()
			invitation2 := fake.Invitation()

			invitation1.Inviter = user.Email
			invitation2.Inviter = user.Email

			err := s.repo.CreateUser(ctx, user)
			s.Require().NoError(err, "error while trying to create user")

			err = s.repo.CreateInvitation(ctx, invitation1)
			s.Require().NoError(err, "error while trying to create invitation 1")

			err = s.repo.CreateInvitation(ctx, invitation2)
			s.Require().NoError(err, "error while trying to create invitation 2")

			filter := &repository.AllInvitationsFilter{
				Inviter:  &invitation1.Inviter,
				Statuses: []models.InvitationStatus{models.Pending},
				Limit:    10,
				Offset:   0,
			}

			invitations, hasMore, err := s.repo.AllInvitations(ctx, filter)
			if err != nil {
				return err
			}

			s.Require().Len(invitations, 2, "should get two invitations")
			s.Require().False(hasMore, "should not have more results")

			return nil
		})

		s.Require().NoError(err, "error while trying to list invitations")
	})

	s.Run("success_invitee", func() {
		err := s.trm.Do(s.ctx, func(ctx context.Context) error {
			user := fake.User()
			userInvitee := fake.User()

			invitation1 := fake.Invitation()
			invitation2 := fake.Invitation()

			invitation1.Inviter = user.Email
			invitation2.Inviter = user.Email

			invitation2.Invitee = userInvitee.Email

			err := s.repo.CreateUser(ctx, user)
			s.Require().NoError(err, "error while trying to create user")

			err = s.repo.CreateInvitation(ctx, invitation1)
			s.Require().NoError(err, "error while trying to create invitation 1")

			err = s.repo.CreateInvitation(ctx, invitation2)
			s.Require().NoError(err, "error while trying to create invitation 2")

			filter := &repository.AllInvitationsFilter{
				Inviter:  &invitation1.Inviter,
				Invitee:  &invitation2.Invitee,
				Statuses: []models.InvitationStatus{models.Pending},
				Limit:    10,
				Offset:   0,
			}

			invitations, hasMore, err := s.repo.AllInvitations(ctx, filter)
			if err != nil {
				return err
			}

			s.Require().Len(invitations, 1, "should get one invitation")
			s.Require().False(hasMore, "should not have more results")

			return nil
		})

		s.Require().NoError(err, "error while trying to list invitations")
	})
}
