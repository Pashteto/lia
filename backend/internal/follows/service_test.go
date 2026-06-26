package follows

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	organizers "github.com/Pashteto/lia/internal/organizers"
)

// --- fakes ---

type fakeRepo struct {
	added, removed [][2]uuid.UUID
	ownerIDs       []uuid.UUID
	following      bool
}

func (f *fakeRepo) Add(_ context.Context, u, o uuid.UUID) error {
	f.added = append(f.added, [2]uuid.UUID{u, o})
	return nil
}
func (f *fakeRepo) Remove(_ context.Context, u, o uuid.UUID) error {
	f.removed = append(f.removed, [2]uuid.UUID{u, o})
	return nil
}
func (f *fakeRepo) IsFollowing(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.following, nil
}
func (f *fakeRepo) ListFollowedOwnerIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return f.ownerIDs, nil
}
func (f *fakeRepo) ListFollowedOrganizers(context.Context, uuid.UUID) ([]FollowedOrg, error) {
	return nil, nil
}

// fakeOrgs implements organizers.Service but only GetByID is exercised; the rest
// panic if unexpectedly called.
type fakeOrgs struct {
	organizers.Service
	org *organizers.Organizer
	err error
}

func (f *fakeOrgs) GetByID(context.Context, uuid.UUID) (*organizers.Organizer, error) {
	return f.org, f.err
}

type fakeEvents struct {
	gotIDs   []uuid.UUID
	gotFrom  time.Time
	gotTo    time.Time
	returned []*models.Event
}

func (f *fakeEvents) ListForCalendar(_ context.Context, ids []uuid.UUID, from, to time.Time) ([]*models.Event, error) {
	f.gotIDs, f.gotFrom, f.gotTo = ids, from, to
	return f.returned, nil
}

// --- tests ---

func TestFollow_VerifiedOrgIsResolvedToOwner(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	repo := &fakeRepo{}
	orgs := &fakeOrgs{org: &organizers.Organizer{OwnerUserID: owner, VerificationStatus: "verified"}}
	svc := NewService(repo, orgs, &fakeEvents{})

	user := uuid.Must(uuid.NewV4())
	profile := uuid.Must(uuid.NewV4())
	if err := svc.Follow(context.Background(), user, profile); err != nil {
		t.Fatalf("Follow: %v", err)
	}
	if len(repo.added) != 1 || repo.added[0] != [2]uuid.UUID{user, owner} {
		t.Fatalf("expected Add(user, owner), got %v", repo.added)
	}
}

func TestFollow_NonVerifiedOrgIsNotFound(t *testing.T) {
	repo := &fakeRepo{}
	orgs := &fakeOrgs{org: &organizers.Organizer{VerificationStatus: "pending"}}
	svc := NewService(repo, orgs, &fakeEvents{})

	err := svc.Follow(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if len(repo.added) != 0 {
		t.Fatalf("must not persist a follow for a non-verified org")
	}
}

func TestFollow_UnknownOrgIsNotFound(t *testing.T) {
	repo := &fakeRepo{}
	orgs := &fakeOrgs{err: organizers.ErrNotFound}
	svc := NewService(repo, orgs, &fakeEvents{})

	if err := svc.Follow(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListEventsFromFollowed_NoFollowsShortCircuits(t *testing.T) {
	events := &fakeEvents{}
	svc := NewService(&fakeRepo{ownerIDs: nil}, &fakeOrgs{}, events)

	got, err := svc.ListEventsFromFollowed(context.Background(), uuid.Must(uuid.NewV4()), time.Now(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("ListEventsFromFollowed: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil events, got %v", got)
	}
	if events.gotIDs != nil {
		t.Fatalf("events service must not be queried when there are no follows")
	}
}

func TestListEventsFromFollowed_DelegatesWithOwnerIDsAndRange(t *testing.T) {
	owners := []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())}
	want := []*models.Event{{ID: uuid.Must(uuid.NewV4())}}
	events := &fakeEvents{returned: want}
	svc := NewService(&fakeRepo{ownerIDs: owners}, &fakeOrgs{}, events)

	from := time.Now()
	to := from.Add(24 * time.Hour)
	got, err := svc.ListEventsFromFollowed(context.Background(), uuid.Must(uuid.NewV4()), from, to)
	if err != nil {
		t.Fatalf("ListEventsFromFollowed: %v", err)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Fatalf("expected delegated events, got %v", got)
	}
	if len(events.gotIDs) != 2 || !events.gotFrom.Equal(from) || !events.gotTo.Equal(to) {
		t.Fatalf("expected delegation with owner ids + range, got ids=%v from=%v to=%v", events.gotIDs, events.gotFrom, events.gotTo)
	}
}
