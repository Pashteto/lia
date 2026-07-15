package service_test

import (
	"context"

	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/stretchr/testify/mock"

	"gateguard/internal/models"
	"gateguard/internal/pkg/clients/organizations"
	omodels "gateguard/internal/pkg/clients/organizations/models"
	"gateguard/internal/pkg/tests/fake"
	"gateguard/internal/repository"
	"gateguard/internal/service"
)

const refCode = "abcDEF"

func (s *UseCaseSuite) Test_ReactToInvitation_Success() {
	s.Run("accept", func() {
		initialInvitee := *fake.User()
		initialInvitee.EmailVerified = true
		initialInvitation := *fake.Invitation()

		initialInvitation.Invitee = initialInvitee.Email
		initialInvitation.ReferralCode = refCode

		s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: initialInvitee.Email}, repository.Email).
			Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
				*model = initialInvitee
			}).Return(nil).Once()

		s.repo.EXPECT().GetInvitation(mock.Anything, &models.Invitation{ReferralCode: refCode}, repository.InvitationByReferralCode).
			Run(func(ctx context.Context, model *models.Invitation, getter repository.InvitationGetter) {
				cp := pointer.Ref(initialInvitation)
				cp.Status = models.Pending

				*model = *cp
			}).Return(nil).Once()

		s.oMock.EXPECT().AddUserToOrganization(mock.Anything, organizations.AddUserParams{
			OrganizationUUID: *initialInvitation.Organization,
			UserUUID:         initialInvitee.UUID,
			Role:             omodels.StringToRoleType(initialInvitation.InviteeRole),
		}).Return(&omodels.Organization{}, nil).Once()

		cp := &initialInvitation
		cp.Status = models.Accepted
		cp.StatusSQL = cp.Status.String()
		s.repo.EXPECT().UpdateInvitationBy(mock.Anything, cp, repository.InvitationByReferralCode, "status").
			Return(nil).Once()

		err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
			InviteeEmail: initialInvitee.Email,
			Status:       models.Accepted,
			RefCode:      refCode,
		})

		s.Require().NoError(err)
	})

	s.Run("decline", func() {
		initialInvitee := *fake.User()
		initialInvitee.EmailVerified = true
		initialInvitation := *fake.Invitation()

		initialInvitation.Invitee = initialInvitee.Email
		initialInvitation.ReferralCode = refCode
		initialInvitation.Status = models.Declined

		s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: initialInvitee.Email}, repository.Email).
			Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
				*model = initialInvitee
			}).Return(nil).Once()

		s.repo.EXPECT().GetInvitation(mock.Anything, &models.Invitation{ReferralCode: refCode}, repository.InvitationByReferralCode).
			Run(func(ctx context.Context, model *models.Invitation, getter repository.InvitationGetter) {
				cp := pointer.Ref(initialInvitation)
				cp.Status = models.Pending

				*model = *cp
			}).Return(nil).Once()

		cp := &initialInvitation
		cp.Status = models.Declined
		cp.StatusSQL = cp.Status.String()
		s.repo.EXPECT().UpdateInvitationBy(mock.Anything, cp, repository.InvitationByReferralCode, "status").
			Return(nil).Once()

		err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
			InviteeEmail: initialInvitee.Email,
			Status:       models.Declined,
			RefCode:      refCode,
		})

		s.Require().NoError(err)
	})
}

func (s *UseCaseSuite) Test_ReactToInvitation_Accept_BlockedWhenUnverified() {
	initialInvitee := *fake.User()
	initialInvitee.EmailVerified = false
	initialInvitee.Status = models.UserActive

	initialInvitation := *fake.Invitation()
	initialInvitation.Invitee = initialInvitee.Email
	initialInvitation.ReferralCode = refCode

	s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: initialInvitee.Email}, repository.Email).
		Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
			*model = initialInvitee
		}).Return(nil).Once()

	s.repo.EXPECT().GetInvitation(mock.Anything, &models.Invitation{ReferralCode: refCode}, repository.InvitationByReferralCode).
		Run(func(ctx context.Context, model *models.Invitation, getter repository.InvitationGetter) {
			cp := pointer.Ref(initialInvitation)
			cp.Status = models.Pending

			*model = *cp
		}).Return(nil).Once()

	err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
		InviteeEmail: initialInvitee.Email,
		Status:       models.Accepted,
		RefCode:      refCode,
	})

	s.Require().ErrorIs(err, service.ErrEmailNotVerified)
}

func (s *UseCaseSuite) Test_ReactToInvitation_Errors() {
	invitee := fake.User()
	invitee.EmailVerified = true
	invitation := fake.Invitation()

	invitation.Invitee = invitee.Email
	invitation.ReferralCode = refCode

	s.Run("user_not_found", func() {
		s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: invitee.Email}, repository.Email).
			Return(repository.ErrUserNotFound).Once()

		err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
			InviteeEmail: invitee.Email,
			Status:       models.Accepted,
			RefCode:      refCode,
		})

		s.Require().ErrorIs(err, service.ErrUserNotFound)
	})

	s.Run("repository_error", func() {
		s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: invitee.Email}, repository.Email).
			Return(errInternal).Once()

		err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
			InviteeEmail: invitee.Email,
			Status:       models.Accepted,
			RefCode:      refCode,
		})

		s.Require().ErrorIs(err, errInternal)
	})

	s.Run("invitation_not_found", func() {
		s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: invitee.Email}, repository.Email).
			Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
				*model = *invitee
			}).Return(nil).Once()

		s.repo.EXPECT().GetInvitation(mock.Anything, &models.Invitation{ReferralCode: refCode}, repository.InvitationByReferralCode).
			Return(repository.ErrInvitationNotFound).Once()

		err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
			InviteeEmail: invitee.Email,
			Status:       models.Accepted,
			RefCode:      refCode,
		})

		s.Require().ErrorIs(err, service.ErrInviteStatusNotValid)
	})

	s.Run("orgs_client_error", func() {
		s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: invitee.Email}, repository.Email).
			Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
				*model = *invitee
			}).Return(nil).Once()

		s.repo.EXPECT().GetInvitation(mock.Anything, &models.Invitation{ReferralCode: refCode}, repository.InvitationByReferralCode).
			Run(func(ctx context.Context, model *models.Invitation, getter repository.InvitationGetter) {
				cp := *invitation
				cp.Status = models.Pending
				*model = cp
			}).Return(nil).Once()

		s.oMock.EXPECT().AddUserToOrganization(mock.Anything, organizations.AddUserParams{
			OrganizationUUID: *invitation.Organization,
			UserUUID:         invitee.UUID,
			Role:             omodels.StringToRoleType(invitation.InviteeRole),
		}).Return(nil, organizations.ErrOrganizationDeleted).Once()

		err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
			InviteeEmail: invitee.Email,
			Status:       models.Accepted,
			RefCode:      refCode,
		})

		s.Require().ErrorIs(err, service.ErrOrganizationNotValid)
	})

	s.Run("orgs_client_error_not_allowed", func() {
		s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: invitee.Email}, repository.Email).
			Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
				*model = *invitee
			}).Return(nil).Once()

		s.repo.EXPECT().GetInvitation(mock.Anything, &models.Invitation{ReferralCode: refCode}, repository.InvitationByReferralCode).
			Run(func(ctx context.Context, model *models.Invitation, getter repository.InvitationGetter) {
				cp := *invitation
				cp.Status = models.Pending
				*model = cp
			}).Return(nil).Once()

		s.oMock.EXPECT().AddUserToOrganization(mock.Anything, organizations.AddUserParams{
			OrganizationUUID: *invitation.Organization,
			UserUUID:         invitee.UUID,
			Role:             omodels.StringToRoleType(invitation.InviteeRole),
		}).Return(nil, organizations.ErrPermissionDenied).Once()

		err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
			InviteeEmail: invitee.Email,
			Status:       models.Accepted,
			RefCode:      refCode,
		})

		s.Require().ErrorIs(err, service.ErrInviteNotAllowed)
	})

	s.Run("update_failed", func() {
		s.repo.EXPECT().GetUser(mock.Anything, &models.User{Email: invitee.Email}, repository.Email).
			Run(func(ctx context.Context, model *models.User, getter repository.UserGetter) {
				*model = *invitee
			}).Return(nil).Once()

		s.repo.EXPECT().GetInvitation(mock.Anything, &models.Invitation{ReferralCode: refCode}, repository.InvitationByReferralCode).
			Run(func(ctx context.Context, model *models.Invitation, getter repository.InvitationGetter) {
				cp := *invitation
				cp.Status = models.Pending
				*model = cp
			}).Return(nil).Once()

		s.oMock.EXPECT().AddUserToOrganization(mock.Anything, organizations.AddUserParams{
			OrganizationUUID: *invitation.Organization,
			UserUUID:         invitee.UUID,
			Role:             omodels.StringToRoleType(invitation.InviteeRole),
		}).Return(nil, nil).Once()

		s.repo.EXPECT().UpdateInvitationBy(mock.Anything, mock.MatchedBy(func(i *models.Invitation) bool {
			return i.ReferralCode == refCode && i.Status == models.Accepted
		}), repository.InvitationByReferralCode, "status").Return(errInternal).Once()

		err := s.service.ReactToInvitation(s.ctx, service.ReactToInvitationIn{
			InviteeEmail: invitee.Email,
			Status:       models.Accepted,
			RefCode:      refCode,
		})

		s.Require().ErrorIs(err, errInternal)
	})
}
