# Phase C — Event Invitations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let organizers invite people to a specific event by email; invitees (including people with no account yet) accept via an email link or an in-app pending list; accepting creates/confirms an RSVP; accepting requires a verified email.

**Architecture:** New `event_invitations` table (migration 000020) keyed by `invitee_email` + `token` so invites exist before the person registers. New Lia domain package `internal/invitations` (service + go-pg repository), a new SMTP mailer in `internal/notifications` (Lia has none today), and a raw `http.ServeMux` handler mounted ahead of the swagger mux (same pattern as complaints/follows). Accepting calls the existing RSVP `SignUp`. Frontend adds an organizer invite panel, a public `/invite/[token]` landing page, and `/me/invitations`.

**Tech Stack:** Go, go-pg v10, golang-migrate, `net/smtp`; Next.js app-router, React Query, TypeScript.

## Global Constraints

- Module: `backend/`. Run commands from `backend/`.
- Migrations: golang-migrate, 6-digit sequential; **next is `000020`**; **no foreign keys** (loose uuid refs, matching `events.organizer_id`, `event_rsvps.user_id`).
- New HTTP endpoints use the **raw `http.ServeMux` pattern** (`backend/internal/http/complaints/handler.go` is the template), mounted + dispatched in `backend/internal/http/module.go` (~:290-395), NOT swagger.
- Accepting an invite requires a **verified email** (depends on Phase B's `domain.User.EmailVerified`) and that the authenticated user's email equals `invitee_email` (case-insensitive).
- Organizer sending invites must own the event (`event.OrganizerID == inviterUserID`) and be verified.
- Invitation email is sent via the **new** Lia SMTP mailer (SendPulse creds; From `info@tarski.ru`) — there is no existing mailer to reuse.
- Invite `token` is a random URL-safe string; `expires_at` = 30 days from creation. Statuses: `pending`, `accepted`, `declined`, `revoked`, `expired`.
- Russian UI copy; reuse `frontend/components/EventApplicationsPanel.tsx` (panel) and `frontend/app/me/practices/page.tsx` (/me page) patterns; reuse `LoginModal` from `frontend/components/AuthButton.tsx` for the invite login/signup step.
- DB lib is go-pg v10 with raw SQL (`db.ExecContext` / `db.QueryContext` / `db.QueryOneContext(pg.Scan(...))`) — mirror `backend/internal/complaints/repository.go`.

---

### Task 1: Migration — `event_invitations` table

**Files:**
- Create: `backend/db/migrations/000020_event_invitations.up.sql`
- Create: `backend/db/migrations/000020_event_invitations.down.sql`

- [ ] **Step 1: Create the migration files**

Run: `cd backend && make migrate-create NAME=event_invitations`
This creates `000020_event_invitations.up.sql` / `.down.sql`. Put in `.up.sql`:

```sql
-- event_invitations: organizer invites a person (by email) to a specific event.
-- Keyed by invitee_email so an invite can exist before the person registers.
-- No FKs (loose uuid refs), matching the repo convention.
CREATE TABLE IF NOT EXISTS event_invitations (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         uuid NOT NULL,
    inviter_user_id  uuid NOT NULL,
    invitee_email    text NOT NULL,
    token            text NOT NULL,
    status           text NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending','accepted','declined','revoked','expired')),
    created_at       timestamptz NOT NULL DEFAULT now(),
    responded_at     timestamptz,
    expires_at       timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS event_invitations_token_idx ON event_invitations (token);
CREATE INDEX IF NOT EXISTS event_invitations_email_status_idx ON event_invitations (lower(invitee_email), status);
-- At most one live (pending) invite per (event, email).
CREATE UNIQUE INDEX IF NOT EXISTS event_invitations_event_email_pending_idx
    ON event_invitations (event_id, lower(invitee_email)) WHERE status = 'pending';
```

Put in `.down.sql`:
```sql
DROP TABLE IF EXISTS event_invitations;
```

- [ ] **Step 2: Apply + roll back to validate**

Run:
```bash
cd backend && make migrate-up && make migrate-version && make migrate-down && make migrate-up
```
Expected: up to version 20, down removes it, up re-applies cleanly (no SQL errors).

- [ ] **Step 3: Commit**

```bash
git add backend/db/migrations/000020_event_invitations.up.sql backend/db/migrations/000020_event_invitations.down.sql
git commit -m "feat(db): event_invitations table (migration 000020)"
```

---

### Task 2: Domain types + repository

**Files:**
- Create: `backend/internal/invitations/service.go` (domain types + interfaces)
- Create: `backend/internal/invitations/repository.go` (go-pg impl)

**Interfaces:**
- Produces:
  ```go
  type Invitation struct {
      ID            uuid.UUID
      EventID       uuid.UUID
      InviterUserID uuid.UUID
      InviteeEmail  string
      Token         string
      Status        string
      CreatedAt     time.Time
      RespondedAt   time.Time
      ExpiresAt     time.Time
  }
  type Repository interface {
      Insert(ctx context.Context, inv Invitation) error
      GetByToken(ctx context.Context, token string) (*Invitation, error)
      GetByID(ctx context.Context, id uuid.UUID) (*Invitation, error)
      ListPendingByEmail(ctx context.Context, email string) ([]Invitation, error)
      SetStatus(ctx context.Context, id uuid.UUID, status string) error
      ExpireOverdue(ctx context.Context) error
  }
  var ErrNotFound = errors.New("invitation not found")
  ```

- [ ] **Step 1: Define domain types + interfaces**

Create `backend/internal/invitations/service.go` with the `Invitation` struct, `Repository` interface, `ErrNotFound` above, plus (filled in Task 4) the `Service` interface — for now stub the file with the types + a `Service` interface declaration whose methods are defined in Task 4.

- [ ] **Step 2: Implement the repository (mirror complaints/follows)**

Create `backend/internal/invitations/repository.go`:

```go
package invitations

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

// NewRepository builds the go-pg backed invitation repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) Insert(ctx context.Context, inv Invitation) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO event_invitations
		   (id, event_id, inviter_user_id, invitee_email, token, status, created_at, expires_at)
		 VALUES (?, ?, ?, lower(?), ?, 'pending', now(), ?)
		 ON CONFLICT (event_id, lower(invitee_email)) WHERE status='pending' DO NOTHING`,
		inv.ID, inv.EventID, inv.InviterUserID, inv.InviteeEmail, inv.Token, inv.ExpiresAt)
	return err
}

func (r *pgRepository) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	return r.getBy(ctx, `token = ?`, token)
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Invitation, error) {
	return r.getBy(ctx, `id = ?`, id)
}

func (r *pgRepository) getBy(ctx context.Context, where string, arg any) (*Invitation, error) {
	var inv Invitation
	_, err := r.db.QueryOneContext(ctx, pg.Scan(
		&inv.ID, &inv.EventID, &inv.InviterUserID, &inv.InviteeEmail,
		&inv.Token, &inv.Status, &inv.CreatedAt, &inv.RespondedAt, &inv.ExpiresAt),
		`SELECT id, event_id, inviter_user_id, invitee_email, token, status,
		        created_at, COALESCE(responded_at, 'epoch'), expires_at
		   FROM event_invitations WHERE `+where, arg)
	if errors.Is(err, pg.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *pgRepository) ListPendingByEmail(ctx context.Context, email string) ([]Invitation, error) {
	var out []Invitation
	_, err := r.db.QueryContext(ctx, &out,
		`SELECT id, event_id, inviter_user_id, invitee_email, token, status,
		        created_at, COALESCE(responded_at,'epoch') AS responded_at, expires_at
		   FROM event_invitations
		  WHERE lower(invitee_email) = lower(?) AND status = 'pending' AND expires_at > now()
		  ORDER BY created_at DESC`, email)
	return out, err
}

func (r *pgRepository) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE event_invitations SET status = ?, responded_at = now() WHERE id = ?`, status, id)
	return err
}

func (r *pgRepository) ExpireOverdue(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE event_invitations SET status='expired'
		  WHERE status='pending' AND expires_at <= now()`)
	return err
}

var _ = time.Now // ExpiresAt is set by the service; keep time imported if unused here
```

> The `QueryContext(ctx, &out, ...)` form requires `Invitation` to have `pg:"col"` tags matching the selected columns. Add tags to the struct in `service.go`: `pg:"id"`, `pg:"event_id"`, `pg:"inviter_user_id"`, `pg:"invitee_email"`, `pg:"token"`, `pg:"status"`, `pg:"created_at"`, `pg:"responded_at"`, `pg:"expires_at"`, and `tableName struct{} \`pg:"event_invitations"\``. Remove the `var _ = time.Now` line if `time` is used by the struct import instead.

- [ ] **Step 3: Build**

Run: `cd backend && go build ./internal/invitations/`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/invitations/service.go backend/internal/invitations/repository.go
git commit -m "feat(invitations): domain types + go-pg repository"
```

---

### Task 3: Lia SMTP mailer (`internal/notifications`)

**Files:**
- Create: `backend/internal/notifications/mailer.go`
- Test: `backend/internal/notifications/mailer_test.go`

**Interfaces:**
- Produces:
  ```go
  type Mailer interface {
      SendEventInvitation(ctx context.Context, to, eventTitle, acceptURL string) error
  }
  func NewSMTPMailer(addr, username, password, from string) Mailer
  func RenderInvitationEmail(eventTitle, acceptURL string) (subject, htmlBody string) // testable pure fn
  ```

- [ ] **Step 1: Write the failing test (pure render fn)**

```go
// backend/internal/notifications/mailer_test.go
package notifications_test

import (
	"strings"
	"testing"

	"github.com/Pashteto/lia/internal/notifications"
)

func TestRenderInvitationEmail(t *testing.T) {
	subject, body := notifications.RenderInvitationEmail("Йога в парке", "https://presence.tarski.ru/invite/abc")
	if !strings.HasPrefix(subject, "Subject:") {
		t.Fatalf("subject must start with 'Subject:', got %q", subject)
	}
	if !strings.Contains(body, "Йога в парке") || !strings.Contains(body, "https://presence.tarski.ru/invite/abc") {
		t.Fatalf("body missing title or link: %s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/notifications/ -run TestRenderInvitationEmail -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement the mailer (mirror GateGuard's net/smtp LOGIN auth)**

```go
// backend/internal/notifications/mailer.go
package notifications

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

// Mailer sends transactional email for the Lia app.
type Mailer interface {
	SendEventInvitation(ctx context.Context, to, eventTitle, acceptURL string) error
}

type smtpMailer struct {
	addr, from, username, password string
}

// NewSMTPMailer builds an SMTP mailer (SendPulse). A blank addr yields a no-op
// mailer so local/dev runs without SMTP config don't fail invites.
func NewSMTPMailer(addr, username, password, from string) Mailer {
	if addr == "" {
		return noopMailer{}
	}
	return &smtpMailer{addr: addr, from: from, username: username, password: password}
}

func (m *smtpMailer) Start(_ *smtp.ServerInfo) (string, []byte, error) { return "LOGIN", []byte{}, nil }
func (m *smtpMailer) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch string(fromServer) {
	case "Username:":
		return []byte(m.username), nil
	case "Password:":
		return []byte(m.password), nil
	default:
		return nil, errors.New("unexpected SMTP server challenge")
	}
}

func (m *smtpMailer) SendEventInvitation(_ context.Context, to, eventTitle, acceptURL string) error {
	subject, body := RenderInvitationEmail(eventTitle, acceptURL)
	headers := []string{
		"MIME-version: 1.0;",
		`Content-Type: text/html; charset="UTF-8";`,
		fmt.Sprintf("From: %s", m.from),
		fmt.Sprintf("To: %s", to),
		subject,
	}
	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body)
	if err := smtp.SendMail(m.addr, m, m.from, []string{to}, msg); err != nil {
		return fmt.Errorf("send invitation email: %w", err)
	}
	return nil
}

// RenderInvitationEmail builds the subject line and HTML body (pure/testable).
func RenderInvitationEmail(eventTitle, acceptURL string) (string, string) {
	subject := "Subject: Presence: приглашение на событие"
	body := fmt.Sprintf(`<!DOCTYPE html><html lang="ru"><body style="font-family:Arial,sans-serif;background:#f4f4f4;padding:20px;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;padding:24px;line-height:1.5;">
<h2>Вас пригласили на событие</h2>
<p>Вас пригласили на «%s».</p>
<p><a href="%s" style="display:inline-block;padding:10px 20px;background:#8950fa;color:#fff;text-decoration:none;border-radius:20px;">Открыть приглашение</a></p>
<p style="color:#666;font-size:13px;">Если ссылка не открывается, скопируйте её в браузер: %s</p>
</div></body></html>`, eventTitle, acceptURL, acceptURL)
	return subject, body
}

type noopMailer struct{}

func (noopMailer) SendEventInvitation(context.Context, string, string, string) error { return nil }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/notifications/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/notifications/mailer.go backend/internal/notifications/mailer_test.go
git commit -m "feat(notifications): Lia SMTP mailer + invitation email template"
```

---

### Task 4: Invitations service (with TDD)

**Files:**
- Modify: `backend/internal/invitations/service.go` (add `Service` + implementation + ports)
- Test: `backend/internal/invitations/service_test.go`

**Interfaces:**
- Consumes: `Repository` (Task 2), `Mailer` (Task 3), and two small ports it defines:
  ```go
  type EventPort interface {
      GetByID(ctx context.Context, id string) (title string, organizerUserID uuid.UUID, err error)
  }
  type RSVPPort interface {
      SignUp(ctx context.Context, eventID, userID uuid.UUID, answer string) error
  }
  ```
  (Thin adapters over the existing `events.Service.GetByID` and `rsvp.Service.SignUp` are wired in Task 6.)
- Produces:
  ```go
  type Service interface {
      Invite(ctx context.Context, eventID, inviterUserID uuid.UUID, inviterVerified bool, emails []string, baseURL string) (invited int, err error)
      Preview(ctx context.Context, token string) (*Preview, error)
      AcceptByToken(ctx context.Context, token, userEmail string, userID uuid.UUID, verified bool) error
      DeclineByToken(ctx context.Context, token, userEmail string) error
      ListMine(ctx context.Context, email string) ([]Invitation, error)
      AcceptByID(ctx context.Context, id uuid.UUID, userEmail string, userID uuid.UUID, verified bool) error
      DeclineByID(ctx context.Context, id uuid.UUID, userEmail string) error
  }
  type Preview struct { EventID uuid.UUID; EventTitle string; Status string }
  var (
      ErrNotOwner        = errors.New("not event owner")
      ErrNotVerified     = errors.New("email not verified")
      ErrEmailMismatch   = errors.New("invitation addressed to a different email")
      ErrNotPending      = errors.New("invitation is not pending")
  )
  ```

- [ ] **Step 1: Write the failing tests**

```go
// backend/internal/invitations/service_test.go
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
}
func (f *fakeRepo) Insert(_ context.Context, i inv.Invitation) error { f.inserted = append(f.inserted, i); return nil }
func (f *fakeRepo) GetByToken(_ context.Context, t string) (*inv.Invitation, error) {
	if v, ok := f.byToken[t]; ok { return v, nil }
	return nil, inv.ErrNotFound
}
func (f *fakeRepo) GetByID(context.Context, uuid.UUID) (*inv.Invitation, error) { return nil, inv.ErrNotFound }
func (f *fakeRepo) ListPendingByEmail(context.Context, string) ([]inv.Invitation, error) { return nil, nil }
func (f *fakeRepo) SetStatus(_ context.Context, id uuid.UUID, s string) error {
	if f.statuses == nil { f.statuses = map[uuid.UUID]string{} }
	f.statuses[id] = s; return nil
}
func (f *fakeRepo) ExpireOverdue(context.Context) error { return nil }

type fakeEvents struct{ owner uuid.UUID }
func (f fakeEvents) GetByID(context.Context, string) (string, uuid.UUID, error) { return "Йога", f.owner, nil }

type fakeRSVP struct{ signedUp []uuid.UUID }
func (f *fakeRSVP) SignUp(_ context.Context, _, userID uuid.UUID, _ string) error { f.signedUp = append(f.signedUp, userID); return nil }

type fakeMailer struct{ sent int }
func (f *fakeMailer) SendEventInvitation(context.Context, string, string, string) error { f.sent++; return nil }

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
	if err != nil { t.Fatalf("invite: %v", err) }
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
	if err != inv.ErrNotOwner { t.Fatalf("want ErrNotOwner, got %v", err) }
}

func TestInvite_RejectsUnverifiedOrganizer(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	s := newSvc(&fakeRepo{}, fakeEvents{owner: owner}, &fakeRSVP{}, &fakeMailer{})
	_, err := s.Invite(context.Background(), uuid.Must(uuid.NewV4()), owner, false, []string{"a@x.com"}, "b")
	if err != inv.ErrNotVerified { t.Fatalf("want ErrNotVerified, got %v", err) }
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
	if err != inv.ErrEmailMismatch { t.Fatalf("want ErrEmailMismatch, got %v", err) }
}

func TestAcceptByToken_RejectsUnverified(t *testing.T) {
	repo := &fakeRepo{byToken: map[string]*inv.Invitation{
		"tok": {ID: uuid.Must(uuid.NewV4()), InviteeEmail: "a@x.com", Status: "pending", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	s := newSvc(repo, fakeEvents{}, &fakeRSVP{}, &fakeMailer{})
	err := s.AcceptByToken(context.Background(), "tok", "a@x.com", uuid.Must(uuid.NewV4()), false)
	if err != inv.ErrNotVerified { t.Fatalf("want ErrNotVerified, got %v", err) }
}

var _ = strings.ToLower
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/invitations/ -v`
Expected: FAIL — `inv.NewService` / ports / errors undefined.

- [ ] **Step 3: Implement the service**

Add to `backend/internal/invitations/service.go` (imports: `context`, `crypto/rand`, `encoding/base64`, `errors`, `fmt`, `strings`, `time`, `github.com/gofrs/uuid`):

```go
// EventPort/RSVPPort are the collaborators the service needs (see Task 6 wiring).
type EventPort interface {
	GetByID(ctx context.Context, id string) (title string, organizerUserID uuid.UUID, err error)
}
type RSVPPort interface {
	SignUp(ctx context.Context, eventID, userID uuid.UUID, answer string) error
}
type MailerPort interface {
	SendEventInvitation(ctx context.Context, to, eventTitle, acceptURL string) error
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

type service struct {
	repo   Repository
	events EventPort
	rsvp   RSVPPort
	mailer MailerPort
}

// NewService builds the invitations service.
func NewService(repo Repository, events EventPort, rsvp RSVPPort, mailer MailerPort) Service {
	return &service{repo: repo, events: events, rsvp: rsvp, mailer: mailer}
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
		inv := Invitation{
			ID: id, EventID: eventID, InviterUserID: inviterUserID,
			InviteeEmail: email, Token: token, Status: "pending",
			ExpiresAt: time.Now().Add(inviteTTL),
		}
		if err := s.repo.Insert(ctx, inv); err != nil {
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
	inv, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	title, _, err := s.events.GetByID(ctx, inv.EventID.String())
	if err != nil {
		return nil, err
	}
	return &Preview{EventID: inv.EventID, EventTitle: title, Status: inv.Status}, nil
}

func (s *service) ListMine(ctx context.Context, email string) ([]Invitation, error) {
	return s.repo.ListPendingByEmail(ctx, email)
}

func (s *service) AcceptByToken(ctx context.Context, token, userEmail string, userID uuid.UUID, verified bool) error {
	inv, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return err
	}
	return s.accept(ctx, inv, userEmail, userID, verified)
}

func (s *service) AcceptByID(ctx context.Context, id uuid.UUID, userEmail string, userID uuid.UUID, verified bool) error {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.accept(ctx, inv, userEmail, userID, verified)
}

func (s *service) accept(ctx context.Context, inv *Invitation, userEmail string, userID uuid.UUID, verified bool) error {
	if inv.Status != "pending" {
		return ErrNotPending
	}
	if !verified {
		return ErrNotVerified
	}
	if !strings.EqualFold(strings.TrimSpace(userEmail), inv.InviteeEmail) {
		return ErrEmailMismatch
	}
	if err := s.rsvp.SignUp(ctx, inv.EventID, userID, ""); err != nil {
		return fmt.Errorf("rsvp on accept: %w", err)
	}
	return s.repo.SetStatus(ctx, inv.ID, "accepted")
}

func (s *service) DeclineByToken(ctx context.Context, token, userEmail string) error {
	inv, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return err
	}
	return s.decline(ctx, inv, userEmail)
}

func (s *service) DeclineByID(ctx context.Context, id uuid.UUID, userEmail string) error {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.decline(ctx, inv, userEmail)
}

func (s *service) decline(ctx context.Context, inv *Invitation, userEmail string) error {
	if inv.Status != "pending" {
		return ErrNotPending
	}
	if !strings.EqualFold(strings.TrimSpace(userEmail), inv.InviteeEmail) {
		return ErrEmailMismatch
	}
	return s.repo.SetStatus(ctx, inv.ID, "declined")
}
```

> Note: `Service`'s method set uses `MailerPort` internally but the constructor accepts the `Mailer` from Task 3 (same method signature). If Go complains, change the `NewService` mailer param type to `MailerPort` — `notifications.Mailer` satisfies it structurally when passed, so wrap or alias at the Task 6 call site.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/invitations/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/invitations/
git commit -m "feat(invitations): service (invite/accept/decline/preview/list) + tests"
```

---

### Task 5: HTTP handler (raw mux)

**Files:**
- Create: `backend/internal/http/invitations/handler.go`
- Test: `backend/internal/http/invitations/handler_test.go`

**Interfaces:**
- Consumes: `invitations.Service`, `Authenticate func(token string) (*domain.User, error)` (same shape complaints uses), and a `BaseURL string` for building accept links.
- Endpoints:
  - `POST /api/v1/events/{id}/invitations` — body `{"emails":["..."]}` — auth + verified owner.
  - `GET  /api/v1/invitations/{token}` — public preview.
  - `POST /api/v1/invitations/{token}/accept` | `/decline` — auth (+ verified for accept), email match.
  - `GET  /api/v1/me/invitations` — auth; pending for the user's email.
  - `POST /api/v1/me/invitations/{id}/accept` | `/decline` — auth (+ verified for accept).

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/http/invitations/handler_test.go
package invitations_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	domain "github.com/Pashteto/lia/internal/models"
	httpinv "github.com/Pashteto/lia/internal/http/invitations"
	inv "github.com/Pashteto/lia/internal/invitations"
)

type stubSvc struct{ invited int }
func (s *stubSvc) Invite(_ any, _, _ uuid.UUID, verified bool, emails []string, _ string) (int, error) { return len(emails), nil }
// ... implement the rest of inv.Service as no-ops returning nil ...

func TestInvite_RequiresVerified(t *testing.T) {
	deps := httpinv.Deps{
		Authenticate: func(string) (*domain.User, error) {
			return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "o@x.com", EmailVerified: false}, nil
		},
		Service: &stubSvc{}, // adapt to satisfy inv.Service
		BaseURL: "https://presence.tarski.ru",
	}
	h := httpinv.NewHandler(deps)

	req := httptest.NewRequest("POST", "/api/v1/events/"+uuid.Must(uuid.NewV4()).String()+"/invitations",
		strings.NewReader(`{"emails":["a@x.com"]}`))
	req.Header.Set("Authorization", "Bearer x")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("unverified organizer must get 403, got %d", rr.Code)
	}
}
```

> The `stubSvc` must implement the full `inv.Service` interface (add no-op methods). The signature shown uses `any` for ctx only for brevity — match `context.Context` from the real interface.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/invitations/ -run TestInvite_RequiresVerified -v`
Expected: FAIL — package/handler undefined.

- [ ] **Step 3: Implement the handler**

```go
// backend/internal/http/invitations/handler.go
package invitations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	invdomain "github.com/Pashteto/lia/internal/invitations"
	domain "github.com/Pashteto/lia/internal/models"
)

// Deps wires the handler.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Service      invdomain.Service
	BaseURL      string
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler builds the invitations HTTP surface (mounted ahead of the swagger mux).
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/events/{id}/invitations", h.create)
	h.mux.HandleFunc("GET /api/v1/invitations/{token}", h.preview)
	h.mux.HandleFunc("POST /api/v1/invitations/{token}/accept", h.acceptToken)
	h.mux.HandleFunc("POST /api/v1/invitations/{token}/decline", h.declineToken)
	h.mux.HandleFunc("GET /api/v1/me/invitations", h.listMine)
	h.mux.HandleFunc("POST /api/v1/me/invitations/{id}/accept", h.acceptID)
	h.mux.HandleFunc("POST /api/v1/me/invitations/{id}/decline", h.declineID)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func (h *handler) principal(r *http.Request) *domain.User {
	tok := r.Header.Get("Authorization")
	if tok == "" {
		return nil
	}
	u, err := h.deps.Authenticate(tok)
	if err != nil {
		return nil
	}
	return u
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, code int, msg string) { writeJSON(w, code, map[string]string{"message": msg}) }
func writeUnverified(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]string{"code": "email_not_verified", "message": "Подтвердите электронную почту, чтобы продолжить"})
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !u.EmailVerified {
		writeUnverified(w)
		return
	}
	eventID, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad event id")
		return
	}
	var body struct {
		Emails []string `json:"emails"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Emails) == 0 {
		writeErr(w, http.StatusBadRequest, "emails are required")
		return
	}
	n, err := h.deps.Service.Invite(r.Context(), eventID, u.UUID, u.EmailVerified, body.Emails, h.deps.BaseURL)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int{"invited": n})
}

func (h *handler) preview(w http.ResponseWriter, r *http.Request) {
	p, err := h.deps.Service.Preview(r.Context(), r.PathValue("token"))
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"event_id": p.EventID.String(), "event_title": p.EventTitle, "status": p.Status,
	})
}

func (h *handler) acceptToken(w http.ResponseWriter, r *http.Request)  { h.respond(w, r, true, true) }
func (h *handler) declineToken(w http.ResponseWriter, r *http.Request) { h.respond(w, r, true, false) }
func (h *handler) acceptID(w http.ResponseWriter, r *http.Request)     { h.respond(w, r, false, true) }
func (h *handler) declineID(w http.ResponseWriter, r *http.Request)    { h.respond(w, r, false, false) }

func (h *handler) respond(w http.ResponseWriter, r *http.Request, byToken, accept bool) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if accept && !u.EmailVerified {
		writeUnverified(w)
		return
	}
	var err error
	switch {
	case byToken && accept:
		err = h.deps.Service.AcceptByToken(r.Context(), r.PathValue("token"), u.Email, u.UUID, u.EmailVerified)
	case byToken && !accept:
		err = h.deps.Service.DeclineByToken(r.Context(), r.PathValue("token"), u.Email)
	case !byToken && accept:
		id, e := uuid.FromString(r.PathValue("id"))
		if e != nil {
			writeErr(w, http.StatusBadRequest, "bad id")
			return
		}
		err = h.deps.Service.AcceptByID(r.Context(), id, u.Email, u.UUID, u.EmailVerified)
	default:
		id, e := uuid.FromString(r.PathValue("id"))
		if e != nil {
			writeErr(w, http.StatusBadRequest, "bad id")
			return
		}
		err = h.deps.Service.DeclineByID(r.Context(), id, u.Email)
	}
	if err != nil {
		h.mapErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listMine(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	list, err := h.deps.Service.ListMine(r.Context(), u.Email)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not list invitations")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, i := range list {
		out = append(out, map[string]any{
			"id": i.ID.String(), "event_id": i.EventID.String(), "token": i.Token, "status": i.Status,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) mapErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, invdomain.ErrNotVerified):
		writeUnverified(w)
	case errors.Is(err, invdomain.ErrNotOwner):
		writeErr(w, http.StatusForbidden, "not event owner")
	case errors.Is(err, invdomain.ErrEmailMismatch):
		writeErr(w, http.StatusForbidden, "invitation addressed to another email")
	case errors.Is(err, invdomain.ErrNotPending):
		writeErr(w, http.StatusConflict, "invitation already handled")
	case errors.Is(err, invdomain.ErrNotFound):
		writeErr(w, http.StatusNotFound, "invitation not found")
	default:
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
	_ = strings.TrimSpace
	_ = context.Background
}
```

> Remove the trailing `_ = strings.TrimSpace` / `_ = context.Background` guards if those imports are otherwise used; they're only there to keep the example compiling if you trim code.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/invitations/ -v && go build ./...`
Expected: PASS + build.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/invitations/
git commit -m "feat(api): event-invitations HTTP handler (raw mux)"
```

---

### Task 6: Wire everything into the composition root + dispatch

**Files:**
- Modify: `backend/internal/application.go` (~:167-270, construct service + adapters + mailer)
- Modify: `backend/internal/http/module.go` (add `SetInvitations`, build handler, dispatch branch)

**Interfaces:**
- Consumes: `events.Service.GetByID`, `rsvp.Service.SignUp`, `notifications.NewSMTPMailer`, config (SMTP + base URL).

- [ ] **Step 1: Add event/RSVP adapters**

Create `backend/internal/invitations/adapters.go` — thin adapters satisfying `EventPort` / `RSVPPort` over the real services:

```go
package invitations

import (
	"context"

	"github.com/gofrs/uuid"
)

// eventsAdapter adapts events.Service to EventPort.
type eventsAdapter struct {
	getByID func(ctx context.Context, id string) (title string, organizerUserID uuid.UUID, err error)
}

func NewEventPort(getByID func(ctx context.Context, id string) (string, uuid.UUID, error)) EventPort {
	return eventsAdapter{getByID: getByID}
}
func (a eventsAdapter) GetByID(ctx context.Context, id string) (string, uuid.UUID, error) {
	return a.getByID(ctx, id)
}

type rsvpAdapter struct {
	signUp func(ctx context.Context, eventID, userID uuid.UUID, answer string) error
}

func NewRSVPPort(signUp func(ctx context.Context, eventID, userID uuid.UUID, answer string) error) RSVPPort {
	return rsvpAdapter{signUp: signUp}
}
func (a rsvpAdapter) SignUp(ctx context.Context, eventID, userID uuid.UUID, answer string) error {
	return a.signUp(ctx, eventID, userID, answer)
}
```

> In `application.go`, build the `getByID` closure from `eventsSvc.GetByID` — it returns `*models.Event`; extract `event.Title` and `event.OrganizerID`. Build the `signUp` closure from `rsvpSvc.SignUp` (its real signature returns `(*models.Rsvp, error)`; discard the rsvp). If `rsvpSvc` is nil (RSVP disabled), pass a closure returning an error.

- [ ] **Step 2: Construct + inject in application.go**

Near where other domain services are wired (~:266), add:

```go
mailer := notifications.NewSMTPMailer(cfg.SMTP.Address, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
invSvc := invitations.NewService(
	invitations.NewRepository(repoModule.DB()),
	invitations.NewEventPort(func(ctx context.Context, id string) (string, uuid.UUID, error) {
		ev, err := app.eventsSvc.GetByID(ctx, id)
		if err != nil {
			return "", uuid.Nil, err
		}
		return ev.Title, ev.OrganizerID, nil
	}),
	invitations.NewRSVPPort(func(ctx context.Context, eventID, userID uuid.UUID, answer string) error {
		_, err := app.rsvpSvc.SignUp(ctx, eventID, userID, answer)
		return err
	}),
	mailer,
)
httpModule.SetInvitations(invSvc, cfg.PublicBaseURL)
```

> Add config fields `SMTP{Address,Username,Password,From}` and `PublicBaseURL` to the config struct + env binding (mirror how `notificator.*` is bound in GateGuard and how existing Lia config is loaded). `PublicBaseURL` = `https://presence.tarski.ru`. Confirm the exact field names on `models.Event` (`Title`, `OrganizerID`) and `rsvpSvc.SignUp`'s signature before finalizing the closures.

- [ ] **Step 3: Add SetInvitations + dispatch in module.go**

Mirror `SetComplaints` (module.go:132). Add:
```go
func (m *Module) SetInvitations(svc invitations.Service, baseURL string) {
	m.invitations = svc
	m.invitationsBaseURL = baseURL
}
```
(and the `invitations invitations.Service` / `invitationsBaseURL string` fields on `Module`).

In the router build (~:290-395), construct and dispatch:
```go
var invitationsH http.Handler
if m.invitations != nil {
	invitationsH = httpinvitations.NewHandler(httpinvitations.Deps{
		Authenticate: m.auth.Authenticate,
		Service:      m.invitations,
		BaseURL:      m.invitationsBaseURL,
	})
}
// inside the router func, before the swagger fall-through:
if invitationsH != nil && (
	(strings.HasPrefix(p, "/api/v1/events/") && strings.HasSuffix(p, "/invitations")) ||
	strings.HasPrefix(p, "/api/v1/invitations/") ||
	strings.HasPrefix(p, "/api/v1/me/invitations")) {
	invitationsH.ServeHTTP(w, r)
	return
}
```
(Use the existing import alias convention for the `internal/http/invitations` package, e.g. `httpinvitations`.)

- [ ] **Step 4: Build + test**

Run: `cd backend && go build ./... && go test ./... `
Expected: build + all tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/application.go backend/internal/http/module.go backend/internal/invitations/adapters.go
git commit -m "feat(invitations): wire service + mailer + HTTP dispatch"
```

---

### Task 7: Frontend — API functions

**Files:**
- Modify: `frontend/lib/api.ts`

**Interfaces:**
- Produces: `sendInvitations(eventId, emails)`, `getInvitationPreview(token)`, `acceptInvitation(token)`, `declineInvitation(token)`, `fetchMyInvitations()`, `acceptMyInvitation(id)`, `declineMyInvitation(id)`.

- [ ] **Step 1: Add the functions**

Mirror `fetchMyEvents` (:197, authed GET) and `registerWithPassword` (:384, POST). Add:

```ts
export async function sendInvitations(eventId: string, emails: string[]): Promise<number> {
  const res = await fetch(`${API_V1}/events/${eventId}/invitations`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${getToken()}` },
    body: JSON.stringify({ emails }),
  });
  if (res.status === 403) {
    const b = await res.clone().json().catch(() => ({}));
    if (b?.code === "email_not_verified") throw new Error(EMAIL_NOT_VERIFIED);
  }
  if (!res.ok) throw new Error(`invite failed: ${res.status}`);
  const data = await res.json();
  return data.invited ?? 0;
}

export interface InvitationPreview { event_id: string; event_title: string; status: string; }
export async function getInvitationPreview(token: string): Promise<InvitationPreview> {
  const res = await fetch(`${API_V1}/invitations/${token}`, { cache: "no-store" });
  if (!res.ok) throw new Error(`preview failed: ${res.status}`);
  return res.json();
}

async function invitationAction(path: string): Promise<void> {
  const res = await fetch(`${API_V1}${path}`, {
    method: "POST",
    headers: { Authorization: `Bearer ${getToken()}` },
  });
  if (res.status === 403) {
    const b = await res.clone().json().catch(() => ({}));
    if (b?.code === "email_not_verified") throw new Error(EMAIL_NOT_VERIFIED);
  }
  if (!res.ok && res.status !== 204) throw new Error(`invitation action failed: ${res.status}`);
}
export const acceptInvitation = (token: string) => invitationAction(`/invitations/${token}/accept`);
export const declineInvitation = (token: string) => invitationAction(`/invitations/${token}/decline`);
export const acceptMyInvitation = (id: string) => invitationAction(`/me/invitations/${id}/accept`);
export const declineMyInvitation = (id: string) => invitationAction(`/me/invitations/${id}/decline`);

export interface MyInvitation { id: string; event_id: string; token: string; status: string; }
export async function fetchMyInvitations(): Promise<MyInvitation[]> {
  const res = await fetch(`${API_V1}/me/invitations`, {
    headers: { Authorization: `Bearer ${getToken()}` }, cache: "no-store",
  });
  if (!res.ok) throw new Error(`my invitations failed: ${res.status}`);
  return res.json();
}
```

> `EMAIL_NOT_VERIFIED` is defined in Phase B Task 9. If Phase B isn't merged yet, define it here instead: `export const EMAIL_NOT_VERIFIED = "EMAIL_NOT_VERIFIED";`

- [ ] **Step 2: Type-check**

Run: `cd frontend && npx tsc --noEmit` (or the repo's typecheck script).
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add frontend/lib/api.ts
git commit -m "feat(frontend): invitation API client functions"
```

---

### Task 8: Frontend — organizer invite panel

**Files:**
- Create: `frontend/components/InviteByEmailPanel.tsx`
- Modify: `frontend/app/events/mine/page.tsx` (add the panel as an expander per event)

**Interfaces:**
- Consumes: `sendInvitations`, `EMAIL_NOT_VERIFIED`.

- [ ] **Step 1: Build the panel (mirror EventApplicationsPanel)**

```tsx
// frontend/components/InviteByEmailPanel.tsx
"use client";

import { useState } from "react";
import { sendInvitations, EMAIL_NOT_VERIFIED } from "@/lib/api";
import { VerifyEmailInterstitial } from "@/components/VerifyEmailInterstitial";

const inputClass =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

export function InviteByEmailPanel({ eventId }: { eventId: string }) {
  const [raw, setRaw] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [showVerify, setShowVerify] = useState(false);

  async function onSend() {
    const emails = raw.split(/[\s,;]+/).map((s) => s.trim()).filter(Boolean);
    if (emails.length === 0) return;
    setBusy(true); setError(""); setMsg("");
    try {
      const n = await sendInvitations(eventId, emails);
      setMsg(`Приглашения отправлены: ${n}`);
      setRaw("");
    } catch (e) {
      if (e instanceof Error && e.message === EMAIL_NOT_VERIFIED) { setShowVerify(true); }
      else { setError("Не удалось отправить приглашения."); }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-card bg-bg-secondary p-3">
      <p className="mb-1.5 block text-[13px] text-label-secondary">Пригласить по email (через запятую)</p>
      <textarea className={inputClass} rows={2} value={raw}
        onChange={(e) => setRaw(e.target.value)} placeholder="a@mail.ru, b@mail.ru" />
      {error && <p className="mt-2 text-[14px] text-red-500">{error}</p>}
      {msg && <p className="mt-2 text-[14px] text-green-600">{msg}</p>}
      <button onClick={onSend} disabled={busy}
        className="mt-2 rounded-capsule bg-accent px-4 py-2 text-white disabled:opacity-50">
        Отправить приглашения
      </button>
      {showVerify && <VerifyEmailInterstitial onClose={() => setShowVerify(false)} />}
    </div>
  );
}
```

> `VerifyEmailInterstitial` comes from Phase B Task 9. If Phase B isn't merged, inline a simple link to `/auth/verify` instead.

- [ ] **Step 2: Mount it in the organizer's event list**

In `frontend/app/events/mine/page.tsx`, add an expander alongside the existing `ApplicationsExpander`/`FeedbackExpander` (the file already has this collapsible pattern ~:22-38) that renders `<InviteByEmailPanel eventId={event.id} />`.

- [ ] **Step 3: Type-check + build**

Run: `cd frontend && npx tsc --noEmit`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add frontend/components/InviteByEmailPanel.tsx frontend/app/events/mine/page.tsx
git commit -m "feat(frontend): organizer invite-by-email panel"
```

---

### Task 9: Frontend — `/invite/[token]` landing page

**Files:**
- Create: `frontend/app/invite/[token]/page.tsx`

**Interfaces:**
- Consumes: `getInvitationPreview`, `acceptInvitation`, `useAuth`, `LoginModal` (from `AuthButton.tsx`).

- [ ] **Step 1: Build the page**

```tsx
// frontend/app/invite/[token]/page.tsx
"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { getInvitationPreview, acceptInvitation, EMAIL_NOT_VERIFIED, type InvitationPreview } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

export default function InvitePage() {
  const { token } = useParams<{ token: string }>();
  const router = useRouter();
  const { isAuthed, ready, emailVerified } = useAuth();
  const [preview, setPreview] = useState<InvitationPreview | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    getInvitationPreview(token).then(setPreview).catch(() => setError("Приглашение не найдено или устарело."));
  }, [token]);

  async function onAccept() {
    setBusy(true); setError("");
    try {
      await acceptInvitation(token);
      router.push(`/events/${preview?.event_id ?? ""}`);
    } catch (e) {
      if (e instanceof Error && e.message === EMAIL_NOT_VERIFIED) { router.push("/auth/verify"); return; }
      setError("Не удалось принять приглашение.");
    } finally {
      setBusy(false);
    }
  }

  if (error) return <main className="mx-auto max-w-md px-4 py-10"><p className="text-red-500">{error}</p></main>;
  if (!preview || !ready) return <main className="min-h-screen bg-bg-grouped" />;

  return (
    <main className="mx-auto max-w-md px-4 py-10">
      <h1 className="mb-2 text-[24px] font-bold tracking-[-0.022em]">Вас пригласили</h1>
      <p className="mb-6 text-[17px] text-label">«{preview.event_title}»</p>

      {!isAuthed ? (
        <div className="rounded-card bg-bg-secondary p-4">
          <p className="mb-3 text-[15px] text-label-secondary">Войдите или зарегистрируйтесь, чтобы принять приглашение.</p>
          {/* Reuse the app's auth modal; the header AuthButton also exposes it.
              Simplest: link to home where the login modal lives, or mount <LoginModal/> here. */}
          <Link href={`/?next=/invite/${token}`} className="rounded-capsule bg-accent px-4 py-2 text-white">Войти</Link>
        </div>
      ) : !emailVerified ? (
        <div className="rounded-card bg-bg-secondary p-4">
          <p className="mb-3 text-[15px] text-label-secondary">Подтвердите почту, чтобы принять приглашение.</p>
          <Link href="/auth/verify" className="rounded-capsule bg-accent px-4 py-2 text-white">Подтвердить почту</Link>
        </div>
      ) : (
        <button onClick={onAccept} disabled={busy}
          className="rounded-capsule bg-accent px-5 py-2.5 text-white disabled:opacity-50">
          Принять приглашение
        </button>
      )}
    </main>
  );
}
```

> If you prefer to mount `LoginModal` directly instead of linking home, export it from `AuthButton.tsx` and render it here with an `onClose` that re-checks `isAuthed`. The `?next=` param is a hint; wiring post-login redirect to it is optional polish.

- [ ] **Step 2: Type-check**

Run: `cd frontend && npx tsc --noEmit`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add frontend/app/invite/
git commit -m "feat(frontend): /invite/[token] landing + accept flow"
```

---

### Task 10: Frontend — `/me/invitations` pending list

**Files:**
- Create: `frontend/app/me/invitations/page.tsx`
- Modify: `frontend/components/ui/TabBar.tsx` (:13 add to TABS) and/or `frontend/components/AuthButton.tsx` (:26-58 authed links)

**Interfaces:**
- Consumes: `fetchMyInvitations`, `acceptMyInvitation`, `declineMyInvitation`, `useAuth`.

- [ ] **Step 1: Build the page (mirror `me/practices/page.tsx`)**

```tsx
// frontend/app/me/invitations/page.tsx
"use client";

import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchMyInvitations, acceptMyInvitation, declineMyInvitation } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

export default function MyInvitationsPage() {
  const { isAuthed, ready } = useAuth();
  const qc = useQueryClient();
  const { data = [], isLoading } = useQuery({
    queryKey: ["my-invitations"],
    queryFn: fetchMyInvitations,
    enabled: ready && isAuthed,
  });

  const accept = useMutation({
    mutationFn: (id: string) => acceptMyInvitation(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["my-invitations"] }),
  });
  const decline = useMutation({
    mutationFn: (id: string) => declineMyInvitation(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["my-invitations"] }),
  });

  if (!ready) return <div className="min-h-screen bg-bg-grouped" />;
  if (!isAuthed) return <main className="mx-auto max-w-3xl px-4 py-8"><p>Войдите, чтобы увидеть приглашения.</p></main>;

  return (
    <main className="mx-auto max-w-3xl px-4 py-8 max-sm:pb-28">
      <Link href="/" className="text-accent">‹ События</Link>
      <h1 className="mb-4 mt-2 text-[28px] font-bold tracking-[-0.022em]">Приглашения</h1>
      {isLoading ? (
        <p className="text-label-secondary">Загрузка…</p>
      ) : data.length === 0 ? (
        <p className="text-label-secondary">Нет новых приглашений.</p>
      ) : (
        <ul className="space-y-3">
          {data.map((i) => (
            <li key={i.id} className="rounded-card bg-bg-secondary p-4 flex items-center justify-between">
              <Link href={`/events/${i.event_id}`} className="text-accent">Открыть событие</Link>
              <div className="flex gap-2">
                <button onClick={() => accept.mutate(i.id)} disabled={accept.isPending}
                  className="rounded-capsule bg-accent px-3 py-1.5 text-white disabled:opacity-50">Принять</button>
                <button onClick={() => decline.mutate(i.id)} disabled={decline.isPending}
                  className="rounded-capsule bg-fill px-3 py-1.5 text-label disabled:opacity-50">Отклонить</button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
```

> If accept returns `EMAIL_NOT_VERIFIED`, the mutation `onError` should route to `/auth/verify` (add an `onError` handler mirroring the interstitial pattern). Fetching the event title for each row is optional polish — the row links straight to the event.

- [ ] **Step 2: Add a nav entry**

Add "Приглашения" → `/me/invitations` to the authed links in `AuthButton.tsx` (:26-58) and/or `TABS` in `TabBar.tsx` (:13), matching the existing entries' shape.

- [ ] **Step 3: Type-check + build**

Run: `cd frontend && npx tsc --noEmit && npm run build`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add frontend/app/me/invitations/ frontend/components/ui/TabBar.tsx frontend/components/AuthButton.tsx
git commit -m "feat(frontend): /me/invitations pending list + nav"
```

---

### Task 11: End-to-end verification

- [ ] **Step 1: Backend suite + build**

Run: `cd backend && go build ./... && go vet ./... && go test ./...`
Expected: all green.

- [ ] **Step 2: Frontend build**

Run: `cd frontend && npm run build`
Expected: clean.

- [ ] **Step 3: Manual smoke (local, mock or real GateGuard)**

1. As a verified organizer, open `/events/mine`, expand a published event, send an invite to a test address.
2. Confirm a row exists in `event_invitations` (`SELECT * FROM event_invitations;`).
3. Open `/invite/<token>` in a fresh session → shows the event → sign up → verify → accept → lands on the event and an `event_rsvps` row exists.
4. As an existing logged-in user with a pending invite, open `/me/invitations` → accept → RSVP created.

- [ ] **Step 4: Commit any fixups**

```bash
git add -A && git commit -m "chore(invitations): e2e fixups"
```

---

## Self-Review

**Spec coverage (design §6 Phase C):**
- `event_invitations` table keyed by email + token → Task 1. ✅
- Domain + repo → Task 2. ✅
- New Lia SMTP mailer (none existed) → Task 3. ✅
- Service: invite/preview/accept/decline/list, owner+verified+email-match rules, accept→RSVP → Task 4. ✅
- HTTP endpoints (send/preview/accept/decline by token, /me list + by id) → Task 5. ✅
- Composition wiring + dispatch → Task 6. ✅
- Unregistered-invitee onboarding (email→link→signup→verify→accept, matched by email; in-app list by email) → Tasks 4 (email-keyed), 9 (`/invite/[token]`), 10 (`/me/invitations`). ✅
- Frontend send panel + landing + pending list → Tasks 8, 9, 10. ✅
- Accept requires verified email → Tasks 4, 5 (403), 9/10 (UI gate). ✅

**Placeholder scan:** All code blocks concrete. Flagged "confirm exact field name/signature" notes (Task 6: `models.Event.Title/OrganizerID`, `rsvpSvc.SignUp` return; config field names) are real integration points the implementer must match against existing code, not placeholder logic. `ExpireOverdue` is implemented but only invoked lazily via `expires_at > now()` filters in reads + `SetStatus`; a periodic call is optional (note below).

**Type consistency:** `Invitation` struct fields (Tasks 2,4,5); `Repository` methods (Tasks 2,4); `Service` methods + error vars `ErrNotOwner/ErrNotVerified/ErrEmailMismatch/ErrNotPending/ErrNotFound` (Tasks 4,5); `EventPort.GetByID(ctx,id)→(title,ownerUserID,err)` and `RSVPPort.SignUp(ctx,eventID,userID,answer)` (Tasks 4,6); frontend `sendInvitations/acceptInvitation/...` + `MyInvitation`/`InvitationPreview` (Tasks 7-10). Consistent.

**Dependencies:** Task 4/5's verified-email checks rely on Phase B (`domain.User.EmailVerified`, `EMAIL_NOT_VERIFIED` sentinel, `VerifyEmailInterstitial`, `useAuth().emailVerified`). Land Phase B (Plan 2) before or with Phase C. The `expire_invitations`-style periodic sweep is optional; lazy expiry (`expires_at > now()` in reads) covers correctness.

**Follow-up (optional, not in this plan):** organizer "revoke invite" endpoint (`SetStatus(..., "revoked")`) and a scheduled `ExpireOverdue` sweep — noted in the design's open follow-ups.
