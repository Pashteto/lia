package notificator

import (
	"context"

	"gateguard/internal/pkg/notificator/templates"
)

func (s *SMTPNotificator) InviteUserToOrganization(ctx context.Context, to string, tmpl *templates.UserInviteToOrg) error {
	return s.sendTemplate(ctx, to, tmpl)
}
