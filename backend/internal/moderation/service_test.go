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
	counts         Counts
}

func (f *fakeRepo) Takedown(_ context.Context, _, _ uuid.UUID, reason string) error {
	f.takedownReason = reason
	return f.takedownErr
}
func (f *fakeRepo) Reinstate(_ context.Context, _, _ uuid.UUID) error { return f.reinstateErr }
func (f *fakeRepo) Counts(_ context.Context) (Counts, error)         { return f.counts, nil }
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
