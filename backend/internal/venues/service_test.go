package venues

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

//nolint:govet // fieldalignment false positive: all 4-field orderings produce 56B
type mockRepo struct {
	searchResult []*models.Venue
	getErr       error
	getResult    *models.Venue
	created      *models.Venue
	updated      *models.Venue
	updateErr    error
}

func (m *mockRepo) Search(string, int) ([]*models.Venue, error) { return m.searchResult, nil }
func (m *mockRepo) GetByID(uuid.UUID) (*models.Venue, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getResult, nil
}
func (m *mockRepo) GetByIDs([]uuid.UUID) ([]*models.Venue, error) { return nil, nil }
func (m *mockRepo) FindOrCreateByName(v *models.Venue) (*models.Venue, error) {
	m.created = v
	return v, nil
}
func (m *mockRepo) Update(v *models.Venue) error { m.updated = v; return m.updateErr }

func venue(name string) *models.Venue {
	id, _ := uuid.NewV4()
	return &models.Venue{ID: id, Name: name}
}

func TestService_Search(t *testing.T) {
	svc := NewService(&mockRepo{searchResult: []*models.Venue{venue("Винзавод")}})
	got, err := svc.Search(context.Background(), "вин", 0)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 venue, got %d", len(got))
	}
}

func TestService_Create_OK(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	got, err := svc.Create(context.Background(), &models.Venue{Name: "  Винзавод  ", Metro: "Чкаловская"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if got == nil || repo.created == nil {
		t.Fatal("expected venue created")
	}
	if repo.created.Name != "Винзавод" {
		t.Fatalf("expected trimmed name, got %q", repo.created.Name)
	}
}

func TestService_Create_EmptyName(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Create(context.Background(), &models.Venue{Name: "   "})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Validate_Zero(t *testing.T) {
	svc := NewService(&mockRepo{})
	got, err := svc.Validate(context.Background(), uuid.Nil)
	if err != nil || got != nil {
		t.Fatalf("expected (nil,nil) for zero id, got (%v,%v)", got, err)
	}
}

func TestService_Validate_Unknown(t *testing.T) {
	// Mirror production: the repo wraps pg.ErrNoRows with %w.
	svc := NewService(&mockRepo{getErr: fmt.Errorf("get venue from db: %w", pg.ErrNoRows)})
	id, _ := uuid.NewV4()
	_, err := svc.Validate(context.Background(), id)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Validate_OK(t *testing.T) {
	v := venue("Винзавод")
	svc := NewService(&mockRepo{getResult: v})
	got, err := svc.Validate(context.Background(), v.ID)
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if got == nil || got.ID != v.ID {
		t.Fatalf("expected resolved venue, got %v", got)
	}
}

func TestService_Create_RejectsHalfCoords(t *testing.T) {
	svc := NewService(&mockRepo{})
	lat := 55.75
	_, err := svc.Create(context.Background(), &models.Venue{Name: "X", Lat: &lat})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for lat without lon, got %v", err)
	}
}

func TestService_Create_RejectsOutOfRange(t *testing.T) {
	svc := NewService(&mockRepo{})
	lat, lon := 91.0, 10.0
	_, err := svc.Create(context.Background(), &models.Venue{Name: "X", Lat: &lat, Lon: &lon})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for lat>90, got %v", err)
	}
}

func TestService_Create_AcceptsValidCoords(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	lat, lon := 55.75, 37.62
	if _, err := svc.Create(context.Background(), &models.Venue{Name: "X", Lat: &lat, Lon: &lon}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created == nil || repo.created.Lat == nil || *repo.created.Lat != 55.75 {
		t.Fatal("expected coords passed through to repo")
	}
}

func TestService_Update_SetsCoords(t *testing.T) {
	existing := &models.Venue{Name: "Hall"}
	repo := &mockRepo{getResult: existing}
	svc := NewService(repo)
	id, _ := uuid.NewV4()
	lat, lon := 55.75, 37.62
	got, err := svc.Update(context.Background(), id, "", "", "", "", &lat, &lon)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lat == nil || *got.Lat != 55.75 || repo.updated == nil {
		t.Fatal("expected coords set and update called")
	}
}

func TestService_Update_RejectsOutOfRange(t *testing.T) {
	repo := &mockRepo{getResult: &models.Venue{Name: "Hall"}}
	svc := NewService(repo)
	id, _ := uuid.NewV4()
	lat, lon := 200.0, 0.0
	if _, err := svc.Update(context.Background(), id, "", "", "", "", &lat, &lon); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
