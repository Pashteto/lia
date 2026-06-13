package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/categories"
	"github.com/Pashteto/lia/internal/models"
)

// mockRepo is an in-memory Repository for tests.
type mockRepo struct {
	created *models.Event
	getErr  error
	get     *models.Event
	list    []*models.Event
}

func (m *mockRepo) Create(event *models.Event) error {
	m.created = event
	return nil
}

func (m *mockRepo) GetByID(uuid.UUID) (*models.Event, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.get, nil
}

func (m *mockRepo) List(ListFilter) ([]*models.Event, error) {
	return m.list, nil
}

// mockValidator is an in-memory CategoryValidator.
type mockValidator struct {
	err      error
	resolved []*models.Category
}

func (m *mockValidator) Validate(context.Context, []uuid.UUID) ([]*models.Category, error) {
	return m.resolved, m.err
}

func validEvent() *models.Event {
	return &models.Event{
		Title:    "Память и архив",
		Status:   models.EventPublished,
		StartsAt: time.Now().Add(24 * time.Hour),
	}
}

func TestService_Create(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{})

	if err := svc.Create(context.Background(), validEvent()); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected repository.Create to be called")
	}
}

func TestService_Create_WithCategories(t *testing.T) {
	id, _ := uuid.NewV4()
	resolved := []*models.Category{{ID: id, Slug: "lecture", Label: "Лекции"}}
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{resolved: resolved})

	ev := validEvent()
	ev.CategoryIDs = []uuid.UUID{id}
	if err := svc.Create(context.Background(), ev); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if len(repo.created.Categories) != 1 {
		t.Fatalf("expected resolved categories on the event, got %d", len(repo.created.Categories))
	}
}

func TestService_Create_UnknownCategory(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{err: categories.ErrInvalidInput})

	ev := validEvent()
	bad, _ := uuid.NewV4()
	ev.CategoryIDs = []uuid.UUID{bad}
	err := svc.Create(context.Background(), ev)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Create_InvalidInput(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{})

	err := svc.Create(context.Background(), &models.Event{}) // missing title/starts_at
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_GetByID_InvalidUUID(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{})

	_, err := svc.GetByID(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_InvalidStatus(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{})

	_, err := svc.List(context.Background(), "bogus")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_OK(t *testing.T) {
	repo := &mockRepo{list: []*models.Event{validEvent()}}
	svc := NewService(repo, &mockValidator{})

	got, err := svc.List(context.Background(), "published")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
}
