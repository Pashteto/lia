package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/gateway-fm/scriptorium/pointer"

	"gateguard/internal/models"
	"gateguard/internal/pkg/clients/organizations"
	omodels "gateguard/internal/pkg/clients/organizations/models"
	"gateguard/internal/repository"
)

type ReactToInvitationIn struct {
	InviteeEmail string
	Status       models.InvitationStatus
	RefCode      string
}

func (u *UsersService) ReactToInvitation(ctx context.Context, in ReactToInvitationIn) error {
	inviteeUser := &models.User{Email: in.InviteeEmail}

	err := u.trm.Do(ctx, func(ctx context.Context) error {
		err := u.repository.GetUser(ctx, inviteeUser, repository.Email)
		if err != nil {
			u.log.ErrorCtx(ctx, err, "repository error while trying to get user")

			if errors.Is(err, repository.ErrUserNotFound) {
				return ErrUserNotFound
			}

			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if inviteeUser.Status != models.UserActive {
			return ErrUserDeleted
		}

		invitation := &models.Invitation{ReferralCode: in.RefCode}
		err = u.repository.GetInvitation(ctx, invitation, repository.InvitationByReferralCode)
		if err != nil {
			if errors.Is(err, repository.ErrInvitationNotFound) {
				return ErrInviteStatusNotValid
			}

			return fmt.Errorf("failed to check if invitation exists: %w", err)
		}

		if invitation.Status != models.Pending {
			u.log.WarnCtx(ctx, "trying to accept invitation with wrong status")

			return ErrInviteStatusNotValid
		}

		if invitation.Invitee != in.InviteeEmail {
			u.log.WarnCtx(ctx, "trying to accept invitation of another user")

			return ErrUserMismatch
		}

		if omodels.StringToRoleType(invitation.InviteeRole) == omodels.RoleUnknown {
			u.log.WarnCtx(ctx, "wrong role type")
			return fmt.Errorf("internal error, incorrect role type in the db: %s", invitation.InviteeRole)
		}

		switch in.Status {
		case models.Accepted:
			if invitation.Organization == nil {
				u.log.WarnCtx(ctx, "referral with empty org uuid")
				return fmt.Errorf("personal referrals not implemented yet")
			}

			_, err = u.orgs.UpdateUserRoleInOrganization(ctx, organizations.UpdateUserRoleParams{
				OrganizationUUID: *invitation.Organization,
				Email:            in.InviteeEmail,
				UserUUID:         &inviteeUser.UUID,
				Role:             pointer.Ref(omodels.StringToRoleType(invitation.InviteeRole)),
				Status:           pointer.Ref(omodels.UserStatusAccepted),
			})
			if err != nil {
				return handleOrganizationError(err)
			}
		case models.Declined:
			err = u.orgs.RemoveUserFromOrganization(ctx,
				organizations.RemoveUserParams{
					OrganizationUUID: *invitation.Organization,
					Email:            in.InviteeEmail,
				})
			if err != nil {
				return handleOrganizationError(err)
			}
		}

		invitation.Status = in.Status
		invitation.StatusSQL = in.Status.String()

		err = u.repository.UpdateInvitationBy(ctx, invitation, repository.InvitationByReferralCode, "status")
		if err != nil {
			u.log.ErrorCtx(ctx, err, "repository error while trying to update invitation")

			return fmt.Errorf("failed to update invitation status after successful accept or decline: %w", err)
		}

		return nil
	})

	return err
}
