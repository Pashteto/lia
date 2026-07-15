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

func (u *UsersService) SignInOAuth(ctx context.Context, user *models.User) ([]byte, *models.User, error) {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"user_email": user.Email,
		"ref_code":   user.RefCode,
	})

	err := u.getOrCreateUser(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	if user.RefCode != "" {
		err = u.trm.Do(ctx, func(ctx context.Context) error {
			invitation := &models.Invitation{ReferralCode: user.RefCode}
			err = u.repository.GetInvitation(ctx, invitation, repository.InvitationByReferralCode)
			if err != nil {
				u.log.ErrorCtx(ctx, err, "error occurred while trying to find invite by referral code")
				return fmt.Errorf("could not find such invite")
			}

			ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
				"invited_by":              invitation.Inviter,
				"invitation_status":       invitation.Status,
				"invitation_organization": pointer.SafeDeref(invitation.Organization).String(),
			})

			if invitation.Status != models.Pending {
				u.log.WarnCtx(ctx, "the invitation has timed out, revoked or declined/accepted already")

				return ErrInviteStatusNotValid
			}

			if !user.EmailVerified {
				return ErrEmailNotVerified
			}

			invitation.Status = models.Accepted
			err = u.repository.UpdateInvitationBy(ctx, invitation, repository.InvitationByReferralCode)
			if err != nil {
				u.log.ErrorCtx(ctx, err, "error occurred while trying to update invite by referral code")
				return fmt.Errorf("could not update this invite")
			}

			if invitation.Organization != nil {
				err = u.AddOrganizationToUser(ctx, user.UUID, *invitation.Organization)
				if err != nil {
					return err
				}

				// all checks concerning adding this user had been done preliminary. However, there are corner cases,
				// when the organization is deleted when the user accepts invite. All in all, if 500 is received from this handler -
				// a redirect to the main page should happen and popup should say something like "Something went wrong!
				// Either the organization does not exist no more, either there is a temporary error.
				// Please ask for another invite"

				_, err = u.orgs.AddUserToOrganization(ctx, organizations.AddUserParams{
					OrganizationUUID: *invitation.Organization,
					UserUUID:         user.UUID,
					Email:            user.Email,
					Role:             omodels.StringToRoleType(invitation.InviteeRole),
					Status:           omodels.UserStatusAccepted,
				})
				if err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return nil, nil, err
		}
	}

	if user.Status == models.UserDeleted {
		user.Status = models.UserActive
		err = u.repository.UpdateUserBy(ctx, user, repository.Email, "status")
		if err != nil {
			u.log.ErrorCtx(ctx, err, "error occurred while trying to update user with status active")
			return nil, nil, err
		}
		user.CreatedOrRestored = true
	}

	err = u.repository.UpdateUserBy(ctx, user, repository.Email, "ip")
	if err != nil {
		u.log.ErrorCtx(ctx, err, "error occurred while trying to update user with login ip")
	}

	token, err := u.createJWT(user)
	if err != nil {
		u.log.ErrorCtx(ctx, err, "create user session token")
		return nil, nil, fmt.Errorf("create user session token: %w", err)
	}

	return token, user, err
}

func (u *UsersService) getOrCreateUser(ctx context.Context, user *models.User) error {
	err := u.repository.GetUser(ctx, user, repository.Email)
	if err != nil && errors.Is(err, repository.ErrUserNotFound) {
		user.StatusSQL = models.UserActive.String()
		user.Role = models.UserRoleCommon

		err = u.repository.CreateUser(ctx, user)
		if err != nil {
			u.log.ErrorCtx(ctx, err, fmt.Sprintf("create user %s", user.Email))
			return fmt.Errorf("create user %s: %w", user.Email, err)
		}
		user.CreatedOrRestored = true

		return nil
	}

	if err != nil {
		u.log.ErrorCtx(ctx, err, fmt.Sprintf("get user %s from database", user.Email))
		return fmt.Errorf("get user %s from database: %w", user.Email, err)
	}

	return nil
}
