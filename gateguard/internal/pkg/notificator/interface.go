package notificator

import (
	"context"

	"gateguard/internal/pkg/notificator/templates"
)

//go:generate ../../../bin/mockery --name INotificator

type INotificator interface {
	InviteUserToOrganization(ctx context.Context, to string, tmpl *templates.UserInviteToOrg) error
}
