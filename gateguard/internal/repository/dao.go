package repository

import (
	"time"

	"github.com/gofrs/uuid"

	"gateguard/internal/models"
)

type AllInvitationsFilter struct {
	Inviter      *string
	Invitee      *string
	Organization *uuid.UUID
	Statuses     []models.InvitationStatus
	DateFrom     *time.Time
	DateTo       *time.Time
	Limit        uint64
	Offset       uint64
}
