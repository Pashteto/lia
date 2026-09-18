package moderation

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
)

type fakeRepo struct {
	takedownReason string
	takedownErr    error
	reinstateErr   error
	approveErr     error
	approveCalled  bool
	reviewCalled   bool
	reviewErr      error
	counts         Counts
}

func (f *fakeRepo) Takedown(_ context.Context, _, _ uuid.UUID, reason string) error {
	f.takedownReason = reason
	return f.takedownErr
}
func (f *fakeRepo) Reinstate(_ context.Context, _, _ uuid.UUID) error { return f.reinstateErr }
func (f *fakeRepo) Approve(_ context.Context, _, _ uuid.UUID) error {
	f.approveCalled = true
	return f.approveErr
}
func (f *fakeRepo) Review(_ context.Context, _, _ uuid.UUID) error {
	f.reviewCalled = true
	return f.reviewErr
}
func (f *fakeRepo) Counts(_ context.Context) (Counts, error)                    { return f.counts, nil }
func (f *fakeRepo) LatestReason(_ context.Context, _ uuid.UUID) (string, error) { return "", nil }

func TestTakedown_RequiresReason(t *testing.T) {
	svc := NewService(&fakeRepo{})
	err := svc.Takedown(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "   ")
	if !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("err = %v, want ErrReasonRequired", err)
	}
}

func TestTakedown_PassesReasonToRepo(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	if err := svc.Takedown(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "spam"); err != nil {
		t.Fatalf("takedown: %v", err)
	}
	if repo.takedownReason != "spam" {
		t.Fatalf("reason = %q, want spam", repo.takedownReason)
	}
}

func TestApprove_DelegatesToRepo(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	if err := svc.Approve(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if !repo.approveCalled {
		t.Fatalf("expected repo.Approve to be called")
	}
}

func TestApprove_PropagatesInvalidTransition(t *testing.T) {
	repo := &fakeRepo{approveErr: ErrInvalidTransition}
	svc := NewService(repo)
	err := svc.Approve(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("err = %v, want ErrInvalidTransition", err)
	}
}

// «Одобрить» on an already-published event used to be a client-side no-op, so
// the post-moderation queue never drained. It now goes through the repository.
func TestReview_DelegatesToRepo(t *testing.T) {
	repo := &fakeRepo{}
	if err := NewService(repo).Review(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())); err != nil {
		t.Fatalf("Review: %v", err)
	}
	if !repo.reviewCalled {
		t.Fatal("expected repo.Review to be called")
	}
}

func TestReview_PropagatesInvalidTransition(t *testing.T) {
	repo := &fakeRepo{reviewErr: ErrInvalidTransition}
	err := NewService(repo).Review(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
	if err != ErrInvalidTransition {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}
