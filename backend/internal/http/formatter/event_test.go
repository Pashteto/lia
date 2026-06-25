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

func TestEventToAPI_CoverURL(t *testing.T) {
	ev := &domainModels.Event{
		Title:    "x",
		Status:   domainModels.EventPublished,
		StartsAt: time.Now(),
		CoverURL: "https://example.com/cover.jpg",
	}
	out := EventToAPI(ev)
	if out.CoverURL == nil || *out.CoverURL != "https://example.com/cover.jpg" {
		t.Fatalf("expected cover_url to be set, got %v", out.CoverURL)
	}
}

func TestEventFromAPIInput_CoverFileID(t *testing.T) {
	id, _ := uuid.NewV4()
	title := "x"
	starts := strfmt.DateTime(time.Now())
	coverID := strfmt.UUID(id.String())
	in := &apiModels.EventInput{
		Title:       &title,
		StartsAt:    &starts,
		CoverFileID: &coverID,
	}
	ev, err := EventFromAPIInput(in)
	if err != nil {
		t.Fatalf("EventFromAPIInput error: %v", err)
	}
	if ev.CoverFileID != id {
		t.Fatalf("expected CoverFileID %s, got %s", id, ev.CoverFileID)
	}
}

func TestEventFromAPIInput_DefaultsToDraft(t *testing.T) {
	title := "X"
	starts := strfmt.DateTime(time.Now())
	in := &apiModels.EventInput{Title: &title, StartsAt: &starts}

	ev, err := EventFromAPIInput(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Status != domainModels.EventDraft {
		t.Fatalf("expected default status draft, got %v", ev.Status)
	}
}

func TestEventPatchToUpdateParams_MapsProvidedFields(t *testing.T) {
	title := "New"
	in := &apiModels.EventPatch{Title: title, Status: "published"}

	p := EventPatchToUpdateParams(in)
	if p.Title == nil || *p.Title != "New" {
		t.Fatalf("title not mapped: %+v", p.Title)
	}
	if p.Status == nil || *p.Status != "published" {
		t.Fatalf("status not mapped: %+v", p.Status)
	}
	if p.Description != nil {
		t.Fatalf("omitted field should be nil, got %+v", p.Description)
	}
}

func TestEventToAPIIncludesSignupFields(t *testing.T) {
	cap := 10
	remaining := 4
	e := &domainModels.Event{
		ID:              uuid.Must(uuid.NewV4()),
		Title:           "x",
		StartsAt:        time.Now(),
		Status:          domainModels.EventPublished,
		SignupMode:      "application",
		Capacity:        &cap,
		CuratorQuestion: "почему?",
		SeatsRemaining:  &remaining,
		MyRsvpStatus:   "applied",
	}
	out := EventToAPI(e)
	if out.SignupMode != "application" {
		t.Fatalf("signup_mode = %q", out.SignupMode)
	}
	if out.Capacity == nil || *out.Capacity != 10 {
		t.Fatalf("capacity = %v", out.Capacity)
	}
	if out.SeatsRemaining == nil || *out.SeatsRemaining != 4 {
		t.Fatalf("seats_remaining = %v", out.SeatsRemaining)
	}
	if out.MyRsvpStatus != "applied" {
		t.Fatalf("my_rsvp_status = %q", out.MyRsvpStatus)
	}
}
