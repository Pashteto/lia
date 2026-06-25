package templates

import (
	"context"
	"fmt"

	"github.com/gateway-fm/scriptorium/clog"
)

const (
	userInviteToOrganizationTemplateName = "organization_invite"
)

func NewUserInviteToOrg(
	log *clog.CustomLogger,
	invitedByEmail, referralLink, organizationName string,
) *UserInviteToOrg {
	return &UserInviteToOrg{
		InviterEmail:     invitedByEmail,
		ReferralLink:     referralLink,
		OrganizationName: organizationName,
		log:              log,
	}
}

type UserInviteToOrg struct {
	TemplateDownloadLink string
	InviterEmail         string
	ReferralLink         string
	OrganizationName     string

	log *clog.CustomLogger
}

func (t UserInviteToOrg) GetTemplateAsString(ctx context.Context) (string, error) {
	return parseTemplate(ctx, parseTemplateIn{
		log:      t.log,
		metadata: t,
	})
}

func (t UserInviteToOrg) TemplateName() string {
	return userInviteToOrganizationTemplateName
}

func (t UserInviteToOrg) Subject() string {
	return fmt.Sprintf(
		"Subject: Presto: User %s has invited you to join Presto organization %s!",
		t.InviterEmail,
		t.OrganizationName,
	)
}
