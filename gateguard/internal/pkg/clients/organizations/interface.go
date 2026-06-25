package organizations

import (
	"context"

	"github.com/gofrs/uuid"

	omodels "gateguard/internal/pkg/clients/organizations/models"
)

//go:generate ../../../../bin/mockery --name IOrganizationsAPI

// IOrganizationsAPI defines the methods for interacting with organizations in the system.
type IOrganizationsAPI interface {
	// CreateOrganization creates a new organization with the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for creating the organization.
	// Returns the created organization or an error if creation failed.
	CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*omodels.Organization, error)

	// DeleteOrganization deletes an organization identified by the given UUID.
	// ctx: The context for the request.
	// organizationUUID: UUID of the organization to be deleted.
	// Returns an error if the deletion failed.
	DeleteOrganization(ctx context.Context, organizationUUID uuid.UUID) error

	// GetOrganization retrieves the details of an organization identified by the given UUID.
	// ctx: The context for the request.
	// organizationUUID: UUID of the organization to retrieve.
	// Returns the organization details or an error if retrieval failed.
	GetOrganization(ctx context.Context, organizationUUID uuid.UUID) (*omodels.Organization, error)

	// AddUserToOrganization adds a user to an organization with the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for adding the user to the organization.
	// Returns the updated organization or an error if the addition failed.
	AddUserToOrganization(ctx context.Context, params AddUserParams) (*omodels.Organization, error)

	// UpdateUserRoleInOrganization updates the role of a user in an organization with the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for updating the user's role.
	// Returns the updated organization or an error if the update failed.
	UpdateUserRoleInOrganization(ctx context.Context, params UpdateUserRoleParams) (*omodels.Organization, error)

	// RemoveUserFromOrganization removes a user from an organization with the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for removing the user from the organization.
	// Returns an error if the removal failed.
	RemoveUserFromOrganization(ctx context.Context, params RemoveUserParams) error

	// ListOrganizations lists all organizations with optional filtering options.
	// ctx: The context for the request.
	// opts: Optional filtering options for listing organizations.
	// Returns a list of organizations or an error if listing failed.
	ListOrganizations(ctx context.Context, opts *AllOrganizationsOpts) ([]*omodels.Organization, *bool, error)

	// TransferRollupsToOrganization transfers rollups to an organization with the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for transferring the rollups.
	// Returns the updated organization or an error if the transfer failed.
	TransferRollupsToOrganization(ctx context.Context, params TransferRollupsParams) (*omodels.Organization, error)

	// DeleteRollupsFromOrganization deletes rollups from an organization with the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for deleting the rollups.
	// Returns the updated organization or an error if the deletion failed.
	DeleteRollupsFromOrganization(ctx context.Context, params DeleteRollupParams) (*omodels.Organization, error)

	// UpdateOrganization updates the organization's details based on the specified parameters.
	// ctx: The context for the request.
	// params: Parameters required for updating the organization's details.
	// Returns the updated organization or an error if the update failed.
	UpdateOrganization(ctx context.Context, params UpdateOrganizationParams) (*omodels.Organization, error)

	// CheckRollupsOperationAvailability checks if a rollup operation is available for the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for checking rollup operation availability.
	// Returns true if the operation is available, otherwise returns false and an error if the operation fails.
	CheckRollupsOperationAvailability(ctx context.Context, params CheckRollupsOperationAvailabilityParams) (bool, error)

	// CheckOrganizationOperationAvailability checks if an organization operation is available for the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for checking organization operation availability.
	// Returns true if the operation is available, otherwise returns false and an error if the operation fails.
	CheckOrganizationOperationAvailability(ctx context.Context, params CheckOrganizationOperationAvailabilityParams) (bool, error)

	// CheckOrganizationInvitationOperationAvailability checks if an organization operation is available for the given parameters.
	// ctx: The context for the request.
	// params: Parameters required for checking organization invitation operation availability.
	// Returns true if the operation is available, otherwise returns false and an error if the operation fails.
	CheckOrganizationInvitationOperationAvailability(ctx context.Context, params CheckOrganizationInvitationOperationAvailabilityParams) (bool, error)

	// Close closes any resources held by the API.
	// Returns an error if closing failed.
	Close() error
}
