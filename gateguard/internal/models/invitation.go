package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"

	"gateguard/internal/pkg/hasher"
	proto "gateguard/protocols/gateguard"
)

type InvitationStatus int

const (
	Pending InvitationStatus = iota
	Accepted
	Declined
	Ignored
	Revoked
	Unknown
)

func (s InvitationStatus) String() string {
	return [...]string{"pending", "accepted", "declined", "ignored", "revoked", "unknown"}[s]
}

func InvitationStatusFromString(status string) (InvitationStatus, error) {
	switch status {
	case "pending":
		return Pending, nil
	case "accepted":
		return Accepted, nil
	case "declined":
		return Declined, nil
	case "ignored":
		return Ignored, nil
	case "revoked":
		return Revoked, nil
	default:
		return Unknown, fmt.Errorf("invalid InvitationStatus: %s", status)
	}
}

type Invitation struct {
	tableName    struct{}         `pg:"invitations,discard_unknown_columns"`
	Inviter      string           `pg:"inviter"`
	Invitee      string           `pg:"invitee"`
	InviteeRole  string           `pg:"invitee_role"`
	Organization *uuid.UUID       `pg:"organization"`
	CreatedAt    time.Time        `pg:"created_at,default:CURRENT_TIMESTAMP"`
	ReferralCode string           `pg:"referral_code"`
	Status       InvitationStatus `pg:"-"`
	StatusSQL    string           `pg:"status"`
}

func (i *Invitation) BeforeInsert(ctx context.Context) (context.Context, error) {
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now()
	}
	i.StatusSQL = i.Status.String()
	return ctx, nil
}

func (i *Invitation) AfterSelect(_ context.Context) error {
	status, err := InvitationStatusFromString(i.StatusSQL)
	if err != nil {
		return fmt.Errorf("parse invitation status: %w", err)
	}
	i.Status = status
	return nil
}

func (i *Invitation) GenerateReferralCode() string {
	refCode := hasher.HashToFixed8Chars(
		fmt.Sprintf(
			"%s%s%s%s",
			i.Inviter,
			i.Invitee,
			pointer.SafeDeref(i.Organization).String(),
			i.CreatedAt.String(),
		),
	)

	return refCode
}

// InvitationToProto converts an Invitation model to its proto representation
func InvitationToProto(invitation *Invitation) *proto.Invitation {
	return &proto.Invitation{
		Inviter:      invitation.Inviter,
		Invitee:      invitation.Invitee,
		Organization: invitation.Organization.String(),
		Status:       proto.InvitationStatus(invitation.Status),
		ReferralCode: invitation.ReferralCode,
		Role:         RoleTypeFromString(invitation.InviteeRole),
		CreatedAt:    invitation.CreatedAt.Unix(),
	}
}

// RoleTypeFromString converts a string to a RoleType.
func RoleTypeFromString(role string) proto.RoleType {
	switch role {
	case "owner":
		return proto.RoleType_RoleOwner
	case "billing":
		return proto.RoleType_RoleBilling
	case "creator":
		return proto.RoleType_RoleCreator
	case "common":
		return proto.RoleType_RoleCommon
	default:
		return proto.RoleType_RoleUnknown
	}
}

// InvitationsToProto converts a slice of Invitation models to their proto representations
func InvitationsToProto(invitations []*Invitation) []*proto.Invitation {
	protoInvitations := make([]*proto.Invitation, len(invitations))
	for i, invitation := range invitations {
		protoInvitations[i] = InvitationToProto(invitation)
	}
	return protoInvitations
}
