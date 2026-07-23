package invitations

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// Invitation is an organizer-issued invite for a person (by email) to attend
// a specific event. Rows live in the event_invitations table.
type Invitation struct {
	tableName struct{} `pg:"event_invitations"`

	ID            uuid.UUID `pg:"id"`
	EventID       uuid.UUID `pg:"event_id"`
	InviterUserID uuid.UUID `pg:"inviter_user_id"`
	InviteeEmail  string    `pg:"invitee_email"`
	Token         string    `pg:"token"`
	Status        string    `pg:"status"`
	CreatedAt     time.Time `pg:"created_at"`
	RespondedAt   time.Time `pg:"responded_at"`
	ExpiresAt     time.Time `pg:"expires_at"`
}

// Repository is the data-access layer over event_invitations.
type Repository interface {
	Insert(ctx context.Context, inv Invitation) error
	GetByToken(ctx context.Context, token string) (*Invitation, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Invitation, error)
	ListPendingByEmail(ctx context.Context, email string) ([]Invitation, error)
	SetStatus(ctx context.Context, id uuid.UUID, status string) error
	ExpireOverdue(ctx context.Context) error
}

// ErrNotFound is returned when an invitation lookup finds no matching row.
var ErrNotFound = errors.New("invitation not found")

// EventPort/RSVPPort/MailerPort are the collaborators the service needs
// (thin adapters over events.Service / rsvp.Service / notifications.Mailer
// are wired at the call site).
type EventPort interface {
	GetByID(ctx context.Context, id string) (title string, organizerUserID uuid.UUID, err error)
	// Details returns the display data the "my invitations" list needs.
	Details(ctx context.Context, id string) (EventDetails, error)
}

// EventDetails is the event display data shown on an invitation row. Only an
// event's owner can issue invites (see Invite), so the event's organizer is the
// inviter — OrganizerName therefore doubles as the inviter's name.
type EventDetails struct {
	Title         string
	StartsAt      time.Time
	OrganizerName string
}
type RSVPPort interface {
	SignUp(ctx context.Context, eventID, userID uuid.UUID, answer string) error
}
type MailerPort interface {
	SendEventInvitation(ctx context.Context, to, eventTitle, acceptURL string) error
}

// EmailVerifier lets the accept flow mark an invitee's address verified: the
// invitation was emailed to that address, so accepting it from the matching
// account proves ownership. Backed by the GateGuard signer at the call site.
type EmailVerifier interface {
	MarkEmailVerified(ctx context.Context, email string) error
}

var (
	ErrNotOwner      = errors.New("not event owner")
	ErrNotVerified   = errors.New("email not verified")
	ErrEmailMismatch = errors.New("invitation addressed to a different email")
	ErrNotPending    = errors.New("invitation is not pending")
)

const inviteTTL = 30 * 24 * time.Hour

type Preview struct {
	EventID    uuid.UUID
	EventTitle string
	Status     string
}

// MineItem is a pending invitation enriched with its event's display data, for
// the authenticated "my invitations" list so each row is identifiable.
type MineItem struct {
	Invitation
	EventTitle    string
	EventStartsAt time.Time
	InviterName   string
}

// Service is the invitations business logic: invite by email, preview an
// invite by token, accept/decline (by token or by id), and list mine.
type Service interface {
	Invite(ctx context.Context, eventID, inviterUserID uuid.UUID, inviterVerified bool, emails []string, baseURL string) (invited int, err error)
	Preview(ctx context.Context, token string) (*Preview, error)
	AcceptByToken(ctx context.Context, token, userEmail string, userID uuid.UUID, verified bool) error
	DeclineByToken(ctx context.Context, token, userEmail string) error
	ListMine(ctx context.Context, email string) ([]MineItem, error)
	AcceptByID(ctx context.Context, id uuid.UUID, userEmail string, userID uuid.UUID, verified bool) error
	DeclineByID(ctx context.Context, id uuid.UUID, userEmail string) error
}

type service struct {
	repo     Repository
	events   EventPort
	rsvp     RSVPPort
	mailer   MailerPort
	verifier EmailVerifier
}

// NewService builds the invitations service. verifier may be nil (e.g. when
// GateGuard is not configured); accept() then falls back to rejecting an
// unverified invitee rather than silently skipping verification.
func NewService(repo Repository, events EventPort, rsvp RSVPPort, mailer MailerPort, verifier EmailVerifier) Service {
	return &service{repo: repo, events: events, rsvp: rsvp, mailer: mailer, verifier: verifier}
}

func newToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *service) Invite(ctx context.Context, eventID, inviterUserID uuid.UUID, inviterVerified bool, emails []string, baseURL string) (int, error) {
	if !inviterVerified {
		return 0, ErrNotVerified
	}
	title, owner, err := s.events.GetByID(ctx, eventID.String())
	if err != nil {
		return 0, err
	}
	if owner != inviterUserID {
		return 0, ErrNotOwner
	}
	count := 0
	for _, raw := range emails {
		email := strings.ToLower(strings.TrimSpace(raw))
		if email == "" {
			continue
		}
		id, _ := uuid.NewV4()
		token := newToken()
		invRow := Invitation{
			ID: id, EventID: eventID, InviterUserID: inviterUserID,
			InviteeEmail: email, Token: token, Status: "pending",
			ExpiresAt: time.Now().Add(inviteTTL),
		}
		if err := s.repo.Insert(ctx, invRow); err != nil {
			return count, fmt.Errorf("insert invite %s: %w", email, err)
		}
		acceptURL := strings.TrimRight(baseURL, "/") + "/invite/" + token
		// Best-effort email; a failed send doesn't undo the invite (it shows in-app too).
		_ = s.mailer.SendEventInvitation(ctx, email, title, acceptURL)
		count++
	}
	return count, nil
}

func (s *service) Preview(ctx context.Context, token string) (*Preview, error) {
	invRow, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	title, _, err := s.events.GetByID(ctx, invRow.EventID.String())
	if err != nil {
		return nil, err
	}
	return &Preview{EventID: invRow.EventID, EventTitle: title, Status: invRow.Status}, nil
}

func (s *service) ListMine(ctx context.Context, email string) ([]MineItem, error) {
	rows, err := s.repo.ListPendingByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	out := make([]MineItem, 0, len(rows))
	for _, row := range rows {
		item := MineItem{Invitation: row}
		// Best-effort enrichment: a lookup failure for one event must not drop
		// the whole list, so we fall back to the bare invitation row.
		if d, dErr := s.events.Details(ctx, row.EventID.String()); dErr == nil {
			item.EventTitle = d.Title
			item.EventStartsAt = d.StartsAt
			item.InviterName = d.OrganizerName
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *service) AcceptByToken(ctx context.Context, token, userEmail string, userID uuid.UUID, verified bool) error {
	invRow, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return err
	}
	return s.accept(ctx, invRow, userEmail, userID, verified)
}

func (s *service) AcceptByID(ctx context.Context, id uuid.UUID, userEmail string, userID uuid.UUID, verified bool) error {
	invRow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.accept(ctx, invRow, userEmail, userID, verified)
}

func (s *service) accept(ctx context.Context, invRow *Invitation, userEmail string, userID uuid.UUID, verified bool) error {
	if invRow.Status != "pending" {
		return ErrNotPending
	}
	if !strings.EqualFold(strings.TrimSpace(userEmail), invRow.InviteeEmail) {
		return ErrEmailMismatch
	}
	// The invitation was emailed to invRow.InviteeEmail; accepting it from the
	// matching account proves ownership, so treat accept as email verification
	// (closes the "invited user still forced to verify" gap, QA 5a).
	if !verified {
		if s.verifier == nil {
			return ErrNotVerified
		}
		if err := s.verifier.MarkEmailVerified(ctx, invRow.InviteeEmail); err != nil {
			return fmt.Errorf("verify invitee on accept: %w", err)
		}
	}
	if err := s.rsvp.SignUp(ctx, invRow.EventID, userID, ""); err != nil {
		return fmt.Errorf("rsvp on accept: %w", err)
	}
	return s.repo.SetStatus(ctx, invRow.ID, "accepted")
}

func (s *service) DeclineByToken(ctx context.Context, token, userEmail string) error {
	invRow, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return err
	}
	return s.decline(ctx, invRow, userEmail)
}

func (s *service) DeclineByID(ctx context.Context, id uuid.UUID, userEmail string) error {
	invRow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.decline(ctx, invRow, userEmail)
}

func (s *service) decline(ctx context.Context, invRow *Invitation, userEmail string) error {
	if invRow.Status != "pending" {
		return ErrNotPending
	}
	if !strings.EqualFold(strings.TrimSpace(userEmail), invRow.InviteeEmail) {
		return ErrEmailMismatch
	}
	return s.repo.SetStatus(ctx, invRow.ID, "declined")
}
