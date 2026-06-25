package models

import (
	"fmt"
	"strings"

	proto "gateguard/protocols/gateguard"
)

// UserStatus represent available User Statuses
type UserStatus int

const (
	// UserActive mean that user have
	// verified and active profile
	UserActive UserStatus = iota

	// UserDeleted mean that user deleted account
	UserDeleted

	// Just unsupported status (for errors)
	userStatusUnsupported
)

// userStatuses is slice of User Statuses string representations
var userStatuses = [...]string{
	UserActive:  "active",
	UserDeleted: "deleted",
}

// String return UserStatus enum as a string
func (s UserStatus) String() string {
	return userStatuses[s]
}

func (s UserStatus) Proto() proto.UserStatus {
	switch s {
	case UserActive:
		return proto.UserStatus_UserActive
	case UserDeleted:
		// FIXME should be UserDeleted, not UserArchive
		return proto.UserStatus_UserArchive
	default:
		return proto.UserStatus_UserUnknown
	}
}

// UserStatusFromString return new UserStatus enum from given string
func UserStatusFromString(s string) (UserStatus, error) {
	for i, r := range userStatuses {
		if strings.EqualFold(s, r) {
			return UserStatus(i), nil
		}
	}
	return userStatusUnsupported, fmt.Errorf("invalid user status value %q", s)
}

func UserStatusFromProto(s proto.UserStatus) UserStatus {
	switch s {
	case proto.UserStatus_UserActive:
		return UserActive
	// FIXME should be UserDeleted, not UserArchive
	case proto.UserStatus_UserArchive:
		return UserDeleted
	default:
		return userStatusUnsupported
	}
}

func UserStatusFromProtoError(s proto.UserStatus) (UserStatus, error) {
	switch s {
	case proto.UserStatus_UserActive:
		return UserActive, nil
	// FIXME should be UserDeleted, not UserArchive
	case proto.UserStatus_UserArchive:
		return UserDeleted, nil
	default:
		return userStatusUnsupported, fmt.Errorf("invalid user status value %q", s.String())
	}
}
