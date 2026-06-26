package complaints

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/moderation"
)

type fakeRepo struct {
	inserted       Complaint
	insertCreated  bool
	insertErr      error
	eventExists    bool
	eventExistsErr error
	resolveStatus  string
	resolveCalled  bool
	resolveErr     error
}

func (f *fakeRepo) Insert(_ context.Context, c Complaint) (bool, error) {
	f.inserted = c
	return f.insertCreated, f.insertErr
}
func (f *fakeRepo) InboxGroups(context.Context) ([]EventReportGroup, error) { return nil, nil }
func (f *fakeRepo) TargetComplaints(context.Context, string, uuid.UUID) ([]Complaint, error) {
	return nil, nil
}
func (f *fakeRepo) ResolveOpenForTarget(_ context.Context, _ string, _, _ uuid.UUID, status, _ string) (int, error) {
	f.resolveCalled = true
	f.resolveStatus = status
	return 1, f.resolveErr
}
func (f *fakeRepo) OpenEventCount(context.Context) (int, error) { return 0, nil }
func (f *fakeRepo) EventExists(context.Context, uuid.UUID) (bool, error) {
	return f.eventExists, f.eventExistsErr
}

// fakeMod records the takedown reason and returns a configurable error.
type fakeMod struct {
	moderation.Service
	takedownErr    error
	takedownReason string
}

func (m *fakeMod) Takedown(_ context.Context, _, _ uuid.UUID, reason string) error {
	m.takedownReason = reason
	return m.takedownErr
}

func TestSubmit_InvalidCategory(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeMod{})
	_, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()), "event", uuid.Must(uuid.NewV4()), "bogus", "")
	if !errors.Is(err, ErrInvalidCategory) {
		t.Fatalf("err = %v, want ErrInvalidCategory", err)
	}
}

func TestSubmit_TargetNotFound(t *testing.T) {
	svc := NewService(&fakeRepo{eventExists: false}, &fakeMod{})
	_, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()), "event", uuid.Must(uuid.NewV4()), "spam", "")
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("err = %v, want ErrTargetNotFound", err)
	}
}

func TestSubmit_InsertsTrimmedNote(t *testing.T) {
	repo := &fakeRepo{eventExists: true, insertCreated: true}
	svc := NewService(repo, &fakeMod{})
	created, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()), "event", uuid.Must(uuid.NewV4()), "spam", "  hi  ")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !created {
		t.Fatalf("created = false, want true")
	}
	if repo.inserted.Note != "hi" || repo.inserted.Status != "open" || repo.inserted.Category != "spam" {
		t.Fatalf("inserted = %+v", repo.inserted)
	}
}

func TestResolve_TakedownRequiresResolution(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeMod{})
	err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "takedown", "   ")
	if !errors.Is(err, ErrResolutionRequired) {
		t.Fatalf("err = %v, want ErrResolutionRequired", err)
	}
}

func TestResolve_TakedownComposesModeration(t *testing.T) {
	repo := &fakeRepo{}
	mod := &fakeMod{}
	svc := NewService(repo, mod)
	if err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "takedown", "scam"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if mod.takedownReason != "scam" {
		t.Fatalf("takedown reason = %q, want scam", mod.takedownReason)
	}
	if !repo.resolveCalled || repo.resolveStatus != "resolved" {
		t.Fatalf("resolve status = %q (called=%v), want resolved", repo.resolveStatus, repo.resolveCalled)
	}
}

func TestResolve_TakedownInvalidTransitionDoesNotCloseComplaints(t *testing.T) {
	repo := &fakeRepo{}
	mod := &fakeMod{takedownErr: moderation.ErrInvalidTransition}
	svc := NewService(repo, mod)
	err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "takedown", "scam")
	if !errors.Is(err, moderation.ErrInvalidTransition) {
		t.Fatalf("err = %v, want ErrInvalidTransition", err)
	}
	if repo.resolveCalled {
		t.Fatalf("ResolveOpenForTarget should not be called when takedown fails")
	}
}

func TestResolve_Dismiss(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeMod{})
	if err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "dismiss", ""); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !repo.resolveCalled || repo.resolveStatus != "dismissed" {
		t.Fatalf("resolve status = %q (called=%v), want dismissed", repo.resolveStatus, repo.resolveCalled)
	}
}

func TestResolve_InvalidAction(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeMod{})
	err := svc.Resolve(context.Background(), "event", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "nuke", "x")
	if !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("err = %v, want ErrInvalidAction", err)
	}
}
