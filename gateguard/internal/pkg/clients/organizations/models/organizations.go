package models

import (
	"time"

	"github.com/gofrs/uuid"

	proto "gateguard/protocols/organizations"
)

// Role represents a user's role within an organization
type Role struct {
	UserUUID uuid.UUID
	Role     RoleType
	Email    string
	Status   UserStatus
}

// RoleType represents the type of role a user can have
type RoleType string

func (r RoleType) String() string {
	return string(r)
}

func StringToRoleType(s string) RoleType {
	switch s {
	case "owner":
		return RoleOwner
	case "billing":
		return RoleBilling
	case "creator":
		return RoleCreator
	case "common":
		return RoleCommon
	default:
		return RoleUnknown
	}
}

const (
	RoleUnknown RoleType = "unknown"
	RoleOwner   RoleType = "owner"
	RoleBilling RoleType = "billing"
	RoleCreator RoleType = "creator"
	RoleCommon  RoleType = "common"
)

// RoleTypeFromProto converts a proto role type to a RoleType
func RoleTypeFromProto(protoRole proto.RoleType) RoleType {
	switch protoRole {
	case proto.RoleType_RoleOwner:
		return RoleOwner
	case proto.RoleType_RoleBilling:
		return RoleBilling
	case proto.RoleType_RoleCreator:
		return RoleCreator
	case proto.RoleType_RoleCommon:
		return RoleCommon
	default:
		return RoleUnknown
	}
}

// Proto converts a RoleType to a proto role type
func (r RoleType) Proto() proto.RoleType {
	switch r {
	case RoleOwner:
		return proto.RoleType_RoleOwner
	case RoleBilling:
		return proto.RoleType_RoleBilling
	case RoleCreator:
		return proto.RoleType_RoleCreator
	case RoleCommon:
		return proto.RoleType_RoleCommon
	default:
		return proto.RoleType_RoleUnknown
	}
}

// RoleFromProto converts a proto role to a Role
func RoleFromProto(protoRole *proto.Role) *Role {
	return &Role{
		UserUUID: uuid.FromBytesOrNil(protoRole.GetUserUuid()),
		Role:     RoleTypeFromProto(protoRole.GetRole()),
		Status:   ConvertProtoUserStatusToModel(protoRole.GetStatus()),
		Email:    protoRole.GetEmail(),
	}
}

// RoleToProto converts a Role to a proto role
func RoleToProto(role *Role) *proto.Role {
	return &proto.Role{
		UserUuid: role.UserUUID.Bytes(),
		Role:     role.Role.Proto(),
		Status:   ConvertModelUserStatusToProto(role.Status),
		Email:    role.Email,
	}
}

// RolesFromProto converts a slice of proto roles to a slice of Roles
func RolesFromProto(protoRoles []*proto.Role) []*Role {
	roles := make([]*Role, len(protoRoles))
	for i, protoRole := range protoRoles {
		roles[i] = RoleFromProto(protoRole)
	}
	return roles
}

// RolesToProto converts a slice of Roles to a slice of proto roles
func RolesToProto(roles []*Role) []*proto.Role {
	protoRoles := make([]*proto.Role, len(roles))
	for i, role := range roles {
		protoRoles[i] = RoleToProto(role)
	}
	return protoRoles
}

type Member struct {
	Role   RoleType
	UUID   uuid.UUID
	Status UserStatus
}

type Organization struct {
	UUID         uuid.UUID
	Name         string
	RollupsUUIDs []uuid.UUID
	Members      map[string]*Member
	Status       OrganizationStatus
	CreatedAt    time.Time
	ModifiedAt   time.Time
}

type OrganizationStatus string

const (
	OrganizationStatusActive  OrganizationStatus = "active"
	OrganizationStatusDeleted OrganizationStatus = "deleted"
)

func (os OrganizationStatus) String() string {
	return string(os)
}

// Proto converts an OrganizationStatus to a proto status
func (os OrganizationStatus) Proto() proto.Status {
	switch os {
	case OrganizationStatusActive:
		return proto.Status_StatusActive
	case OrganizationStatusDeleted:
		return proto.Status_StatusDeleted
	default:
		return proto.Status_StatusUnknown
	}
}

// OrganizationFromProto converts a proto organization to an Organization
func OrganizationFromProto(protoOrg *proto.Organization) *Organization {
	userEmailsWithRoles := make(map[string]*Member)
	for email, role := range protoOrg.GetUserEmailsWithRoles() {
		userUUID, err := uuid.FromBytes(role.UserUuid)
		if err != nil {
			return nil
		}

		userEmailsWithRoles[email] = &Member{
			UUID:   userUUID,
			Role:   RoleType(role.Role),
			Status: ConvertProtoUserStatusToModel(role.Status),
		}
	}

	rollupsUUIDs := make([]uuid.UUID, len(protoOrg.GetRollupsUuids()))
	for i, rollupUUID := range protoOrg.GetRollupsUuids() {
		rollupsUUIDs[i] = uuid.FromStringOrNil(string(rollupUUID))
	}

	return &Organization{
		UUID:         uuid.FromStringOrNil(string(protoOrg.GetUuid())), // TODO: think how to do smarter here
		Name:         protoOrg.GetName(),
		RollupsUUIDs: rollupsUUIDs,
		Members:      userEmailsWithRoles,
		Status:       OrganizationStatus(protoOrg.GetStatus().String()),
		CreatedAt:    time.Unix(protoOrg.GetCreatedAt(), 0),
		ModifiedAt:   time.Unix(protoOrg.GetModifiedAt(), 0),
	}
}

// OrganizationsFromProto converts a slice of proto organizations to a slice of Organizations
func OrganizationsFromProto(protoOrgs []*proto.Organization) []*Organization {
	orgs := make([]*Organization, len(protoOrgs))
	for i, protoOrg := range protoOrgs {
		orgs[i] = OrganizationFromProto(protoOrg)
	}
	return orgs
}
