package organizers

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
)

// fakeRepo records calls and lets tests control GetByOwner/GetByID results.
type fakeRepo struct {
	owner        *Organizer
	submitAuto   bool
	submitCalled bool
	upsertName   string
	upsertCalled bool
}

func (f *fakeRepo) GetByOwner(context.Context, uuid.UUID) (*Organizer, error) {
	if f.owner == nil {
		return nil, ErrNotFound
	}
	return f.owner, nil
}
func (f *fakeRepo) GetByID(context.Context, uuid.UUID) (*Organizer, error) { return f.owner, nil }
func (f *fakeRepo) Upsert(_ context.Context, _ uuid.UUID, in Input) (*Organizer, error) {
	f.upsertCalled, f.upsertName = true, in.Name
	if f.owner == nil {
		f.owner = &Organizer{ID: uuid.Must(uuid.NewV4()), Name: in.Name, VerificationStatus: "draft"}
	}
	return f.owner, nil
}
func (f *fakeRepo) Submit(_ context.Context, _, _ uuid.UUID, auto bool) (string, error) {
	f.submitCalled, f.submitAuto = true, auto
	if auto {
		return "verified", nil
	}
	return "pending", nil
}
func (f *fakeRepo) Verify(context.Context, uuid.UUID, uuid.UUID) error              { return nil }
func (f *fakeRepo) Reject(context.Context, uuid.UUID, uuid.UUID, string) error      { return nil }
func (f *fakeRepo) Revoke(context.Context, uuid.UUID, uuid.UUID, string) error      { return nil }
func (f *fakeRepo) SetAutoVerify(context.Context, uuid.UUID, uuid.UUID, bool) error { return nil }
func (f *fakeRepo) SetDailyEventLimit(context.Context, uuid.UUID, uuid.UUID, *int) error {
	return nil
}
func (f *fakeRepo) List(context.Context, ListFilter) ([]Organizer, error)      { return nil, nil }
func (f *fakeRepo) History(context.Context, uuid.UUID) ([]HistoryEntry, error) { return nil, nil }
func (f *fakeRepo) Counts(context.Context) (Counts, error)                     { return Counts{}, nil }

type fakeSettings struct{ autoAll bool }

func (f fakeSettings) Bool(context.Context, string) (bool, error)             { return f.autoAll, nil }
func (f fakeSettings) SetBool(context.Context, string, uuid.UUID, bool) error { return nil }
func (f fakeSettings) All(context.Context) (map[string]bool, error)           { return nil, nil }

func TestUpsertRequiresName(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeSettings{})
	_, err := svc.Upsert(context.Background(), uuid.Must(uuid.NewV4()), Input{Name: "  "})
	if err != ErrNameRequired {
		t.Fatalf("err = %v; want ErrNameRequired", err)
	}
}

func TestRejectRequiresReason(t *testing.T) {
	svc := NewService(&fakeRepo{owner: &Organizer{}}, fakeSettings{})
	err := svc.Reject(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "   ")
	if err != ErrReasonRequired {
		t.Fatalf("err = %v; want ErrReasonRequired", err)
	}
}

func TestRevokeRequiresReason(t *testing.T) {
	svc := NewService(&fakeRepo{owner: &Organizer{}}, fakeSettings{})
	err := svc.Revoke(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "")
	if err != ErrReasonRequired {
		t.Fatalf("err = %v; want ErrReasonRequired", err)
	}
}

func TestSubmitAutoVerifiesWhenGlobalOn(t *testing.T) {
	r := &fakeRepo{owner: &Organizer{AutoVerify: false}}
	svc := NewService(r, fakeSettings{autoAll: true})
	status, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil || status != "verified" || !r.submitAuto {
		t.Fatalf("status=%q auto=%v err=%v; want verified/true/nil", status, r.submitAuto, err)
	}
}

func TestSubmitAutoVerifiesWhenOrgFlagOn(t *testing.T) {
	r := &fakeRepo{owner: &Organizer{AutoVerify: true}}
	svc := NewService(r, fakeSettings{autoAll: false})
	status, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil || status != "verified" || !r.submitAuto {
		t.Fatalf("status=%q auto=%v err=%v; want verified/true/nil", status, r.submitAuto, err)
	}
}

func TestSubmitQueuesWhenBothOff(t *testing.T) {
	r := &fakeRepo{owner: &Organizer{AutoVerify: false}}
	svc := NewService(r, fakeSettings{autoAll: false})
	status, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil || status != "pending" || r.submitAuto {
		t.Fatalf("status=%q auto=%v err=%v; want pending/false/nil", status, r.submitAuto, err)
	}
}

func TestSubmitErrorsWhenNoProfile(t *testing.T) {
	svc := NewService(&fakeRepo{owner: nil}, fakeSettings{})
	if _, err := svc.Submit(context.Background(), uuid.Must(uuid.NewV4())); err != ErrNotFound {
		t.Fatalf("err = %v; want ErrNotFound", err)
	}
}

func TestEnsureForOwnerCreatesAndAutoVerifies(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeSettings{})

	org, err := svc.EnsureForOwner(context.Background(), uuid.Must(uuid.NewV4()), " Студия Лиа ")
	if err != nil {
		t.Fatalf("EnsureForOwner: %v", err)
	}
	if org == nil {
		t.Fatal("expected a profile")
	}
	if !repo.upsertCalled || repo.upsertName != "Студия Лиа" {
		t.Errorf("upsert name = %q; want the trimmed user name", repo.upsertName)
	}
	if !repo.submitCalled || !repo.submitAuto {
		t.Errorf("submitCalled=%v auto=%v; want an auto-verifying submit", repo.submitCalled, repo.submitAuto)
	}
}

func TestEnsureForOwnerLeavesAnExistingProfileAlone(t *testing.T) {
	// Notably including a revoked one: a new event must not undo moderation.
	repo := &fakeRepo{owner: &Organizer{ID: uuid.Must(uuid.NewV4()), Name: "Уже есть", VerificationStatus: "rejected"}}
	svc := NewService(repo, fakeSettings{})

	org, err := svc.EnsureForOwner(context.Background(), uuid.Must(uuid.NewV4()), "Другое имя")
	if err != nil {
		t.Fatalf("EnsureForOwner: %v", err)
	}
	if repo.upsertCalled || repo.submitCalled {
		t.Errorf("existing profile was touched (upsert=%v submit=%v)", repo.upsertCalled, repo.submitCalled)
	}
	if org.Name != "Уже есть" || org.VerificationStatus != "rejected" {
		t.Errorf("profile changed: %+v", org)
	}
}

func TestEnsureForOwnerFallsBackWhenTheUserHasNoName(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeSettings{})

	if _, err := svc.EnsureForOwner(context.Background(), uuid.Must(uuid.NewV4()), "   "); err != nil {
		t.Fatalf("EnsureForOwner: %v", err)
	}
	if repo.upsertName != "Организатор" {
		t.Errorf("upsert name = %q; want the fallback", repo.upsertName)
	}
}
