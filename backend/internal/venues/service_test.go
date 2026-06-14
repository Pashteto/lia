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

type mockRepo struct {
	searchResult []*models.Venue
	getResult    *models.Venue
	getErr       error
	created      *models.Venue
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
