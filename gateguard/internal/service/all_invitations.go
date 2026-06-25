package service

import (
	"context"
	"fmt"

	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"

	"gateguard/internal/models"
	"gateguard/internal/repository"
)

const (
	defaultLimit  = 100
	defaultOffset = 0
	maxLimit      = 1000
)

type AllInvitationsFilter struct {
	Inviter      *string
	Invitee      *string
	Organization *uuid.UUID
	Statuses     []models.InvitationStatus
	Limit        *int32
	Offset       *int32
}

func (u *UsersService) AllInvitations(ctx context.Context, filter *AllInvitationsFilter) ([]*models.Invitation, bool, error) {
	var (
		limit  = pointer.SafeDeref(filter.Limit)
		offset = pointer.SafeDeref(filter.Offset)
	)

	if limit <= 0 {
		limit = defaultLimit
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	if offset < 0 {
		offset = defaultOffset
	}

	invitations, hasMore, err := u.repository.AllInvitations(ctx, &repository.AllInvitationsFilter{
		Inviter:      filter.Inviter,
		Invitee:      filter.Invitee,
		Organization: filter.Organization,
		Statuses:     filter.Statuses,
		Limit:        uint64(limit),
		Offset:       uint64(offset),
	})
	if err != nil {
		u.log.ErrorCtx(ctx, err, "failed to fetch invitations")
		return nil, false, fmt.Errorf("failed to fetch invitations: %w", err)
	}

	return invitations, hasMore, nil
}
