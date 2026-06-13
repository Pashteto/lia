package categories

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

type mockRepo struct {
	getErr error
	list   []*models.Category
	byIDs  []*models.Category
}

func (m *mockRepo) List() ([]*models.Category, error) { return m.list, nil }
func (m *mockRepo) GetByIDs([]uuid.UUID) ([]*models.Category, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.byIDs, nil
}

func cat(slug string) *models.Category {
	id, _ := uuid.NewV4()
	return &models.Category{ID: id, Slug: slug, Label: slug}
}

func TestService_List(t *testing.T) {
	svc := NewService(&mockRepo{list: []*models.Category{cat("lecture")}})
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 category, got %d", len(got))
	}
}

func TestService_Validate_Empty(t *testing.T) {
	svc := NewService(&mockRepo{})
	got, err := svc.Validate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Validate(nil) returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil categories, got %v", got)
	}
}

func TestService_Validate_AllResolve(t *testing.T) {
	c := cat("lecture")
	svc := NewService(&mockRepo{byIDs: []*models.Category{c}})
	got, err := svc.Validate(context.Background(), []uuid.UUID{c.ID})
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resolved category, got %d", len(got))
	}
}

func TestService_Validate_UnknownID(t *testing.T) {
	svc := NewService(&mockRepo{byIDs: nil})
	unknown, _ := uuid.NewV4()
	_, err := svc.Validate(context.Background(), []uuid.UUID{unknown})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Validate_DeduplicatesRequestedIDs(t *testing.T) {
	c := cat("lecture")
	svc := NewService(&mockRepo{byIDs: []*models.Category{c}})
	_, err := svc.Validate(context.Background(), []uuid.UUID{c.ID, c.ID})
	if err != nil {
		t.Fatalf("expected duplicate ids to validate, got %v", err)
	}
}
