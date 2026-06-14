package venues

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// ErrInvalidInput indicates a venue failed validation or a referenced id is unknown.
var ErrInvalidInput = errors.New("invalid input")

// Service is the venues business-logic interface.
type Service interface {
	// Search returns venues matching q (see Repository.Search).
	Search(ctx context.Context, q string, limit int) ([]*models.Venue, error)
	// GetByID returns a single venue by id.
	GetByID(ctx context.Context, id uuid.UUID) (*models.Venue, error)
	// Create validates (name required), trims the name, and find-or-creates.
	Create(ctx context.Context, v *models.Venue) (*models.Venue, error)
	// Validate resolves a non-zero venue id; returns (nil,nil) for the zero id
	// (meaning "no venue"), or ErrInvalidInput if a non-zero id does not exist.
	Validate(ctx context.Context, id uuid.UUID) (*models.Venue, error)
}

type service struct {
	repo Repository
}

// NewService creates a venues service backed by the given repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Search(_ context.Context, q string, limit int) ([]*models.Venue, error) {
	list, err := s.repo.Search(q, limit)
	if err != nil {
		return nil, fmt.Errorf("search venues: %w", err)
	}
	return list, nil
}

func (s *service) GetByID(_ context.Context, id uuid.UUID) (*models.Venue, error) {
	v, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get venue: %w", err)
	}
	return v, nil
}

func (s *service) Create(_ context.Context, v *models.Venue) (*models.Venue, error) {
	if v == nil {
		return nil, fmt.Errorf("%w: venue is required", ErrInvalidInput)
	}
	v.Name = strings.TrimSpace(v.Name)
	if v.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	v.Address = strings.TrimSpace(v.Address)
	v.Metro = strings.TrimSpace(v.Metro)
	v.District = strings.TrimSpace(v.District)

	created, err := s.repo.FindOrCreateByName(v)
	if err != nil {
		return nil, fmt.Errorf("create venue: %w", err)
	}
	return created, nil
}

func (s *service) Validate(_ context.Context, id uuid.UUID) (*models.Venue, error) {
	if id == uuid.Nil {
		return nil, nil
	}
	v, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, fmt.Errorf("%w: venue %s does not exist", ErrInvalidInput, id)
		}
		return nil, fmt.Errorf("validate venue: %w", err)
	}
	return v, nil
}
