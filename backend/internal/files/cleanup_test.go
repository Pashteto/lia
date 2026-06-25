package files

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// fakeCleanupRepo records calls for testing the Cleaner.
type fakeCleanupRepo struct {
	orphans []*models.File
	deleted []uuid.UUID
	listErr error
}

func (r *fakeCleanupRepo) Create(_ *models.File) error                { return nil }
func (r *fakeCleanupRepo) GetByID(_ uuid.UUID) (*models.File, error)  { return nil, ErrNotFound }
func (r *fakeCleanupRepo) ListOrphansOlderThan(_ time.Duration) ([]*models.File, error) {
	return r.orphans, r.listErr
}
func (r *fakeCleanupRepo) Delete(id uuid.UUID) error {
	r.deleted = append(r.deleted, id)
	return nil
}

// fakeCleanupStorage records Delete calls and optionally returns errors per key.
type fakeCleanupStorage struct {
	deleted  []string
	failKeys map[string]error
}

func (s *fakeCleanupStorage) Put(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}
func (s *fakeCleanupStorage) Get(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, nil
}
func (s *fakeCleanupStorage) Delete(_ context.Context, key string) error {
	if err, ok := s.failKeys[key]; ok {
		return err
	}
	s.deleted = append(s.deleted, key)
	return nil
}
func (s *fakeCleanupStorage) URL(_ string) string                             { return "" }
func (s *fakeCleanupStorage) Exists(_ context.Context, _ string) (bool, error) { return false, nil }

func TestCleaner_Run_DeletesBothOrphans(t *testing.T) {
	id1 := uuid.Must(uuid.NewV4())
	id2 := uuid.Must(uuid.NewV4())
	repo := &fakeCleanupRepo{
		orphans: []*models.File{
			{ID: id1, StorageKey: "uploads/a.png"},
			{ID: id2, StorageKey: "uploads/b.jpg"},
		},
	}
	store := &fakeCleanupStorage{}
	cleaner := NewCleaner(repo, store, 24*time.Hour)

	deleted, err := cleaner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("want deleted=2, got %d", deleted)
	}
	if len(store.deleted) != 2 {
		t.Fatalf("want 2 blob deletes, got %d", len(store.deleted))
	}
	if len(repo.deleted) != 2 {
		t.Fatalf("want 2 row deletes, got %d", len(repo.deleted))
	}
}

func TestCleaner_Run_StorageErrorSkippedNotFatal(t *testing.T) {
	id1 := uuid.Must(uuid.NewV4())
	id2 := uuid.Must(uuid.NewV4())
	repo := &fakeCleanupRepo{
		orphans: []*models.File{
			{ID: id1, StorageKey: "uploads/a.png"},
			{ID: id2, StorageKey: "uploads/b.jpg"},
		},
	}
	store := &fakeCleanupStorage{
		failKeys: map[string]error{
			"uploads/a.png": errors.New("storage failure"),
		},
	}
	cleaner := NewCleaner(repo, store, 24*time.Hour)

	deleted, err := cleaner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: storage error must not be fatal: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("want deleted=1 (one skipped), got %d", deleted)
	}
	// Only the second blob should be deleted
	if len(store.deleted) != 1 || store.deleted[0] != "uploads/b.jpg" {
		t.Fatalf("unexpected store.deleted: %v", store.deleted)
	}
	// Only the second row should be deleted
	if len(repo.deleted) != 1 || repo.deleted[0] != id2 {
		t.Fatalf("unexpected repo.deleted: %v", repo.deleted)
	}
}
