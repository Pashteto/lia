package files

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/storage"
)

// ErrNotFound is returned when a requested file does not exist.
var ErrNotFound = errors.New("file not found")

// Service is the files business-logic interface.
type Service interface {
	// Register records file metadata (key, contentType, size, owner) in the DB
	// and returns the persisted File row. It does NOT upload bytes; the caller
	// must have already called storage.Storage.Put before calling Register.
	Register(ctx context.Context, key, contentType string, size int64, owner uuid.UUID) (*models.File, error)
	// Get returns the file metadata for the given id.
	Get(ctx context.Context, id uuid.UUID) (*models.File, error)
}

type service struct {
	repo  Repository
	store storage.Storage
}

// NewService creates a files service backed by the given repository and storage.
func NewService(repo Repository, store storage.Storage) Service {
	return &service{repo: repo, store: store}
}

func (s *service) Register(_ context.Context, key, ct string, size int64, owner uuid.UUID) (*models.File, error) {
	f := &models.File{
		ID:          uuid.Must(uuid.NewV4()),
		StorageKey:  key,
		ContentType: ct,
		Size:        size,
		OwnerUserID: owner,
	}
	if err := s.repo.Create(f); err != nil {
		return nil, fmt.Errorf("register file: %w", err)
	}
	return f, nil
}

func (s *service) Get(_ context.Context, id uuid.UUID) (*models.File, error) {
	f, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return f, nil
}
