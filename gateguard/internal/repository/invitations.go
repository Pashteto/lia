package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"gateguard/internal/models"
)

func (r *Repository) CreateInvitation(ctx context.Context, model *models.Invitation) error {
	if _, err := r.transactionFactory.Transaction(ctx).Model(model).Returning("*").Insert(); err != nil {
		var pgErr pg.Error
		if errors.As(err, &pgErr) && pgErr.IntegrityViolation() {
			return fmt.Errorf("invitation already exists or breaks the metastructure: %w", pgErr)
		}
		return fmt.Errorf("insert invitation %s into db: %w", model.Invitee, err)
	}
	return nil
}

func (r *Repository) GetInvitation(ctx context.Context, model *models.Invitation, getter InvitationGetter) error {
	query := r.transactionFactory.Transaction(ctx).Model(model).Column("invitation.*")

	if err := getter.Get(query, model); err != nil {
		return fmt.Errorf("parse getter: %w", err)
	}

	if err := query.Select(); err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return fmt.Errorf("invitation not found: %w", ErrInvitationNotFound)
		}
		return fmt.Errorf("get invitation %s from database by %s: %w", model.Invitee, getter.String(), err)
	}

	return nil
}

func (r *Repository) UpdateInvitationBy(ctx context.Context, model *models.Invitation, getter InvitationGetter, columns ...string) error {
	query := r.transactionFactory.Transaction(ctx).Model(model).Column(columns...)

	if err := getter.Get(query, model); err != nil {
		return fmt.Errorf("parse getter: %w", err)
	}

	result, err := query.Update()
	if err != nil {
		return fmt.Errorf("update invitation in database by %s: %w", getter.String(), err)
	}

	if result.RowsAffected() == 0 {
		return ErrInvitationNotFound
	}

	return nil
}

func (r *Repository) AllInvitations(ctx context.Context, filter *AllInvitationsFilter, options ...QueryOption) ([]*models.Invitation, bool, error) {
	var invitations []*models.Invitation

	query := r.transactionFactory.Transaction(ctx).Model(&invitations).Column("invitation.*")

	ApplyQueryOptions(query, options...)

	if filter != nil {
		if filter.Inviter != nil && *filter.Inviter != "" {
			query.Where("inviter = ?", filter.Inviter)
		}

		if filter.Invitee != nil && *filter.Invitee != "" {
			query.Where("invitee = ?", filter.Invitee)
		}

		if filter.Organization != nil && *filter.Organization != uuid.Nil {
			query.Where("organization = ?", filter.Organization.String())
		}

		if len(filter.Statuses) > 0 {
			statuses := make([]string, len(filter.Statuses))
			for i, status := range filter.Statuses {
				statuses[i] = status.String()
			}
			query.Where("status IN (?)", pg.In(statuses))
		}

		if filter.DateFrom != nil {
			query.Where("created_at >= ?", filter.DateFrom)
		}

		if filter.DateTo != nil {
			query.Where("created_at <= ?", filter.DateTo)
		}

		if filter.Limit > 0 {
			query.Limit(int(filter.Limit + 1)) // get one more row to see if there is something there
		}

		query.Offset(int(filter.Offset))
	}

	if err := query.Select(); err != nil {
		return nil, false, fmt.Errorf("select invitations: %w", err)
	}

	hasMore := false
	if filter.Limit > 0 && len(invitations) > int(filter.Limit) {
		hasMore = true
		invitations = invitations[:filter.Limit] // trim the last record
	}

	return invitations, hasMore, nil
}
