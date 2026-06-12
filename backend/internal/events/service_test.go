package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

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

func validEvent() *models.Event {
	return &models.Event{
		Title:    "Память и архив",
		Status:   models.EventPublished,
		StartsAt: time.Now().Add(24 * time.Hour),
	}
}

func TestService_Create(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	if err := svc.Create(context.Background(), validEvent()); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected repository.Create to be called")
	}
}

func TestService_Create_InvalidInput(t *testing.T) {
	svc := NewService(&mockRepo{})

	err := svc.Create(context.Background(), &models.Event{}) // missing title/starts_at
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_GetByID_InvalidUUID(t *testing.T) {
	svc := NewService(&mockRepo{})

	_, err := svc.GetByID(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_InvalidStatus(t *testing.T) {
	svc := NewService(&mockRepo{})

	_, err := svc.List(context.Background(), "bogus")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_OK(t *testing.T) {
	repo := &mockRepo{list: []*models.Event{validEvent()}}
	svc := NewService(repo)

	got, err := svc.List(context.Background(), "published")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
}
