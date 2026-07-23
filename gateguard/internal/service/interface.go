package service

import (
	"context"

	sessions "github.com/andskur/gatekeeper"
	"github.com/gofrs/uuid"

	"gateguard/internal/models"
)

//go:generate ../../bin/mockery --name IUsersService

// IUsersService defines the methods for user-related operations.
type IUsersService interface {
	// SignInOAuth handles OAuth sign-in for a user and returns a JWT token, the user, and any error encountered.
	SignInOAuth(ctx context.Context, user *models.User) ([]byte, *models.User, error)

	// SignOut deletes the user session associated with the given JWT token.
	SignOut(ctx context.Context, token []byte) error

	// CheckAuth verifies the given session JWT and returns the associated user and any error encountered.
	CheckAuth(ctx context.Context, token []byte) (*models.User, error)

	// SignUpWithPassword creates a credentialed account (or attaches a password to a
	// pre-existing passwordless account) and returns a session JWT, the user, and any error.
	SignUpWithPassword(ctx context.Context, email, name, plain string) ([]byte, *models.User, error)

	// SignInWithPassword verifies the password for the given email and returns a session JWT.
	SignInWithPassword(ctx context.Context, email, plain string) ([]byte, *models.User, error)

	// RequestEmailVerification issues + persists a verification token (STUB: send is logged, not emailed).
	RequestEmailVerification(ctx context.Context, email string) error

	// VerifyEmail marks the account verified if the email/token pair matches (STUB flow).
	VerifyEmail(ctx context.Context, email, token string) error

	// MarkEmailVerified (trusted) flips email_verified=true for an address without a
	// code. Called when Lia proves ownership (an emailed invitation was accepted).
	MarkEmailVerified(ctx context.Context, email string) error

	// UserByUUID retrieves a user by their UUID and returns the user and any error encountered.
	UserByUUID(ctx context.Context, userUUID uuid.UUID) (*models.User, error)

	// UserByEmail retrieves a user by their email and returns the user and any error encountered.
	UserByEmail(ctx context.Context, email string) (*models.User, error)

	// DeleteUser deletes a user session associated with the given JWT token and returns any error encountered.
	DeleteUser(ctx context.Context, token []byte) error

	// AllUsers retrieves all users and returns a list of users and any error encountered.
	AllUsers(ctx context.Context) ([]*models.User, error)

	// UpdateRole updates the role of a user identified by their UUID and returns any error encountered.
	UpdateRole(ctx context.Context, userUUID uuid.UUID, role models.UserRole) error

	// AddOrganizationToUser adds an organization to a user's list of organizations.
	// Takes the user's UUID and the organization's UUID as parameters and returns any error encountered.
	AddOrganizationToUser(ctx context.Context, userUUID, organizationUUID uuid.UUID) error

	// RemoveOrganizationFromUser removes an organization from a user's list of organizations.
	// Takes the user's UUID and the organization's UUID as parameters and returns any error encountered.
	RemoveOrganizationFromUser(ctx context.Context, userUUID, organizationUUID uuid.UUID) error

	// InviteUserByEmail sends an invitation to a user by their email.
	// Takes an InviteUserByEmailIn struct as input and returns any error encountered.
	InviteUserByEmail(ctx context.Context, in InviteUserByEmailIn) error

	// AllInvitations retrieves all invitations with optional filters and returns a list of invitations,
	// a boolean indicating if there are more invitations, and any error encountered.
	AllInvitations(ctx context.Context, filter *AllInvitationsFilter) ([]*models.Invitation, bool, error)

	// ReactToInvitation allows a user to accept or decline an invitation.
	// Takes a ReactToInvitationIn struct as input and returns any error encountered.
	ReactToInvitation(ctx context.Context, in ReactToInvitationIn) error

	// ExpireInvitations expires all invitations that are older than t time ago, where t is set in the service struct.
	// Returns any error encountered.
	ExpireInvitations(ctx context.Context) error

	SetUsersPreferredStack(ctx context.Context, userUUID uuid.UUID, stacks []int64) error

	SetUsersTrialUsed(ctx context.Context, userUUID uuid.UUID, trialUsed bool) error

	// SetExtendedSession sets an extended session for special users
	SetExtendedSession(extendedSession sessions.ISessions)
}
