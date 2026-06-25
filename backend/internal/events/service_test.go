package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/categories"
	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/venues"
)

// mockRepo is an in-memory Repository for tests.
type mockRepo struct {
	created      *models.Event
	getErr       error
	get          *models.Event
	list         []*models.Event
	nearbyResult []*NearbyResult
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

func (m *mockRepo) Nearby(lat, lon float64, limit int) ([]*NearbyResult, error) {
	return m.nearbyResult, nil
}

// mockValidator is an in-memory CategoryValidator.
type mockValidator struct {
	err      error
	resolved []*models.Category
}

func (m *mockValidator) Validate(context.Context, []uuid.UUID) ([]*models.Category, error) {
	return m.resolved, m.err
}

// mockVenueValidator is an in-memory VenueValidator.
type mockVenueValidator struct {
	resolved *models.Venue
	err      error
}

func (m *mockVenueValidator) Validate(context.Context, uuid.UUID) (*models.Venue, error) {
	return m.resolved, m.err
}

// newServiceWithMock is a convenience helper used by Nearby tests.
func newServiceWithMock(repo *mockRepo) Service {
	return NewService(repo, &mockValidator{}, &mockVenueValidator{})
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
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{})

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
	svc := NewService(repo, &mockValidator{resolved: resolved}, &mockVenueValidator{})

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
	svc := NewService(repo, &mockValidator{err: categories.ErrInvalidInput}, &mockVenueValidator{})

	ev := validEvent()
	bad, _ := uuid.NewV4()
	ev.CategoryIDs = []uuid.UUID{bad}
	err := svc.Create(context.Background(), ev)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Create_InvalidInput(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{})

	err := svc.Create(context.Background(), &models.Event{}) // missing title/starts_at
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_GetByID_InvalidUUID(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{})

	_, err := svc.GetByID(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_InvalidStatus(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{})

	_, err := svc.List(context.Background(), "bogus")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_OK(t *testing.T) {
	repo := &mockRepo{list: []*models.Event{validEvent()}}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{})

	got, err := svc.List(context.Background(), "published")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
}

func TestService_Create_WithVenue(t *testing.T) {
	id, _ := uuid.NewV4()
	resolved := &models.Venue{ID: id, Name: "Винзавод"}
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{resolved: resolved})

	ev := validEvent()
	ev.VenueID = id
	if err := svc.Create(context.Background(), ev); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created.Venue == nil || repo.created.Venue.ID != id {
		t.Fatalf("expected resolved venue on the event, got %v", repo.created.Venue)
	}
}

func TestService_Create_UnknownVenue(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{err: venues.ErrInvalidInput})

	ev := validEvent()
	bad, _ := uuid.NewV4()
	ev.VenueID = bad
	err := svc.Create(context.Background(), ev)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Nearby_RequiresCoords(t *testing.T) {
	svc := newServiceWithMock(&mockRepo{})
	if _, err := svc.Nearby(context.Background(), nil, nil, 10); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput when coords missing, got %v", err)
	}
}

func TestService_Nearby_OK(t *testing.T) {
	repo := &mockRepo{nearbyResult: []*NearbyResult{{Event: &models.Event{Title: "X"}, DistanceM: 1200}}}
	svc := newServiceWithMock(repo)
	lat, lon := 55.75, 37.62
	got, err := svc.Nearby(context.Background(), &lat, &lon, 10)
	if err != nil || len(got) != 1 || got[0].DistanceM != 1200 {
		t.Fatalf("unexpected: %v %v", got, err)
	}
}

func TestService_Create_WithCoverFileID(t *testing.T) {
	coverID, _ := uuid.NewV4()
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{})

	ev := validEvent()
	ev.CoverFileID = coverID
	if err := svc.Create(context.Background(), ev); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected repository.Create to be called")
	}
	if repo.created.CoverFileID != coverID {
		t.Fatalf("expected CoverFileID %s on created event, got %s", coverID, repo.created.CoverFileID)
	}
}
