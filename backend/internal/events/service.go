package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/pkg/logger"
)

// Domain errors. The HTTP layer maps these to status codes.
var (
	// ErrInvalidInput indicates the event failed validation.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound indicates no event matched the query.
	ErrNotFound = errors.New("not found")
)

// Service is the events business-logic interface.
type Service interface {
	// Create validates and persists a new event.
	Create(ctx context.Context, event *models.Event) error
	// GetByID returns a single event by its string UUID.
	GetByID(ctx context.Context, id string) (*models.Event, error)
	// List returns events, optionally filtered by status.
	List(ctx context.Context, status string) ([]*models.Event, error)
}

type service struct {
	repo Repository
}

// NewService creates an events service backed by the given repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(_ context.Context, event *models.Event) error {
	if event == nil {
		return fmt.Errorf("%w: event is required", ErrInvalidInput)
	}

	if err := event.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	if err := s.repo.Create(event); err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	logger.Log().Infof("event created via service: %s", event.ID)
	return nil
}

func (s *service) GetByID(_ context.Context, id string) (*models.Event, error) {
	parsed, err := uuid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid event id %q", ErrInvalidInput, id)
	}

	event, err := s.repo.GetByID(parsed)
	if err != nil {
		if wrapped := errors.Unwrap(err); wrapped != nil &&
			wrapped.Error() == "pg: no rows in result set" {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("get event by id: %w", err)
	}

	return event, nil
}

func (s *service) List(_ context.Context, status string) ([]*models.Event, error) {
	if status != "" {
		if _, err := models.EventStatusFromString(status); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
	}

	list, err := s.repo.List(ListFilter{Status: status})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return list, nil
}
