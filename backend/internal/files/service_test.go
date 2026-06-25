package files

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/Pashteto/lia/internal/models"
)

type fakeRepo struct{ created *models.File }

func (f *fakeRepo) Create(file *models.File) error { f.created = file; return nil }
func (f *fakeRepo) GetByID(id uuid.UUID) (*models.File, error) {
	if f.created != nil && f.created.ID == id {
		return f.created, nil
	}
	return nil, ErrNotFound
}
func (f *fakeRepo) ListOrphansOlderThan(_ time.Duration) ([]*models.File, error) {
	return nil, nil
}
func (f *fakeRepo) Delete(_ uuid.UUID) error { return nil }

func TestRegister_PersistsAndReturnsFile(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)
	owner := uuid.Must(uuid.NewV4())
	file, err := svc.Register(context.Background(), "uploads/a.png", "image/png", 123, owner)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if file.StorageKey != "uploads/a.png" || file.OwnerUserID != owner || file.Size != 123 {
		t.Fatalf("unexpected file: %+v", file)
	}
	if repo.created == nil {
		t.Fatal("repo.Create not called")
	}
}

func TestGet_ReturnsRegisteredFile(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)
	owner := uuid.Must(uuid.NewV4())
	registered, err := svc.Register(context.Background(), "uploads/b.jpg", "image/jpeg", 456, owner)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, err := svc.Get(context.Background(), registered.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != registered.ID {
		t.Fatalf("got wrong file: want %s, got %s", registered.ID, got.ID)
	}
}

func TestGet_NotFound(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)
	_, err := svc.Get(context.Background(), uuid.Must(uuid.NewV4()))
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
