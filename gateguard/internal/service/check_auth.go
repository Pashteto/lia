package service

import (
	"context"

	"gateguard/internal/models"
)

func (u *UsersService) CheckAuth(ctx context.Context, token []byte) (*models.User, error) {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"token": string(token),
	})

	userUUID, err := u.userUUIDFromToken(ctx, token)
	if err != nil {
		return nil, err
	}

	user, err := u.UserByUUID(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	return user, err
}
