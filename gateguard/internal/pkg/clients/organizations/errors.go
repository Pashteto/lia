package organizations

import "errors"

var (
	ErrInvalidRequest       = errors.New("invalid request")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrValidationError      = errors.New("validation error")
	ErrOrganizationNotFound = errors.New("such organization not found")
	ErrAlreadyExists        = errors.New("such entity already exists")
	ErrMultipleResults      = errors.New("multiple organizations found for given user id, ambiguous results")
	ErrUserNotFound         = errors.New("user not found")
	ErrOrganizationDeleted  = errors.New("organization deleted, you cannot perform operations on it")
	ErrRollupNotFound       = errors.New("rollup not found")
)
