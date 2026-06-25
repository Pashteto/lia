package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"
	"golang.org/x/sync/errgroup"

	"gateguard/internal/models"
	"gateguard/internal/pkg/clients/organizations"
	omodels "gateguard/internal/pkg/clients/organizations/models"
	"gateguard/internal/pkg/notificator/templates"
	"gateguard/internal/repository"
)

type InviteUserByEmailIn struct {
	InviterUUID                             uuid.UUID
	InviterEmail, InviteeEmail, InviteeRole string
	OrganizationUUID                        *uuid.UUID
}

func (u *UsersService) InviteUserByEmail(ctx context.Context, in InviteUserByEmailIn) error {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"inviter_email":      in.InviterEmail,
		"invitee_email":      in.InviteeEmail,
		"organization_id":    pointer.SafeDeref(in.OrganizationUUID).String(),
		"invitee_role":       in.InviteeRole,
		"max_weekly_invites": u.maxWeeklyInvitesNum,
	})
	u.log.DebugCtx(ctx, "inviting user by email")

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	invites, _, err := u.repository.AllInvitations(ctx, &repository.AllInvitationsFilter{
		Inviter:  &in.InviterEmail,
		DateFrom: &sevenDaysAgo,
		DateTo:   &now,
	})
	if err != nil {
		u.log.ErrorCtx(ctx, err, "repository error occurred while trying to query for invites for user")

		return fmt.Errorf("failed to check if rate limit is exceeded: %w", err)
	}

	// here, for testing purposes, we introduce the concept of VIP users, or the ones who can send unlimited invites
	if !isVIP(in.InviterEmail) && u.maxWeeklyInvitesNum != 0 && len(invites) >= u.maxWeeklyInvitesNum {
		u.log.WarnCtx(ctx, "user has exceeded the rate limit on sending invites")
		nextTryAllowedAt := invites[0].CreatedAt.AddDate(0, 0, 7)

		return fmt.Errorf("user will be able to request next invite at %v: %w", nextTryAllowedAt.String(), ErrInvitesRateLimitReached)
	}

	// if there is already an invitation from this user to this invitee to this org in pending state
	// -- you cannot send an invitation to him

	invites, _, err = u.repository.AllInvitations(ctx, &repository.AllInvitationsFilter{
		Inviter:      &in.InviterEmail,
		Invitee:      &in.InviteeEmail,
		Organization: in.OrganizationUUID,
		Statuses:     []models.InvitationStatus{models.Pending},
	})

	if len(invites) > 0 {
		u.log.WarnCtx(ctx, "user has exceeded the rate limit on sending invites")

		return fmt.Errorf("user will be able to request next invite when the user accepts or declines his previous: %w", ErrInvitesRateLimitReached)
	}

	inviteeUser := &models.User{Email: in.InviteeEmail}
	err = u.repository.GetUser(ctx, inviteeUser, repository.Email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		u.log.ErrorCtx(ctx, err, "repository error occurred while trying to query for invitee")

		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	invitation := &models.Invitation{
		Inviter:      in.InviterEmail,
		Invitee:      in.InviteeEmail,
		InviteeRole:  in.InviteeRole,
		Organization: in.OrganizationUUID,
		Status:       models.Pending,
		CreatedAt:    time.Now(),
	}

	refCode := invitation.GenerateReferralCode()
	invitation.ReferralCode = refCode

	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"referral_code": refCode,
	})

	// idempotency
	existingInvitation := &models.Invitation{ReferralCode: refCode}
	err = u.repository.GetInvitation(ctx, existingInvitation, repository.InvitationByReferralCode)
	if err == nil {
		u.log.InfoCtx(ctx, "invitation with this referral code already exists")

		return nil
	}

	organizationName, err := u.organizationPreCheck(ctx, preCheckIn{
		organizationUUID: in.OrganizationUUID,
		inviterEmail:     in.InviterEmail,
		inviteeEmail:     in.InviteeEmail,
		inviteeRole:      omodels.StringToRoleType(in.InviteeRole),
	})
	if err != nil {
		return fmt.Errorf("failed to precheck organization invitation: %w", err)
	}

	switch organizationName {
	case nil:
		return fmt.Errorf("personal referrals not supported yet")
	default:
		err = u.notificator.InviteUserToOrganization(
			ctx,
			in.InviteeEmail,
			templates.NewUserInviteToOrg(
				u.log,
				in.InviterEmail,
				u.lb.GetReferralLink(refCode),
				*organizationName,
			),
		)
		if err != nil {
			return fmt.Errorf("failed to send invitation email: %w", err)
		}

		u.log.InfoCtx(ctx, "sent email with invitation to the specified email")
	}

	return u.trm.Do(ctx, func(ctx context.Context) error {
		err = u.repository.CreateInvitation(ctx, invitation)
		if err != nil {
			u.log.ErrorCtx(ctx, err, "repository error while trying to create invitation")
			return fmt.Errorf("failed to create invitation: %w", err)
		}

		if organizationName != nil {
			_, err = u.orgs.AddUserToOrganization(ctx, organizations.AddUserParams{
				OrganizationUUID: *in.OrganizationUUID,
				Email:            in.InviteeEmail,
				UserUUID:         uuid.Nil,
				Role:             omodels.StringToRoleType(in.InviteeRole),
				Status:           omodels.UserStatusPending,
			})
			if err != nil {
				return handleOrganizationError(err)
			}
		}

		return nil
	})
}

type preCheckIn struct {
	organizationUUID *uuid.UUID
	inviterEmail     string
	inviteeEmail     string
	inviteeRole      omodels.RoleType
}

func (u *UsersService) organizationPreCheck(ctx context.Context, in preCheckIn) (*string, error) {
	if in.organizationUUID == nil {
		return nil, nil
	}

	eg, egCtx := errgroup.WithContext(ctx)

	var (
		isAvailable  bool
		organization *omodels.Organization
	)

	eg.Go(func() error {
		var err error

		isAvailable, err = u.orgs.CheckOrganizationInvitationOperationAvailability(
			egCtx,
			organizations.CheckOrganizationInvitationOperationAvailabilityParams{
				OrganizationUUID: *in.organizationUUID,
				Email:            in.inviterEmail,
				InviteeRole:      in.inviteeRole,
			})
		if err != nil {
			return handleOrganizationError(err)
		}

		if !isAvailable {
			u.log.WarnCtx(ctx, "user is not allowed to send invites")
			return ErrInviteNotAllowed
		}

		return nil
	})

	eg.Go(func() error {
		var err error
		organization, err = u.orgs.GetOrganization(egCtx, *in.organizationUUID)

		return err
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if member := organization.Members[in.inviteeEmail]; member != nil {
		return nil, ErrSuchUserAlreadyExists
	}

	return pointer.Ref(organization.Name), nil
}

// function isVIP lists all users who can send unlimited invites
func isVIP(email string) bool {
	return slices.Contains([]string{
		"test.adm.010203@gmail.com",
		"anton.polshkov@gateway.fm",
		"karmasleeper@gmail.com",
	}, email)
}
