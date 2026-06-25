package repository

import (
	"fmt"

	"github.com/go-pg/pg/v10/orm"

	"gateguard/internal/models"
)

type InvitationGetter int

const (
	InvitationByInvitee InvitationGetter = iota
	InvitationByReferralCode
)

var invitationGetters = [...]string{
	InvitationByInvitee:      "invitee",
	InvitationByReferralCode: "referral_code",
}

func (g InvitationGetter) String() string {
	return invitationGetters[g]
}

func (g InvitationGetter) Get(query *orm.Query, inv *models.Invitation) error {
	switch g {
	case InvitationByInvitee:
		query.Where(fmt.Sprintf("%s = ?", g.String()), inv.Invitee)
	case InvitationByReferralCode:
		query.Where(fmt.Sprintf("%s = ?", g.String()), inv.ReferralCode)
	default:
		return fmt.Errorf("unsupported invitation getter: %s", g)
	}
	return nil
}
