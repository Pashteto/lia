package service

import (
	"errors"

	"gateguard/internal/pkg/clients/organizations"
)

var (
	ErrUserNotFound              = errors.New("user not found")
	ErrUserDeleted               = errors.New("user has been deleted")
	ErrUserMismatch              = errors.New("user mismatch")
	ErrOrganizationAlreadyExists = errors.New("organization already exists")
	ErrOrganizationNotFound      = errors.New("organization not found")
	ErrInviteNotAllowed          = errors.New("user is not allowed to invite to organization")
	ErrInviteStatusNotValid      = errors.New("the invitation timed out or does not exist")
	ErrOrganizationNotValid      = errors.New("entered organization is not valid")
	ErrSuchUserAlreadyExists     = errors.New("this user already exists in the organization")
	ErrInvitesRateLimitReached   = errors.New("invites rate limit reached")
)

// HandleOrganizationError maps errors to HTTP status codes and payloads
func handleOrganizationError(err error) error {
	switch err {
	case organizations.ErrMultipleResults, organizations.ErrOrganizationNotFound, organizations.ErrValidationError,
		organizations.ErrInvalidRequest, organizations.ErrOrganizationDeleted:
		return ErrOrganizationNotValid
	case organizations.ErrPermissionDenied:
		return ErrInviteNotAllowed
	default:
		return err
	}
}
