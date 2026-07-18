package invitations_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	inv "github.com/Pashteto/lia/internal/invitations"
)

// --- fakes ---
type fakeRepo struct {
	inserted []inv.Invitation
	byToken  map[string]*inv.Invitation
	statuses map[uuid.UUID]string
	pending  []inv.Invitation
}

func (f *fakeRepo) Insert(_ context.Context, i inv.Invitation) error {
	f.inserted = append(f.inserted, i)
	return nil
}
func (f *fakeRepo) GetByToken(_ context.Context, t string) (*inv.Invitation, error) {
	if v, ok := f.byToken[t]; ok {
		return v, nil
	}
	return nil, inv.ErrNotFound
}
func (f *fakeRepo) GetByID(context.Context, uuid.UUID) (*inv.Invitation, error) {
	return nil, inv.ErrNotFound
}
func (f *fakeRepo) ListPendingByEmail(context.Context, string) ([]inv.Invitation, error) {
	return f.pending, nil
}
func (f *fakeRepo) SetStatus(_ context.Context, id uuid.UUID, s string) error {
	if f.statuses == nil {
		f.statuses = map[uuid.UUID]string{}
	}
	f.statuses[id] = s
	return nil
}
func (f *fakeRepo) ExpireOverdue(context.Context) error { return nil }

type fakeEvents struct{ owner uuid.UUID }

func (f fakeEvents) GetByID(context.Context, string) (string, uuid.UUID, error) {
	return "Йога", f.owner, nil
}

func (f fakeEvents) Details(context.Context, string) (inv.EventDetails, error) {
	return inv.EventDetails{Title: "Йога", OrganizerName: "Студия"}, nil
}

type fakeRSVP struct{ signedUp []uuid.UUID }

func (f *fakeRSVP) SignUp(_ context.Context, _, userID uuid.UUID, _ string) error {
	f.signedUp = append(f.signedUp, userID)
	return nil
}

type fakeMailer struct{ sent int }

func (f *fakeMailer) SendEventInvitation(context.Context, string, string, string) error {
	f.sent++
	return nil
}

func newSvc(repo inv.Repository, ev inv.EventPort, r inv.RSVPPort, m *fakeMailer) inv.Service {
	return inv.NewService(repo, ev, r, m)
}

func TestInvite_CreatesRowsAndSends(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	repo := &fakeRepo{}
	mail := &fakeMailer{}
	s := newSvc(repo, fakeEvents{owner: owner}, &fakeRSVP{}, mail)

	n, err := s.Invite(context.Background(), uuid.Must(uuid.NewV4()), owner, true,
		[]string{"a@x.com", "b@x.com"}, "https://presence.tarski.ru")
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if n != 2 || len(repo.inserted) != 2 || mail.sent != 2 {
		t.Fatalf("want 2 invites+2 emails, got n=%d rows=%d mail=%d", n, len(repo.inserted), mail.sent)
	}
	if repo.inserted[0].Token == "" || repo.inserted[0].ExpiresAt.Before(time.Now()) {
		t.Fatal("invite must have a token and a future expiry")
	}
}

func TestInvite_RejectsNonOwner(t *testing.T) {
	s := newSvc(&fakeRepo{}, fakeEvents{owner: uuid.Must(uuid.NewV4())}, &fakeRSVP{}, &fakeMailer{})
	_, err := s.Invite(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), true, []string{"a@x.com"}, "b")
	if err != inv.ErrNotOwner {
		t.Fatalf("want ErrNotOwner, got %v", err)
	}
}

func TestInvite_RejectsUnverifiedOrganizer(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	s := newSvc(&fakeRepo{}, fakeEvents{owner: owner}, &fakeRSVP{}, &fakeMailer{})
	_, err := s.Invite(context.Background(), uuid.Must(uuid.NewV4()), owner, false, []string{"a@x.com"}, "b")
	if err != inv.ErrNotVerified {
		t.Fatalf("want ErrNotVerified, got %v", err)
	}
}

func TestAcceptByToken_CreatesRSVP(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	eventID := uuid.Must(uuid.NewV4())
	repo := &fakeRepo{byToken: map[string]*inv.Invitation{
		"tok": {ID: id, EventID: eventID, InviteeEmail: "a@x.com", Status: "pending", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	rsvp := &fakeRSVP{}
	s := newSvc(repo, fakeEvents{}, rsvp, &fakeMailer{})

	userID := uuid.Must(uuid.NewV4())
	if err := s.AcceptByToken(context.Background(), "tok", "A@X.com", userID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if len(rsvp.signedUp) != 1 || rsvp.signedUp[0] != userID {
		t.Fatal("accept must sign the user up for the event")
	}
	if repo.statuses[id] != "accepted" {
		t.Fatalf("invite must be marked accepted, got %q", repo.statuses[id])
	}
}

func TestAcceptByToken_RejectsWrongEmail(t *testing.T) {
	repo := &fakeRepo{byToken: map[string]*inv.Invitation{
		"tok": {ID: uuid.Must(uuid.NewV4()), InviteeEmail: "a@x.com", Status: "pending", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	s := newSvc(repo, fakeEvents{}, &fakeRSVP{}, &fakeMailer{})
	err := s.AcceptByToken(context.Background(), "tok", "other@x.com", uuid.Must(uuid.NewV4()), true)
	if err != inv.ErrEmailMismatch {
		t.Fatalf("want ErrEmailMismatch, got %v", err)
	}
}

func TestAcceptByToken_RejectsUnverified(t *testing.T) {
	repo := &fakeRepo{byToken: map[string]*inv.Invitation{
		"tok": {ID: uuid.Must(uuid.NewV4()), InviteeEmail: "a@x.com", Status: "pending", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	s := newSvc(repo, fakeEvents{}, &fakeRSVP{}, &fakeMailer{})
	err := s.AcceptByToken(context.Background(), "tok", "a@x.com", uuid.Must(uuid.NewV4()), false)
	if err != inv.ErrNotVerified {
		t.Fatalf("want ErrNotVerified, got %v", err)
	}
}

func TestListMine_EnrichesWithEventAndInviter(t *testing.T) {
	eventID := uuid.Must(uuid.NewV4())
	repo := &fakeRepo{pending: []inv.Invitation{
		{ID: uuid.Must(uuid.NewV4()), EventID: eventID, InviteeEmail: "a@x.com", Status: "pending"},
	}}
	s := newSvc(repo, fakeEvents{}, &fakeRSVP{}, &fakeMailer{})

	items, err := s.ListMine(context.Background(), "a@x.com")
	if err != nil {
		t.Fatalf("list mine: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	// fakeEvents.Details returns title "Йога", organizer "Студия".
	if items[0].EventTitle != "Йога" || items[0].InviterName != "Студия" {
		t.Fatalf("row not enriched: title=%q inviter=%q", items[0].EventTitle, items[0].InviterName)
	}
	if items[0].EventID != eventID {
		t.Fatalf("event id lost: %v", items[0].EventID)
	}
}

var _ = strings.ToLower
