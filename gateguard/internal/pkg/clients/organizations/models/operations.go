package models

import (
	proto "gateguard/protocols/organizations"
)

// RollupOperationType represents different operations for rollups
type RollupOperationType int

const (
	RotView RollupOperationType = iota
	RotCreate
	RotUpdate
	RotDelete
)

// OrganizationOperationType represents different operations for organizations
type OrganizationOperationType int

const (
	OotView OrganizationOperationType = iota
	OotUpdate
	OotDelete
)

// ToProto converts the internal RollupOperationType model to the corresponding proto message
func (r RollupOperationType) ToProto() proto.RollupOperationType {
	switch r {
	case RotView:
		return proto.RollupOperationType_RotView
	case RotCreate:
		return proto.RollupOperationType_RotCreate
	case RotUpdate:
		return proto.RollupOperationType_RotUpdate
	case RotDelete:
		return proto.RollupOperationType_RotDelete
	default:
		return proto.RollupOperationType_RotUnknown
	}
}

// ToProto converts the internal OrganizationOperationType model to the corresponding proto message
func (o OrganizationOperationType) ToProto() proto.OrganizationOperationType {
	switch o {
	case OotView:
		return proto.OrganizationOperationType_OotView
	case OotUpdate:
		return proto.OrganizationOperationType_OotUpdate
	case OotDelete:
		return proto.OrganizationOperationType_OotDelete
	default:
		return proto.OrganizationOperationType_OotUnknown
	}
}
