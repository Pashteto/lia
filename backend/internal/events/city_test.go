package events

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

func TestList_PassesCityIntoFilter(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)
	_, err := svc.List(context.Background(), "published", nil, nil, nil, models.CitySPB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listFilter.City != models.CitySPB {
		t.Fatalf("expected filter.City=%q, got %q", models.CitySPB, repo.listFilter.City)
	}
}

func TestList_EmptyCityMeansNoFilter(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)
	if _, err := svc.List(context.Background(), "published", nil, nil, nil, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listFilter.City != "" {
		t.Fatalf("expected empty filter.City, got %q", repo.listFilter.City)
	}
}

func TestList_InvalidCityRejected(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{}, 0)
	_, err := svc.List(context.Background(), "published", nil, nil, nil, "ekb")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_InheritsCityFromVenue(t *testing.T) {
	repo := &mockRepo{}
	venue := &models.Venue{ID: uuid.Must(uuid.NewV4()), Name: "Эрарта", City: models.CitySPB}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{resolved: venue}, 0)

	ev := validEventWithOrganizer()
	ev.VenueID = venue.ID
	if err := svc.Create(context.Background(), ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created.City != models.CitySPB {
		t.Fatalf("expected created city %q, got %q", models.CitySPB, repo.created.City)
	}
}

func TestCreate_ExplicitCityConflictingWithVenueRejected(t *testing.T) {
	venue := &models.Venue{ID: uuid.Must(uuid.NewV4()), Name: "Эрарта", City: models.CitySPB}
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{resolved: venue}, 0)

	ev := validEventWithOrganizer()
	ev.VenueID = venue.ID
	ev.City = models.CityMSK
	err := svc.Create(context.Background(), ev)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_NoVenueDefaultsToMSK(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)
	ev := validEventWithOrganizer()
	ev.City = ""
	if err := svc.Create(context.Background(), ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created.City != models.CityMSK {
		t.Fatalf("expected default city %q, got %q", models.CityMSK, repo.created.City)
	}
}

func TestCreate_NoVenueInvalidCityRejected(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockValidator{}, &mockVenueValidator{}, 0)
	ev := validEventWithOrganizer()
	ev.City = "МСК"
	if !errors.Is(svc.Create(context.Background(), ev), ErrInvalidInput) {
		t.Fatal("expected ErrInvalidInput for non-slug city")
	}
}

func TestUpdate_VenueChangeCarriesItsCity(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := publishedEvent(owner)
	ev.City = models.CityMSK
	venue := &models.Venue{ID: uuid.Must(uuid.NewV4()), Name: "Севкабель", City: models.CitySPB}
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{resolved: venue}, 0)

	_, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{VenueID: &venue.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated.City != models.CitySPB {
		t.Fatalf("expected updated city %q, got %q", models.CitySPB, repo.updated.City)
	}
}

func TestUpdate_VenueRemovalKeepsTheCity(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := publishedEvent(owner)
	ev.VenueID = uuid.Must(uuid.NewV4())
	ev.City = models.CitySPB
	// venues.Validate returns (nil, nil) for the zero id — venue removal.
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	zero := uuid.Nil
	if _, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{VenueID: &zero}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated.City != models.CitySPB {
		t.Fatalf("venue removal must keep city %q, got %q", models.CitySPB, repo.updated.City)
	}
}

func TestUpdate_ExplicitCityOnVenuelessEvent(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	ev := publishedEvent(owner)
	ev.VenueID = uuid.Nil
	ev.City = models.CityMSK
	repo := &mockRepo{get: ev}
	svc := NewService(repo, &mockValidator{}, &mockVenueValidator{}, 0)

	spb := models.CitySPB
	if _, err := svc.Update(context.Background(), ev.ID, owner, UpdateParams{City: &spb}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated.City != models.CitySPB {
		t.Fatalf("expected updated city %q, got %q", models.CitySPB, repo.updated.City)
	}
}
