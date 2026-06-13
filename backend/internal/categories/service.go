package categories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// ErrInvalidInput indicates a category reference failed validation.
var ErrInvalidInput = errors.New("invalid input")

// Service is the categories business-logic interface.
type Service interface {
	// List returns the full curated taxonomy, ordered by sort_order.
	List(ctx context.Context) ([]*models.Category, error)
	// Validate resolves the given category ids, returning the matching
	// categories. Returns ErrInvalidInput if any id does not exist. An empty
	// input is valid and resolves to nil.
	Validate(ctx context.Context, ids []uuid.UUID) ([]*models.Category, error)
}

type service struct {
	repo Repository
}

// NewService creates a categories service backed by the given repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) List(_ context.Context) ([]*models.Category, error) {
	list, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return list, nil
}

func (s *service) Validate(_ context.Context, ids []uuid.UUID) ([]*models.Category, error) {
	unique := dedupeUUIDs(ids)
	if len(unique) == 0 {
		return nil, nil
	}

	found, err := s.repo.GetByIDs(unique)
	if err != nil {
		return nil, fmt.Errorf("resolve categories: %w", err)
	}
	if len(found) != len(unique) {
		foundSet := make(map[uuid.UUID]struct{}, len(found))
		for _, c := range found {
			foundSet[c.ID] = struct{}{}
		}
		missing := make([]string, 0, len(unique)-len(found))
		for _, id := range unique {
			if _, ok := foundSet[id]; !ok {
				missing = append(missing, id.String())
			}
		}
		return nil, fmt.Errorf("%w: unknown category_ids: %s", ErrInvalidInput, strings.Join(missing, ", "))
	}
	return found, nil
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
