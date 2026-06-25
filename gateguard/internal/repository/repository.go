package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/gateway-fm/scriptorium/transactions"
	"github.com/go-pg/pg/v10"

	"gateguard/internal/models"
)

// Repository is
type Repository struct {
	transactionFactory transactions.TransactionFactory
}

// NewRepository is
func NewRepository(trf transactions.TransactionFactory) IRepository {
	return &Repository{transactionFactory: trf}
}

func (r *Repository) CreateUser(ctx context.Context, model *models.User) error {
	if _, err := r.transactionFactory.Transaction(ctx).Model(model).Returning("*").Insert(); err != nil {
		return fmt.Errorf("insert user %s into db: %w", model.Email, err)
	}

	return nil
}

// GetUser fetches a user from the database based on the provided criteria.
func (r *Repository) GetUser(ctx context.Context, model *models.User, getter UserGetter) error {
	query := r.transactionFactory.Transaction(ctx).Model(model).Column("user.*")

	if err := getter.Get(query, model); err != nil {
		return fmt.Errorf("parse getter: %w", err)
	}

	if err := query.Select(); err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("get user %s from database by %s: %w", model.Email, getter.String(), err)
	}

	return nil
}

func (r *Repository) UpdateUserBy(ctx context.Context, model *models.User, getter UserGetter, columns ...string) error {
	query := r.transactionFactory.Transaction(ctx).Model(model).Column(columns...)

	if err := getter.Get(query, model); err != nil {
		return fmt.Errorf("parse getter: %w", err)
	}

	if _, err := query.Update(); err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("update user in database by %s: %w", getter.String(), err)
	}
	return nil
}

func (r *Repository) AllUsers(ctx context.Context) ([]*models.User, error) {
	var users []*models.User

	query := r.transactionFactory.Transaction(ctx).Model(&users).Column("user.*")

	query.Where("status != 'deleted'")

	if err := query.Select(); err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}

	return users, nil
}
