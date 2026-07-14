package feedback_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pashteto/lia/internal/feedback"
	"github.com/gofrs/uuid"
)

type fakeRepo struct {
	owner    uuid.UUID
	endsAt   time.Time
	exists   bool
	active   bool
	already  bool
	inserted *feedback.Feedback
	items    []feedback.Item
}

func (f *fakeRepo) EventGate(_ context.Context, _ uuid.UUID) (uuid.UUID, time.Time, bool, error) {
	return f.owner, f.endsAt, f.exists, nil
}
func (f *fakeRepo) HasActiveRsvp(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return f.active, nil
}
func (f *fakeRepo) ExistsForUser(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return f.already, nil
}
func (f *fakeRepo) Insert(_ context.Context, fb feedback.Feedback) error {
	f.inserted = &fb
	return nil
}
func (f *fakeRepo) ListForEvent(_ context.Context, _ uuid.UUID) ([]feedback.Item, error) {
	return f.items, nil
}

func base() *fakeRepo {
	return &fakeRepo{owner: uuid.Must(uuid.NewV4()), endsAt: time.Now().Add(-time.Hour), exists: true, active: true}
}

func TestSubmit_HappyPath(t *testing.T) {
	r := base()
	svc := feedback.NewService(r)
	if err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 4, "ок"); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if r.inserted == nil || r.inserted.Rating != 4 {
		t.Fatal("not inserted")
	}
}

func TestSubmit_NotEnded(t *testing.T) {
	r := base()
	r.endsAt = time.Now().Add(time.Hour)
	err := feedback.NewService(r).Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 4, "")
	if !errors.Is(err, feedback.ErrNotEnded) {
		t.Fatalf("want ErrNotEnded, got %v", err)
	}
}

func TestSubmit_NotParticipant(t *testing.T) {
	r := base()
	r.active = false
	err := feedback.NewService(r).Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 4, "")
	if !errors.Is(err, feedback.ErrNotParticipant) {
		t.Fatalf("want ErrNotParticipant, got %v", err)
	}
}

func TestSubmit_Duplicate(t *testing.T) {
	r := base()
	r.already = true
	err := feedback.NewService(r).Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 4, "")
	if !errors.Is(err, feedback.ErrAlreadySubmitted) {
		t.Fatalf("want ErrAlreadySubmitted, got %v", err)
	}
}

func TestSubmit_BadRating(t *testing.T) {
	err := feedback.NewService(base()).Submit(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), 6, "")
	if !errors.Is(err, feedback.ErrInvalidRating) {
		t.Fatalf("want ErrInvalidRating, got %v", err)
	}
}

func TestForOwner_ForbiddenForNonOwner(t *testing.T) {
	r := base()
	_, err := feedback.NewService(r).ForOwner(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), false)
	if !errors.Is(err, feedback.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestForOwner_AverageAndCount(t *testing.T) {
	r := base()
	r.items = []feedback.Item{{Rating: 5}, {Rating: 3}}
	sum, err := feedback.NewService(r).ForOwner(context.Background(), uuid.Must(uuid.NewV4()), r.owner, false)
	if err != nil {
		t.Fatalf("ForOwner: %v", err)
	}
	if sum.Count != 2 || sum.Average != 4.0 {
		t.Fatalf("want avg 4 count 2, got %+v", sum)
	}
}
