package service

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"gateguard/internal/models"
	"gateguard/internal/pkg/clients/organizations"
	"gateguard/internal/repository"
)

func (u *UsersService) ExpireInvitations(ctx context.Context) error {
	oneWeekAgo := time.Now().Add(-u.invitesTTLHours)

	invites, _, err := u.repository.AllInvitations(ctx, &repository.AllInvitationsFilter{
		Statuses: []models.InvitationStatus{models.Pending},
		DateTo:   &oneWeekAgo,
	})
	if err != nil {
		u.log.ErrorCtx(ctx, err, "repository error occurred while trying to query for expired invites")

		return fmt.Errorf("failed to retrieve expired invites: %w", err)
	}

	if len(invites) == 0 {
		u.log.InfoCtx(ctx, "nothing expired")
		return nil
	}

	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"expired_invites": len(invites),
	})

	eg, egCtx := errgroup.WithContext(ctx)
	for _, invite := range invites {
		invite := invite

		eg.Go(func() error {
			egCtx = u.log.AddKeysValuesToCtx(egCtx, map[string]interface{}{
				"inviter":  invite.Invitee,
				"invitee":  invite.Invitee,
				"ref_code": invite.ReferralCode,
			})

			if invite.Organization != nil {
				err = u.orgs.RemoveUserFromOrganization(egCtx, organizations.RemoveUserParams{
					OrganizationUUID: *invite.Organization,
					Email:            invite.Invitee,
				})
				if err != nil {
					return err
				}
			}

			invite.StatusSQL = models.Ignored.String()
			err = u.repository.UpdateInvitationBy(egCtx, invite, repository.InvitationByReferralCode, "status")
			if err != nil {
				u.log.ErrorCtx(egCtx, err, "repository error occurred while trying to expire invite")

				return fmt.Errorf("failed to expire invite: %w", err)
			}

			u.log.InfoCtx(egCtx, "moved invitation to ignored status")
			return nil
		})
	}

	if err = eg.Wait(); err != nil {
		return err
	}

	u.log.InfoCtx(ctx, "completed with success")
	return nil
}
