package models

import proto "gateguard/protocols/organizations"

// UserStatus represents the status of user in the organization
type UserStatus string

const (
	UserStatusPending  UserStatus = "pending"
	UserStatusAccepted UserStatus = "accepted"
)

// ConvertProtoUserStatusToModel converts proto UserStatus to model UserStatus
func ConvertProtoUserStatusToModel(status proto.UserStatus) UserStatus {
	switch status {
	case proto.UserStatus_UserStatusPending:
		return UserStatusPending
	case proto.UserStatus_UserStatusActive:
		return UserStatusAccepted
	default:
		return ""
	}
}

// ConvertModelUserStatusToProto converts model UserStatus to proto UserStatus
func ConvertModelUserStatusToProto(status UserStatus) proto.UserStatus {
	switch status {
	case UserStatusPending:
		return proto.UserStatus_UserStatusPending
	case UserStatusAccepted:
		return proto.UserStatus_UserStatusActive
	default:
		return proto.UserStatus_UserStatusUnknown
	}
}
