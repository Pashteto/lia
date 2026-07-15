package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	proto "gateguard/protocols/gateguard"
)

type User struct {
	tableName         struct{} `pg:"users,discard_unknown_columns"`
	UUID              uuid.UUID
	Email             string
	Name              string
	Avatar            string
	IP                uint32      `pg:",use_zero"`
	Status            UserStatus  `pg:"-"`
	StatusSQL         string      `pg:"status"`
	Role              UserRole    `pg:"-"`
	RoleSQL           string      `pg:"role"`
	RefCode           string      `pg:"-"`
	Organizations     []uuid.UUID `pg:"organizations,array"`
	PreferredStacks   []int64     `pg:"preferred_stacks,array"`
	TrialUsed         bool        `pg:",use_zero"`
	// Credential + email-verification fields. PasswordHash is NEVER serialized
	// to proto / API responses (see Proto(), which omits it).
	PasswordHash            string    `pg:"password_hash"`
	EmailVerified           bool      `pg:"email_verified,use_zero"`
	EmailVerificationToken  string    `pg:"email_verification_token"`
	EmailVerificationSentAt time.Time `pg:"email_verification_sent_at"`
	CreatedOrRestored       bool      `pg:"-"`
	UpdatedAt               time.Time
	CreatedAt               time.Time
	DeletedAt               time.Time
}

// BeforeInsert is Pre-Insert hook that creates new
// UUID for User and cast Status to string
func (u *User) BeforeInsert(ctx context.Context) (context.Context, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		return ctx, fmt.Errorf("create UUID for new User: %w", err)
	}

	u.UUID = guid
	u.StatusSQL = u.Status.String()
	u.RoleSQL = u.Role.String()

	return ctx, nil
}

// BeforeUpdate is Pre-Update hook that cast
// UserStatus enum to string
func (u *User) BeforeUpdate(ctx context.Context) (context.Context, error) {
	u.StatusSQL = u.Status.String()
	u.RoleSQL = u.Role.String()

	return ctx, nil
}

// AfterSelect is Post-Select hook that cast
// SqlStatus to UserStatus enum
func (u *User) AfterSelect(ctx context.Context) error {
	status, err := UserStatusFromString(u.StatusSQL)
	if err != nil {
		return fmt.Errorf("parse user status: %w", err)
	}

	u.Status = status
	u.Role = UserRoleFromString(u.RoleSQL)

	return nil
}

// ToJWT format user model to JWT claims
func (u *User) ToJWT() map[string]interface{} {
	jwtMap := map[string]interface{}{
		"uuid":  u.UUID,
		"email": u.Email,
		"name":  u.Name,
	}

	return jwtMap
}

func (u *User) Proto() *proto.User {
	orgs := make([][]byte, len(u.Organizations))
	for i, org := range u.Organizations {
		orgs[i] = org.Bytes()
	}

	return &proto.User{
		Uuid:            u.UUID.Bytes(),
		Email:           u.Email,
		Name:            u.Name,
		Avatar:          u.Avatar,
		Status:          u.Status.Proto(),
		Created:         u.CreatedAt.Unix(),
		Ip:              u.IP,
		RefCode:         u.RefCode,
		Role:            u.Role.Proto(),
		Organizations:   orgs,
		PreferredStacks: u.PreferredStacks,
		TrialUsed:       u.TrialUsed,
		EmailVerified:   u.EmailVerified,
	}
}

func UserFromProto(pb *proto.User) *User {
	orgs := make([]uuid.UUID, len(pb.Organizations))
	for i, org := range pb.Organizations {
		orgs[i] = uuid.FromBytesOrNil(org)
	}

	return &User{
		UUID:          uuid.FromBytesOrNil(pb.Uuid),
		Email:         pb.Email,
		Name:          pb.Name,
		Avatar:        pb.Avatar,
		Status:        UserStatusFromProto(pb.Status),
		IP:            pb.Ip,
		RefCode:       pb.RefCode,
		Role:          UserRoleFromProto(pb.Role),
		Organizations: orgs,
		TrialUsed:     pb.TrialUsed,
		EmailVerified: pb.EmailVerified,
	}
}

func UsersToProto(users []*User) (pb []*proto.User) {
	for _, u := range users {
		pb = append(pb, u.Proto())
	}
	return
}
