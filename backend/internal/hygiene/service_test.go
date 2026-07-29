package hygiene

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
)

type fakeEvents struct {
	gotStatus string
	events    []*models.Event
	err       error
}

func (f *fakeEvents) List(_ context.Context, status string, _, _ *time.Time, _ *uuid.UUID) ([]*models.Event, error) {
	f.gotStatus = status
	return f.events, f.err
}

type fakeModerator struct {
	calls []struct {
		id     uuid.UUID
		reason string
	}
	errs map[uuid.UUID]error
}

func (f *fakeModerator) Takedown(_ context.Context, id, _ uuid.UUID, reason string) error {
	f.calls = append(f.calls, struct {
		id     uuid.UUID
		reason string
	}{id, reason})
	return f.errs[id]
}

func event(title, org string) *models.Event {
	return &models.Event{
		ID: uuid.Must(uuid.NewV4()), Title: title,
		Organizer: &models.Organizer{Name: org}, PriceType: "free",
	}
}

func TestList_OnlyPublishedAndDetects(t *testing.T) {
	ev := &fakeEvents{events: []*models.Event{event("QA Тур", "QA Block8"), event("Лекция", "Гараж")}}
	issues, err := NewService(ev, &fakeModerator{}).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ev.gotStatus != "published" {
		t.Fatalf("status = %q, want published", ev.gotStatus)
	}
	if len(issues) != 1 || issues[0].Kind != KindTestData {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestHideAll_TakesDownEachIssueWithItsReason(t *testing.T) {
	e := event("bla bla meet", "kornkorn10")
	ev := &fakeEvents{events: []*models.Event{e, event("Лекция", "Гараж")}}
	mod := &fakeModerator{}
	res, err := NewService(ev, mod).HideAll(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil {
		t.Fatalf("HideAll: %v", err)
	}
	if res.Hidden != 1 || res.Skipped != 0 {
		t.Fatalf("result = %+v, want hidden 1", res)
	}
	if len(mod.calls) != 1 || mod.calls[0].id != e.ID || mod.calls[0].reason != ReasonFor(KindTestData) {
		t.Fatalf("calls = %+v", mod.calls)
	}
}

func TestHideAll_SkipsAlreadyMovedEvents(t *testing.T) {
	e := event("QA Тур", "QA Block8")
	mod := &fakeModerator{errs: map[uuid.UUID]error{e.ID: moderation.ErrInvalidTransition}}
	res, err := NewService(&fakeEvents{events: []*models.Event{e}}, mod).
		HideAll(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil {
		t.Fatalf("HideAll: %v", err)
	}
	if res.Hidden != 0 || res.Skipped != 1 {
		t.Fatalf("result = %+v, want skipped 1", res)
	}
}

func TestHideAll_ReturnsRealErrors(t *testing.T) {
	e := event("QA Тур", "QA Block8")
	boom := errors.New("boom")
	mod := &fakeModerator{errs: map[uuid.UUID]error{e.ID: boom}}
	if _, err := NewService(&fakeEvents{events: []*models.Event{e}}, mod).
		HideAll(context.Background(), uuid.Must(uuid.NewV4())); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}
