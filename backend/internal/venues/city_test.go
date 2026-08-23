package venues

import (
	"context"
	"errors"
	"testing"

	"github.com/Pashteto/lia/internal/models"
)

// cityCapturingRepo records the city Search was scoped to.
type cityCapturingRepo struct {
	mockRepo
	searchCity string
}

func (m *cityCapturingRepo) Search(city, _ string, _ int) ([]*models.Venue, error) {
	m.searchCity = city
	return m.searchResult, nil
}

func TestSearch_PassesCityToRepo(t *testing.T) {
	repo := &cityCapturingRepo{}
	svc := NewService(repo)
	if _, err := svc.Search(context.Background(), models.CitySPB, "эрарта", 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.searchCity != models.CitySPB {
		t.Fatalf("expected search scoped to %q, got %q", models.CitySPB, repo.searchCity)
	}
}

func TestSearch_EmptyCityDefaultsToMSK(t *testing.T) {
	repo := &cityCapturingRepo{}
	svc := NewService(repo)
	if _, err := svc.Search(context.Background(), "", "гэс", 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.searchCity != models.CityMSK {
		t.Fatalf("expected default city %q, got %q", models.CityMSK, repo.searchCity)
	}
}

func TestSearch_InvalidCityRejected(t *testing.T) {
	svc := NewService(&cityCapturingRepo{})
	_, err := svc.Search(context.Background(), "ekb", "x", 10)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_DefaultsCityToMSK(t *testing.T) {
	repo := &cityCapturingRepo{}
	svc := NewService(repo)
	created, err := svc.Create(context.Background(), &models.Venue{Name: "ГЭС-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.City != models.CityMSK {
		t.Fatalf("expected default city %q, got %q", models.CityMSK, created.City)
	}
}

func TestCreate_KeepsExplicitCity(t *testing.T) {
	repo := &cityCapturingRepo{}
	svc := NewService(repo)
	created, err := svc.Create(context.Background(), &models.Venue{Name: "Севкабель Порт", City: models.CitySPB})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.City != models.CitySPB {
		t.Fatalf("expected city %q, got %q", models.CitySPB, created.City)
	}
}

func TestCreate_InvalidCityRejected(t *testing.T) {
	svc := NewService(&cityCapturingRepo{})
	_, err := svc.Create(context.Background(), &models.Venue{Name: "X", City: "СПБ"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
