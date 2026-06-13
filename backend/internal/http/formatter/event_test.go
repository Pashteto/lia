package formatter

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gofrs/uuid"

	apiModels "github.com/Pashteto/lia/internal/http/models"
	domainModels "github.com/Pashteto/lia/internal/models"
)

func TestCategoryToAPI(t *testing.T) {
	id, _ := uuid.NewV4()
	got := CategoryToAPI(&domainModels.Category{ID: id, Slug: "lecture", Label: "Лекции"})
	if got == nil || got.ID == nil || got.Slug == nil || got.Label == nil {
		t.Fatal("expected non-nil category and fields")
	}
	if got.ID.String() != id.String() || *got.Slug != "lecture" || *got.Label != "Лекции" {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}

func TestCategoryToAPI_Nil(t *testing.T) {
	if CategoryToAPI(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestEventToAPI_Categories(t *testing.T) {
	id, _ := uuid.NewV4()
	ev := &domainModels.Event{
		Title:      "x",
		Status:     domainModels.EventPublished,
		StartsAt:   time.Now(),
		Categories: []*domainModels.Category{{ID: id, Slug: "lecture", Label: "Лекции"}},
	}
	out := EventToAPI(ev)
	if len(out.Categories) != 1 || *out.Categories[0].Slug != "lecture" {
		t.Fatalf("expected 1 mapped category, got %+v", out.Categories)
	}
}

func TestEventToAPI_EmptyCategories(t *testing.T) {
	ev := &domainModels.Event{Title: "x", Status: domainModels.EventPublished, StartsAt: time.Now()}
	out := EventToAPI(ev)
	if out.Categories == nil {
		t.Fatal("expected non-nil empty slice so JSON serializes as []")
	}
	if len(out.Categories) != 0 {
		t.Fatalf("expected 0 categories, got %d", len(out.Categories))
	}
}

func TestEventFromAPIInput_CategoryIDs(t *testing.T) {
	id, _ := uuid.NewV4()
	title := "x"
	starts := strfmt.DateTime(time.Now())
	in := &apiModels.EventInput{
		Title:       &title,
		StartsAt:    &starts,
		CategoryIds: []strfmt.UUID{strfmt.UUID(id.String())},
	}
	ev, err := EventFromAPIInput(in)
	if err != nil {
		t.Fatalf("EventFromAPIInput error: %v", err)
	}
	if len(ev.CategoryIDs) != 1 || ev.CategoryIDs[0] != id {
		t.Fatalf("expected parsed category id, got %v", ev.CategoryIDs)
	}
}
