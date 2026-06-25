package organizations

import (
	"context"
	"fmt"
	"time"

	"github.com/gateway-fm/scriptorium/clog"
	"github.com/gateway-fm/scriptorium/pointer"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	omodels "gateguard/internal/pkg/clients/organizations/models"
	proto "gateguard/protocols/organizations"
)

type api struct {
	logger  *clog.CustomLogger
	timeout time.Duration
	*grpc.ClientConn
	proto.OrganizationServiceClient
}

func New(addr string, logger *clog.CustomLogger, timeout time.Duration) (*api, error) {
	rpc := &api{logger: logger, timeout: timeout}

	if err := rpc.initConn(addr); err != nil {
		return nil, fmt.Errorf("create api: %w", err)
	}

	rpc.OrganizationServiceClient = proto.NewOrganizationServiceClient(rpc.ClientConn)
	return rpc, nil
}

func (api *api) initConn(addr string) error {
	kacp := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             api.timeout,
		PermitWithoutStream: true,
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithKeepaliveParams(kacp))
	if err != nil {
		api.logger.ErrorCtx(context.Background(), err, "Failed to initialize connection")
		return err
	}
	api.ClientConn = conn
	return nil
}

type CreateOrganizationParams struct {
	Name         string
	InitialRoles []*omodels.Role
}

func (api *api) CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*omodels.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Creating organization")
	initialRoles := omodels.RolesToProto(params.InitialRoles)

	resp, err := api.OrganizationServiceClient.CreateOrganization(ctx, &proto.CreateOrganizationRequest{
		Name:         params.Name,
		InitialRoles: initialRoles,
	})
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to create organization")
		return nil, fmt.Errorf("create organization api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.CreateOrganizationError_CoeInvalidArgument:
			return nil, ErrInvalidRequest
		case proto.CreateOrganizationError_CoePermissionDenied:
			return nil, ErrPermissionDenied
		case proto.CreateOrganizationError_CoeValidationError:
			return nil, ErrValidationError
		case proto.CreateOrganizationError_CoeAlreadyExistError:
			return nil, ErrAlreadyExists
		}
	}

	api.logger.DebugCtx(ctx, "Organization created successfully")
	return omodels.OrganizationFromProto(resp.GetOrganization()), nil
}

func (api *api) DeleteOrganization(ctx context.Context, organizationUUID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Deleting organization")
	req := &proto.DeleteOrganizationRequest{
		OrganizationUuid: organizationUUID.Bytes(),
	}

	resp, err := api.OrganizationServiceClient.DeleteOrganization(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to delete organization")
		return fmt.Errorf("delete organization api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.DeleteOrganizationError_DoeNotFound:
			return ErrOrganizationNotFound
		case proto.DeleteOrganizationError_DoePermissionDenied:
			return ErrPermissionDenied
		case proto.DeleteOrganizationError_DoeValidationError:
			return ErrValidationError
		}
	}

	api.logger.DebugCtx(ctx, "Organization deleted successfully")
	return nil
}

func (api *api) GetOrganization(ctx context.Context, organizationUUID uuid.UUID) (*omodels.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Fetching organization")
	req := &proto.GetOrganizationRequest{
		Uuid: organizationUUID.Bytes(),
	}

	resp, err := api.OrganizationServiceClient.GetOrganization(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to get organization")
		return nil, fmt.Errorf("get organization api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.GetOrganizationError_GoeNotFound:
			return nil, ErrOrganizationNotFound
		case proto.GetOrganizationError_GoeMoreThanOneOrgFound:
			return nil, ErrMultipleResults
		case proto.GetOrganizationError_GoeValidationError:
			return nil, ErrValidationError
		}
	}

	api.logger.DebugCtx(ctx, "Organization fetched successfully")
	return omodels.OrganizationFromProto(resp.GetOrganization()), nil
}

type AddUserParams struct {
	OrganizationUUID uuid.UUID
	UserUUID         uuid.UUID
	Role             omodels.RoleType
	Email            string
	Status           omodels.UserStatus
}

func (api *api) AddUserToOrganization(ctx context.Context, params AddUserParams) (*omodels.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Adding user to organization")
	req := &proto.AddUserRequest{
		OrganizationUuid: params.OrganizationUUID.Bytes(),
		Role: omodels.RoleToProto(
			&omodels.Role{
				UserUUID: params.UserUUID,
				Role:     params.Role,
				Email:    params.Email,
				Status:   params.Status,
			}),
	}
	resp, err := api.OrganizationServiceClient.AddUserToOrganization(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to add user to organization")
		return nil, fmt.Errorf("add user to organization api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.AddUserError_AueNotFound:
			return nil, ErrOrganizationNotFound
		case proto.AddUserError_AuePermissionDenied:
			return nil, ErrPermissionDenied
		case proto.AddUserError_AueValidationError:
			return nil, ErrValidationError
		}
	}

	api.logger.DebugCtx(ctx, "User added to organization successfully")
	return omodels.OrganizationFromProto(resp.GetOrganization()), nil
}

type UpdateUserRoleParams struct {
	OrganizationUUID uuid.UUID
	Email            string
	Role             *omodels.RoleType
	UserUUID         *uuid.UUID
	Status           *omodels.UserStatus
}

func (api *api) UpdateUserRoleInOrganization(ctx context.Context, params UpdateUserRoleParams) (*omodels.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "updating user role in organization")

	updateReq := &proto.UpdateUserRoleRequest{
		OrganizationUuid: params.OrganizationUUID.Bytes(),
		Email:            params.Email,
	}

	if params.Role != nil {
		updateReq.Role = pointer.Ref(params.Role.Proto())
	}
	if params.Status != nil {
		updateReq.Status = pointer.Ref(omodels.ConvertModelUserStatusToProto(*params.Status))
	}
	if params.UserUUID != nil {
		updateReq.UserUuid = pointer.SafeDeref(params.UserUUID).Bytes()
	}

	resp, err := api.OrganizationServiceClient.UpdateUserRoleInOrganization(ctx, updateReq)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to update user role in organization")
		return nil, fmt.Errorf("update user role in organization api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.UpdateUserRoleError_UureOrganizationNotFound:
			return nil, ErrOrganizationNotFound
		case proto.UpdateUserRoleError_UurePermissionDenied:
			return nil, ErrPermissionDenied
		case proto.UpdateUserRoleError_UureValidationError:
			return nil, ErrValidationError
		case proto.UpdateUserRoleError_UureUserNotFound:
			return nil, ErrUserNotFound
		}
	}

	api.logger.DebugCtx(ctx, "User role updated successfully")
	return omodels.OrganizationFromProto(resp.GetOrganization()), nil
}

type RemoveUserParams struct {
	OrganizationUUID uuid.UUID
	Email            string
}

func (api *api) RemoveUserFromOrganization(ctx context.Context, params RemoveUserParams) error {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Removing user from organization")
	req := &proto.RemoveUserRequest{
		OrganizationUuid: params.OrganizationUUID.Bytes(),
		Email:            params.Email,
	}

	resp, err := api.OrganizationServiceClient.RemoveUserFromOrganization(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to remove user from organization")
		return fmt.Errorf("remove user from organization api request: %w", err)
	}

	if resp.Error != nil {
		api.logger.WarnCtx(ctx, "got handled error from organizations")

		switch resp.GetError() {
		case proto.RemoveUserError_RueNotFound:
			return ErrOrganizationNotFound
		case proto.RemoveUserError_RuePermissionDenied:
			return ErrPermissionDenied
		case proto.RemoveUserError_RueValidationError:
			return ErrValidationError
		}
	}

	api.logger.DebugCtx(ctx, "User removed from organization successfully")
	return nil
}

type TransferRollupsParams struct {
	OrganizationUUID uuid.UUID
	RollupUUIDs      []uuid.UUID
}

func (api *api) TransferRollupsToOrganization(ctx context.Context, params TransferRollupsParams) (*omodels.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Transferring rollups to organization")
	rollupUUIDs := make([][]byte, len(params.RollupUUIDs))
	for i, rollupUUID := range params.RollupUUIDs {
		rollupUUIDs[i] = rollupUUID.Bytes()
	}

	resp, err := api.OrganizationServiceClient.TransferRollupsToOrganization(ctx, &proto.TransferRollupsRequest{
		OrganizationUuid: params.OrganizationUUID.Bytes(),
		RollupUuids:      rollupUUIDs,
	})
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to transfer rollups to organization")
		return nil, fmt.Errorf("transfer rollups to organization api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.TransferRollupsError_TreNotFound:
			return nil, ErrOrganizationNotFound
		case proto.TransferRollupsError_TrePermissionDenied:
			return nil, ErrPermissionDenied
		case proto.TransferRollupsError_TreValidationError:
			return nil, ErrValidationError
		case proto.TransferRollupsError_TreAlreadyExists:
			return nil, ErrAlreadyExists
		}
	}

	api.logger.DebugCtx(ctx, "Rollups transferred to organization successfully")
	return omodels.OrganizationFromProto(resp.GetOrganization()), nil
}

type DeleteRollupParams struct {
	RollupUUIDs []uuid.UUID
}

func (api *api) DeleteRollupsFromOrganization(ctx context.Context, params DeleteRollupParams) (*omodels.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Deleting rollups from organization")
	rollupsBytes := make([][]byte, len(params.RollupUUIDs))
	for index, rollupUUID := range params.RollupUUIDs {
		rollupsBytes[index] = rollupUUID.Bytes()
	}

	resp, err := api.OrganizationServiceClient.DeleteRollupFromOrganization(ctx, &proto.DeleteRollupsRequest{
		RollupUuids: rollupsBytes,
	})
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to delete rollup from organization")
		return nil, fmt.Errorf("delete rollup from organization api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.DeleteRollupsError_DreOrganizationNotFound:
			return nil, ErrOrganizationNotFound
		case proto.DeleteRollupsError_DreRollupNotFound:
			return nil, ErrRollupNotFound
		case proto.DeleteRollupsError_DrePermissionDenied:
			return nil, ErrPermissionDenied
		case proto.DeleteRollupsError_DreValidationError:
			return nil, ErrValidationError
		}
	}

	api.logger.DebugCtx(ctx, "Rollups deleted from organization successfully")
	return omodels.OrganizationFromProto(resp.GetOrganization()), nil
}

type AllOrganizationsOpts struct {
	OrganizationUUID uuid.UUID
	UserEmails       []string
	Statuses         []omodels.OrganizationStatus
	Limit            uint64
	Offset           uint64
}

func (api *api) ListOrganizations(ctx context.Context, opts *AllOrganizationsOpts) ([]*omodels.Organization, *bool, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Listing organizations")

	protoStatuses := make([]proto.Status, len(opts.Statuses))
	for index, status := range opts.Statuses {
		protoStatuses[index] = status.Proto()
	}

	req := &proto.ListOrganizationsRequest{
		OrganizationUuid: opts.OrganizationUUID.Bytes(),
		UserEmails:       opts.UserEmails,
		Statuses:         protoStatuses,
		Limit:            opts.Limit,
		Offset:           opts.Offset,
	}

	resp, err := api.OrganizationServiceClient.ListOrganizations(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to list organizations")
		return nil, pointer.Ref(false), fmt.Errorf("list organizations api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.ListOrganizationsError_LoeNotFound:
			api.logger.WarnCtx(ctx, "No organizations found")
			return []*omodels.Organization{}, nil, nil
		case proto.ListOrganizationsError_LoeInvalidQuery:
			return nil, nil, ErrInvalidRequest
		case proto.ListOrganizationsError_LoeValidationError:
			return nil, nil, ErrValidationError
		}
	}

	api.logger.DebugCtx(ctx, "Organizations listed successfully")
	return omodels.OrganizationsFromProto(resp.GetOrganizations()), resp.HasMore, nil
}

type UpdateOrganizationParams struct {
	OrganizationUUID uuid.UUID
	Name             string
}

func (api *api) UpdateOrganization(ctx context.Context, params UpdateOrganizationParams) (*omodels.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Updating organization")
	req := &proto.UpdateOrganizationRequest{
		OrganizationUuid: params.OrganizationUUID.Bytes(),
		Name:             params.Name,
	}

	resp, err := api.OrganizationServiceClient.UpdateOrganization(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to update organization")
		return nil, fmt.Errorf("update organization api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.UpdateOrganizationError_UoeNotFound:
			return nil, ErrOrganizationNotFound
		case proto.UpdateOrganizationError_UoePermissionDenied:
			return nil, ErrPermissionDenied
		case proto.UpdateOrganizationError_UoeValidationError:
			return nil, ErrValidationError
		}
	}

	api.logger.DebugCtx(ctx, "Organization updated successfully")
	return omodels.OrganizationFromProto(resp.GetOrganization()), nil
}

type CheckRollupsOperationAvailabilityParams struct {
	OrganizationUUID uuid.UUID
	Email            string
	OperationType    omodels.RollupOperationType
}

func (api *api) CheckRollupsOperationAvailability(ctx context.Context, params CheckRollupsOperationAvailabilityParams) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.InfoCtx(ctx, "Checking rollup operation availability")

	req := &proto.IsRollupOperationAvailableRequest{
		OrganizationUuid: params.OrganizationUUID.Bytes(),
		Email:            params.Email,
		OperationType:    params.OperationType.ToProto(),
	}

	resp, err := api.OrganizationServiceClient.IsRollupOperationAvailable(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to check rollup operation availability")
		return false, fmt.Errorf("check rollup operation availability api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.IsRollupOperationAvailableError_IroaErrorNotFound:
			return false, ErrOrganizationNotFound
		case proto.IsRollupOperationAvailableError_IroaErrorPermissionDenied:
			return false, ErrPermissionDenied
		case proto.IsRollupOperationAvailableError_IroaErrorValidationError:
			return false, ErrValidationError
		case proto.IsRollupOperationAvailableError_IroaErrorOrganizationDeleted:
			return false, ErrOrganizationDeleted
		case proto.IsRollupOperationAvailableError_IroaErrorMoreThanOneOrgFound:
			return false, ErrMultipleResults
		}
	}

	api.logger.DebugCtx(ctx, "Rollup operation availability checked")
	return resp.GetIsAvailable(), nil
}

type CheckOrganizationOperationAvailabilityParams struct {
	OrganizationUUID uuid.UUID
	Email            string
	OperationType    omodels.OrganizationOperationType
}

func (api *api) CheckOrganizationOperationAvailability(ctx context.Context, params CheckOrganizationOperationAvailabilityParams) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Checking organization operation availability")
	req := &proto.IsOrganizationOperationAvailableRequest{
		OrganizationUuid: params.OrganizationUUID.Bytes(),
		Email:            params.Email,
		OperationType:    params.OperationType.ToProto(),
	}

	resp, err := api.OrganizationServiceClient.IsOrganizationOperationAvailable(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to check organization operation availability")
		return false, fmt.Errorf("check organization operation availability api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.IsOrganizationOperationAvailableError_IooaErrorNotFound:
			return false, ErrOrganizationNotFound
		case proto.IsOrganizationOperationAvailableError_IooaErrorPermissionDenied:
			return false, ErrPermissionDenied
		case proto.IsOrganizationOperationAvailableError_IooaErrorValidationError:
			return false, ErrValidationError
		case proto.IsOrganizationOperationAvailableError_IooaErrorOrganizationDeleted:
			return false, ErrOrganizationDeleted
		}
	}

	api.logger.DebugCtx(ctx, "Organization operation availability checked")
	return resp.GetIsAvailable(), nil
}

type CheckOrganizationInvitationOperationAvailabilityParams struct {
	OrganizationUUID uuid.UUID
	Email            string
	InviteeRole      omodels.RoleType
}

func (api *api) CheckOrganizationInvitationOperationAvailability(ctx context.Context, params CheckOrganizationInvitationOperationAvailabilityParams) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, api.timeout)
	defer cancel()

	api.logger.DebugCtx(ctx, "Checking organization operation availability")
	req := &proto.IsOrganizationInvitationOperationAvailableRequest{
		OrganizationUuid: params.OrganizationUUID.Bytes(),
		InviterEmail:     params.Email,
		InviteeRole:      params.InviteeRole.Proto(),
	}

	resp, err := api.OrganizationServiceClient.IsOrganizationInvitationOperationAvailable(ctx, req)
	if err != nil {
		api.logger.ErrorCtx(ctx, err, "Failed to check organization operation availability")
		return false, fmt.Errorf("check organization operation availability api request: %w", err)
	}

	if resp.Error != nil {
		switch resp.GetError() {
		case proto.IsOrganizationInvitationOperationAvailableError_IoioaErrorNotFound:
			return false, ErrOrganizationNotFound
		case proto.IsOrganizationInvitationOperationAvailableError_IoioaErrorPermissionDenied:
			return false, ErrPermissionDenied
		case proto.IsOrganizationInvitationOperationAvailableError_IoioaErrorValidationError:
			return false, ErrValidationError
		case proto.IsOrganizationInvitationOperationAvailableError_IoioaErrorOrganizationDeleted:
			return false, ErrOrganizationDeleted
		}
	}

	api.logger.DebugCtx(ctx, "Organization operation availability checked")
	return resp.GetIsAvailable(), nil
}

func (api *api) Close() error {
	return api.ClientConn.Close()
}
