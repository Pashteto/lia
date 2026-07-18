package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gateguard/internal/models"
	"gateguard/internal/repository"
)

func (u *UsersService) UserByUUID(ctx context.Context, userUUID uuid.UUID) (user *models.User, err error) {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"user_uuid": userUUID.String(),
	})

	user = &models.User{UUID: userUUID}

	if err = u.repository.GetUser(ctx, user, repository.UserUUID); err != nil {
		u.log.ErrorCtx(ctx, err, "get user with uuid %s repository error", user.UUID.String())
		return nil, fmt.Errorf("get user %s from repository by UUID: %w", userUUID.String(), err)
	}

	return
}

func (u *UsersService) UserByEmail(ctx context.Context, email string) (user *models.User, err error) {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"user_email": email,
	})

	user = &models.User{Email: email}

	if err = u.repository.GetUser(ctx, user, repository.Email); err != nil {
		u.log.ErrorCtx(ctx, err, "get user %s from database by email", email)
		return nil, fmt.Errorf("get user %s from database by email: %w", email, err)
	}

	return
}

func (u *UsersService) DeleteUser(ctx context.Context, token []byte) error {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"token": string(token),
	})

	userUUID, err := u.userUUIDFromToken(ctx, token)
	if err != nil {
		u.log.ErrorCtx(ctx, err, "get user uuid from claims")
		return fmt.Errorf("get user uuid from claims: %w", err)
	}

	user, err := u.UserByUUID(ctx, userUUID)
	if err != nil {
		u.log.ErrorCtx(ctx, err, "get user by UUID")
		return fmt.Errorf("get user by UUID: %w", err)
	}

	user.Status = models.UserDeleted
	user.DeletedAt = time.Now()

	if err = u.repository.UpdateUserBy(ctx, user, repository.UserUUID, "status", "deleted_at"); err != nil {
		u.log.ErrorCtx(ctx, err, "update user %s status", user.UUID.String())
		return fmt.Errorf("update user %s status: %w", user.UUID.String(), err)
	}
	return nil
}

func (u *UsersService) AllUsers(ctx context.Context) ([]*models.User, error) {
	users, err := u.repository.AllUsers(ctx)
	if err != nil {
		u.log.ErrorCtx(ctx, err, "get all users from database")
	}
	return users, err
}

func (u *UsersService) UpdateRole(ctx context.Context, userUUID uuid.UUID, role models.UserRole) error {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"user_uuid": userUUID.String(),
		"role":      role.String(),
	})

	if role == models.UserRoleAdmin {
		return fmt.Errorf("admin role cannot be set through protocol calls, this incident is reported")
	}

	user := &models.User{
		UUID: userUUID,
		Role: role,
	}

	if err := u.repository.UpdateUserBy(ctx, user, repository.Email, "role"); err != nil {
		u.log.ErrorCtx(ctx, err, "update user %s", user.UUID.String())
		return fmt.Errorf("update user %s: %w", user.UUID.String(), err)
	}

	return nil
}

func (u *UsersService) createJWT(ctx context.Context, user *models.User) (token []byte, err error) {
	// Special users with extended token lifetime (3 months)
	specialUsers := []string{
		"innovativeblock501@gmail.com",
	}

	// Check if the user is a special user
	for _, specialEmail := range specialUsers {
		if user.Email == specialEmail {
			u.log.Info(fmt.Sprintf("Creating extended session token for special user: %s", user.Email))
			// For special users, use the extended session if available
			if u.extendedSession != nil {
				return u.extendedSession.Create(ctx, user.ToJWT())
			} else {
				u.log.Warn(fmt.Sprintf("Extended session not available for special user: %s", user.Email))
			}
			break
		}
	}

	u.log.Info(fmt.Sprintf("Creating regular session token for user: %s", user.Email))
	return u.sessions.Create(ctx, user.ToJWT())
}

func (u *UsersService) userUUIDFromToken(ctx context.Context, token []byte) (uuid.UUID, error) {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"token": string(token),
	})

	// Try regular session first
	claims, err := u.sessions.Get(ctx, token)
	if err != nil {
		// If regular session fails, try extended session
		if u.extendedSession != nil {
			claims, err = u.extendedSession.Get(ctx, token)
			if err != nil {
				u.log.ErrorCtx(ctx, err, "get claims from token (both regular and extended sessions)")
				return uuid.Nil, fmt.Errorf("check auth: %w", err)
			}
		} else {
			u.log.ErrorCtx(ctx, err, "get claims from token")
			return uuid.Nil, fmt.Errorf("check auth: %w", err)
		}
	}

	userUUID, ok := claims["uuid"]
	if !ok {
		return uuid.Nil, fmt.Errorf("check auth: jwt claims don't have valid user UUID")
	}

	var userID uuid.UUID
	if err := userID.Scan(userUUID); err != nil {
		return uuid.Nil, fmt.Errorf("check auth: convert userUUID to UUID: %w", err)
	}

	return userID, nil
}

func (u *UsersService) AddOrganizationToUser(ctx context.Context, userUUID, organizationUUID uuid.UUID) error {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"user_uuid":         userUUID.String(),
		"organization_uuid": organizationUUID.String(),
	})

	user, err := u.UserByUUID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		u.log.ErrorCtx(ctx, err, "failed to get user")
		return fmt.Errorf("failed to get user %w", err)
	}

	for _, org := range user.Organizations {
		if org == organizationUUID {
			return ErrOrganizationAlreadyExists
		}
	}

	user.Organizations = append(user.Organizations, organizationUUID)

	return u.repository.UpdateUserBy(ctx, user, repository.UserUUID, "organizations")
}

func (u *UsersService) RemoveOrganizationFromUser(ctx context.Context, userUUID, organizationUUID uuid.UUID) error {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"user_uuid":         userUUID.String(),
		"organization_uuid": organizationUUID.String(),
	})

	user, err := u.UserByUUID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		u.log.ErrorCtx(ctx, err, "failed to get user")
		return fmt.Errorf("failed to get user %w", err)
	}

	for i, org := range user.Organizations {
		if org == organizationUUID {
			user.Organizations = append(user.Organizations[:i], user.Organizations[i+1:]...)

			return u.repository.UpdateUserBy(ctx, user, repository.UserUUID, "organizations")
		}
	}

	return ErrOrganizationNotFound
}

func (u *UsersService) SetUsersPreferredStack(ctx context.Context, userUUID uuid.UUID, stacks []int64) error {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"user_uuid": userUUID.String(),
		"stacks":    stacks,
	})

	user, err := u.UserByUUID(ctx, userUUID)
	if err != nil {
		u.log.ErrorCtx(ctx, err, "failed to get user by UUID")

		return fmt.Errorf("failed to get user by UUID: %w", err)
	}

	user.PreferredStacks = stacks

	if err = u.repository.UpdateUserBy(ctx, user, repository.UserUUID, "preferred_stacks"); err != nil {
		u.log.ErrorCtx(ctx, err, "update user %s preferred stacks", user.UUID.String())

		return fmt.Errorf("update user %s preferred stacks: %w", user.UUID.String(), err)
	}

	return nil
}

func (u *UsersService) SetUsersTrialUsed(ctx context.Context, userUUID uuid.UUID, trialUsed bool) error {
	ctx = u.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"user_uuid":  userUUID.String(),
		"trial_used": trialUsed,
	})

	user, err := u.UserByUUID(ctx, userUUID)
	if err != nil {
		u.log.ErrorCtx(ctx, err, "failed to get user by UUID")

		return fmt.Errorf("failed to get user by UUID: %w", err)
	}

	user.TrialUsed = trialUsed

	if err = u.repository.UpdateUserBy(ctx, user, repository.UserUUID, "trial_used"); err != nil {
		u.log.ErrorCtx(ctx, err, "update user %s used trial", user.UUID.String())

		return fmt.Errorf("update user %s used trial: %w", user.UUID.String(), err)
	}

	return nil
}
