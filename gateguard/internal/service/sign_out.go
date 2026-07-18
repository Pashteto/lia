package service

import (
	"context"
)

func (u *UsersService) SignOut(ctx context.Context, token []byte) error {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"token": string(token),
	})

	u.log.DebugCtx(ctx, "user sign out")

	// Try to delete from regular session first
	err := u.sessions.Delete(ctx, token)
	if err != nil && u.extendedSession != nil {
		// If regular session fails, try extended session
		err = u.extendedSession.Delete(ctx, token)
	}

	return err
}
