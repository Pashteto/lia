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
	created          *models.Event
	getErr           error
	get              *models.Event
	list             []*models.Event
	nearbyResult     []*NearbyResult
	countByOrganizer int
	countErr         error
	countSinceArg    time.Time // captured arg from CountByOrganizerSince
	updated          *models.Event
	updateErr        error
	listFilter       ListFilter
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

func (m *mockRepo) List(f ListFilter) ([]*models.Event, error) {
	m.listFilter = f
	return m.list, nil
}

func (m *mockRepo) Update(event *models.Event) error {
	m.updated = event
	return m.updateErr
}

func (m *mockRepo) Nearby(lat, lon float64, limit int) ([]*NearbyResult, error) {
	return m.nearbyResult, nil
}

func (m *mockRepo) CountByOrganizerSince(_ uuid.UUID, since time.Time) (int, error) {
	m.countSinceArg = since
	return m.countByOrganizer, m.countErr
}

func (m *mockRepo) SetCapacityTx(uuid.UUID, *int) (int, error) {
	return 0, nil
}

func (m *mockRepo) WriteEditAudit(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
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
	return NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)
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
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

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
	svc := NewService(repo, &mockValidator{resolved: resolved}, &mockVenueValidator{}, 0)

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
	svc := NewService(repo, &mockValidator{err: categories.ErrInvalidInput}, &mockVenueValidator{}, 0)

	ev := validEvent()
	bad, _ := uuid.NewV4()
	ev.CategoryIDs = []uuid.UUID{bad}
	err := svc.Create(context.Background(), ev)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Create_InvalidInput(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{}, 0)

	err := svc.Create(context.Background(), &models.Event{}) // missing title/starts_at
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_GetByID_InvalidUUID(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{}, 0)

	_, err := svc.GetByID(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_InvalidStatus(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{}, 0)

	_, err := svc.List(context.Background(), "bogus", nil, nil, nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_List_OK(t *testing.T) {
	repo := &mockRepo{list: []*models.Event{validEvent()}}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	got, err := svc.List(context.Background(), "published", nil, nil, nil)
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
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{resolved: resolved}, 0)

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
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{err: venues.ErrInvalidInput}, 0)

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
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

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

// validEventWithOrganizer returns a valid event with a non-nil OrganizerID.
func validEventWithOrganizer() *models.Event {
	ev := validEvent()
	id, _ := uuid.NewV4()
	ev.OrganizerID = id
	return ev
}

func TestCreate_UnderLimit_OK(t *testing.T) {
	// count=9, limit=10 → should succeed
	repo := &mockRepo{countByOrganizer: 9}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 10)
	if err := svc.Create(context.Background(), validEventWithOrganizer()); err != nil {
		t.Fatalf("expected no error at count=9/limit=10, got: %v", err)
	}
}

func TestCreate_AtLimit_ReturnsErrQuota(t *testing.T) {
	// count=10, limit=10 → must return ErrQuotaExceeded
	repo := &mockRepo{countByOrganizer: 10}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 10)
	err := svc.Create(context.Background(), validEventWithOrganizer())
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded at count=10/limit=10, got: %v", err)
	}
}

func TestCreate_LimitZero_Unlimited(t *testing.T) {
	// limit=0 → quota check is disabled, even if count is very high
	repo := &mockRepo{countByOrganizer: 9999}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)
	if err := svc.Create(context.Background(), validEventWithOrganizer()); err != nil {
		t.Fatalf("expected no error when limit=0 (unlimited), got: %v", err)
	}
}

func TestStartOfMonthMoscow_ReturnsFirstDayMidnight(t *testing.T) {
	// 2026-06-25 15:42:00 UTC → 2026-06-01 00:00:00 Europe/Moscow
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("load Moscow tz: %v", err)
	}
	input := time.Date(2026, 6, 25, 15, 42, 0, 0, time.UTC)
	got := startOfMonthMoscow(input)
	want := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("startOfMonthMoscow(%v) = %v, want %v", input, got, want)
	}
}

// TestCreate_PriorMonthEventNotCounted asserts that the since boundary passed to
// CountByOrganizerSince is exactly startOfMonthMoscow(now) — i.e. the count
// window starts at the first of the current calendar month in Moscow time, not
// 30 days back.  It also verifies that when the repo returns 0, Create succeeds
// even when the limit is 1 (simulating that a prior-month event is excluded).
func TestCreate_PriorMonthEventNotCounted(t *testing.T) {
	repo := &mockRepo{countByOrganizer: 0} // prior-month event excluded by since
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 1)

	beforeCall := time.Now()
	if err := svc.Create(context.Background(), validEventWithOrganizer()); err != nil {
		t.Fatalf("expected success when count=0/limit=1 (prior month excluded), got: %v", err)
	}
	afterCall := time.Now()

	// The since argument must equal startOfMonthMoscow of the moment Create ran.
	// We bound the expected value between the two calls to tolerate clock ticks.
	wantLow := startOfMonthMoscow(beforeCall)
	wantHigh := startOfMonthMoscow(afterCall)

	got := repo.countSinceArg
	if got.IsZero() {
		t.Fatal("CountByOrganizerSince was not called — quota check skipped unexpectedly")
	}
	// wantLow and wantHigh are always the same value unless the test straddles a
	// month boundary (astronomically unlikely), so a simple Equal check suffices.
	if !got.Equal(wantLow) && !got.Equal(wantHigh) {
		t.Errorf("since = %v, want startOfMonthMoscow(now) = %v", got, wantLow)
	}

	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("load Moscow tz: %v", err)
	}
	moscowSince := got.In(loc)
	if moscowSince.Day() != 1 || moscowSince.Hour() != 0 || moscowSince.Minute() != 0 || moscowSince.Second() != 0 {
		t.Errorf("since in Moscow time must be 1st at 00:00:00, got %v", moscowSince)
	}
}

func ownedDraft(owner uuid.UUID) *models.Event {
	return &models.Event{
		ID:          uuid.Must(uuid.NewV4()),
		OrganizerID: owner,
		Title:       "Draft",
		Status:      models.EventDraft,
		StartsAt:    time.Now().Add(24 * time.Hour),
	}
}

func TestService_Update_NonOwner_ReturnsNotFound(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	other := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	_, err := svc.Update(context.Background(), ev.ID, other, UpdateParams{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Update_PublishedIsLocked(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	ev.Status = models.EventPublished
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	_, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{})
	if !errors.Is(err, ErrNotEditable) {
		t.Fatalf("expected ErrNotEditable, got %v", err)
	}
}

func TestService_Update_AppliesOnlyProvidedFields(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	ev.Description = "keep me"
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	newTitle := "Updated Title"
	if _, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{Title: &newTitle}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated.Title != "Updated Title" {
		t.Fatalf("title not applied: %q", repo.updated.Title)
	}
	if repo.updated.Description != "keep me" {
		t.Fatalf("omitted field not preserved: %q", repo.updated.Description)
	}
}

func TestService_Update_PublishSetsPublishedAt(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	published := "published"
	if _, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{Status: &published}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated.Status != models.EventPublished {
		t.Fatalf("status not set to published: %v", repo.updated.Status)
	}
	if repo.updated.PublishedAt == nil {
		t.Fatal("expected PublishedAt to be set on publish")
	}
}

func TestService_Update_RejectsNonSettableStatus(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := ownedDraft(owner)
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	pending := "pending_review"
	_, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{Status: &pending})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestList_PassesOrganizerOwnerIDIntoFilter(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)
	owner := uuid.Must(uuid.NewV4())
	_, _ = svc.List(context.Background(), "published", nil, nil, &owner)
	if len(repo.listFilter.OrganizerIDs) != 1 || repo.listFilter.OrganizerIDs[0] != owner {
		t.Fatalf("expected OrganizerIDs=[%s], got %v", owner, repo.listFilter.OrganizerIDs)
	}
}
