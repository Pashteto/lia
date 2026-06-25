package repository

import (
	"context"

	"gateguard/internal/models"
)

//go:generate ../../bin/mockery --name IRepository

// IRepository is repository storage interface for invitations DB
type IRepository interface {
	// CreateUser is
	CreateUser(ctx context.Context, model *models.User) error

	// GetUser is
	GetUser(ctx context.Context, model *models.User, getter UserGetter) error

	// UpdateUserBy is
	UpdateUserBy(ctx context.Context, model *models.User, getter UserGetter, columns ...string) error

	// AllUsers is
	AllUsers(ctx context.Context) (users []*models.User, err error)

	// CreateInvitation is
	CreateInvitation(ctx context.Context, model *models.Invitation) error

	// GetInvitation is
	GetInvitation(ctx context.Context, model *models.Invitation, getter InvitationGetter) error

	// AllInvitations is
	AllInvitations(ctx context.Context, filter *AllInvitationsFilter, options ...QueryOption) (invitations []*models.Invitation, hasMore bool, err error)

	// UpdateInvitationBy is
	UpdateInvitationBy(ctx context.Context, model *models.Invitation, getter InvitationGetter, columns ...string) error
}
