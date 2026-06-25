package models

import (
	"strings"

	proto "gateguard/protocols/gateguard"
)

type UserRole int

const (
	UserRoleCommon = iota

	UserRoleViewer

	UserRoleBilling

	UserRoleAdmin

	UserRoleUnsupported
)

var userRoles = [...]string{
	UserRoleCommon:  "common",
	UserRoleViewer:  "viewer",
	UserRoleBilling: "billing",
	UserRoleAdmin:   "admin",
}

func (s UserRole) String() string {
	return userRoles[s]
}

func UserRoleFromString(s string) UserRole {
	for index, userRole := range userRoles {
		if strings.EqualFold(s, userRole) {
			return UserRole(index)
		}
	}

	return UserRoleUnsupported
}

func (s UserRole) Proto() proto.UserRole {
	switch s {
	case UserRoleCommon:
		return proto.UserRole_UserRoleCommon
	case UserRoleViewer:
		return proto.UserRole_UserRoleViewer
	case UserRoleBilling:
		return proto.UserRole_UserRoleBilling
	case UserRoleAdmin:
		return proto.UserRole_UserRoleAdmin
	default:
		return proto.UserRole_UserRoleUnknown
	}
}

func UserRoleFromProto(u proto.UserRole) UserRole {
	switch u {
	case proto.UserRole_UserRoleAdmin:
		return UserRoleAdmin
	case proto.UserRole_UserRoleViewer:
		return UserRoleViewer
	case proto.UserRole_UserRoleBilling:
		return UserRoleBilling
	case proto.UserRole_UserRoleCommon:
		return UserRoleCommon
	default:
		return UserRoleUnsupported
	}
}
