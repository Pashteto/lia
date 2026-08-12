# External Registration Whitelist Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status: EXECUTED & DEPLOYED LIVE 2026-08-12** (all tasks complete, final review clean; runbook `docs/superpowers/runbooks/2026-08-12-external-registration-deploy.md`)

**Goal:** Whitelist of trusted RU ticketing platforms for `signup_mode=external` events: known domain → publish immediately with platform-branded CTA; unknown domain → `pending_review` moderation; plus «Оплата на месте» label for paid open-signup events, a hidden-capacity flag, and a CTA-column layout fix.

**Architecture:** New `internal/platforms` module (pure suffix matcher + go-pg repo + service) consulted synchronously by the events service on create/update. Read path enriches events with the matched platform display name. Moderation gets an `Approve` transition (`pending_review → published`). New public endpoint `/api/v1/trusted-platforms` (plain handler, mirrors `/api/v1/places`), admin CRUD under `/api/v1/admin/trusted-platforms`.

**Tech Stack:** Go (go-pg v10, go-swagger codegen, `golang.org/x/net/idna` — already in go.mod), PostgreSQL (migration 000026), Next.js frontend (zod, react-hook-form, vitest).

**Spec:** `docs/superpowers/specs/2026-08-12-external-registration-design.md` — read it first.

## Global Constraints

- Suffix matching is done on **punycode** hosts; seed rows store punycode (`xn--80atdujec4e.xn--p1ai` for культура.рф).
- URL hard-validation (the only 422): must be `https://`, no userinfo, host must not be an IP. Everything else goes the moderation path — an unknown domain must NEVER produce an error or lose data.
- Snapshot semantics: `events.external_url_verified` is set when the URL is (re)checked; whitelist edits do not retroactively change published events. Re-check happens only when the URL **changes** (or on first publish). Admin approve sets `external_url_verified = true`.
- Exact RU copy (verbatim): «Билеты на {name}» (платное), «Записаться через {name}» (бесплатное), «Записаться на сайте организатора», «Переход на {домен}», «Ограничено · наличие на сайте регистрации», «Оплата на месте», «Платформа: {name}», «Неизвестная платформа — событие уйдёт на проверку модератору», «Событие отправлено на проверку модератору и опубликуется после одобрения».
- Backend commands run from `backend/`: tests `go test ./internal/... `, lint `golangci-lint run` (v1 — see memory `lia-dev-gotchas`), codegen `make generate-api` (only Task 6 needs it). Frontend commands run from `frontend/`: `npx vitest run`.
- Commit after every task (small commits, messages given per task).
- `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` on every commit.

---

### Task 1: Migration 000026 — trusted_platforms table + events columns + seed

**Files:**
- Create: `backend/db/migrations/000026_trusted_platforms.up.sql`
- Create: `backend/db/migrations/000026_trusted_platforms.down.sql`

**Interfaces:**
- Produces: table `trusted_platforms(id uuid, domain_suffix text, display_name text, category text, is_active bool, created_at timestamptz)`; columns `events.capacity_limited boolean`, `events.external_url_verified boolean`. Later tasks rely on these exact names.

- [x] **Step 1: Write the up migration**

```sql
-- 000026_trusted_platforms.up.sql
-- Trusted external-registration platforms (spec 2026-08-12). domain_suffix is
-- stored in punycode, no scheme; matching is exact host or dot-boundary suffix.
CREATE TABLE IF NOT EXISTS trusted_platforms (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_suffix text NOT NULL UNIQUE,
    display_name  text NOT NULL,
    category      text NOT NULL CHECK (category IN ('ticketing', 'afisha', 'gov', 'social')),
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE events ADD COLUMN IF NOT EXISTS capacity_limited boolean NOT NULL DEFAULT false;
ALTER TABLE events ADD COLUMN IF NOT EXISTS external_url_verified boolean NOT NULL DEFAULT false;

INSERT INTO trusted_platforms (domain_suffix, display_name, category) VALUES
  ('timepad.ru',            'TimePad',          'ticketing'),
  ('afisha.yandex.ru',      'Яндекс Афиша',     'afisha'),
  ('kassir.ru',             'Кассир.ру',        'ticketing'),
  ('qtickets.ru',           'Qtickets',         'ticketing'),
  ('qtickets.events',       'Qtickets',         'ticketing'),
  ('ticketscloud.com',      'Ticketscloud',     'ticketing'),
  ('ticketscloud.org',      'Ticketscloud',     'ticketing'),
  ('intickets.ru',          'Intickets',        'ticketing'),
  ('radario.ru',            'Радарио',          'ticketing'),
  ('ticketland.ru',         'Ticketland',       'ticketing'),
  ('ponominalu.ru',         'Ponominalu',       'ticketing'),
  ('live.mts.ru',           'МТС Live',         'ticketing'),
  ('afisha.ru',             'Афиша',            'afisha'),
  ('kinopoisk.ru',          'Кинопоиск',        'afisha'),
  ('vk.com',                'ВКонтакте',        'social'),
  ('events.nethouse.ru',    'Nethouse.События', 'ticketing'),
  ('leader-id.ru',          'Leader-ID',        'gov'),
  ('culture.ru',            'Культура.РФ',      'gov'),
  ('xn--80atdujec4e.xn--p1ai',   'Культура.РФ',      'gov'),
  ('mos.ru',                'mos.ru',           'gov'),
  ('vmuzey.com',            'ВМузей',           'afisha'),
  ('kudago.com',            'KudaGo',           'afisha')
ON CONFLICT (domain_suffix) DO NOTHING;
```

- [x] **Step 2: Write the down migration**

```sql
-- 000026_trusted_platforms.down.sql
ALTER TABLE events DROP COLUMN IF EXISTS external_url_verified;
ALTER TABLE events DROP COLUMN IF EXISTS capacity_limited;
DROP TABLE IF EXISTS trusted_platforms;
```

- [x] **Step 3: Apply to the local dev DB**

Run from `backend/`: `make migrate-up` (uses `$(DATABASE_URL)`; if the local Docker postgres is down, start it first — see `backend/README.md`; local Docker can be flaky per memory `lia-dev-gotchas`, host-run postgres is the workaround).
Expected: `26/u trusted_platforms` applied. Verify: `psql "$DATABASE_URL" -c "select count(*) from trusted_platforms"` → 22.

- [x] **Step 4: Commit**

```bash
git add backend/db/migrations/000026_trusted_platforms.*
git commit -m "feat(db): trusted_platforms whitelist + capacity_limited/external_url_verified on events"
```

---

### Task 2: `internal/platforms` — URL normalization + suffix matcher (pure functions)

**Files:**
- Create: `backend/internal/platforms/match.go`
- Test: `backend/internal/platforms/match_test.go`

**Interfaces:**
- Produces: `platforms.NormalizeHost(rawURL string) (string, error)` — returns lowercased punycode host; error sentinel `platforms.ErrBadURL` for non-https/userinfo/IP/unparseable. `platforms.MatchesSuffix(host, suffix string) bool` — exact or dot-boundary suffix match. Both consumed by Tasks 3, 4, 6.

- [x] **Step 1: Write the failing tests**

```go
package platforms

import (
	"errors"
	"testing"
)

func TestNormalizeHost(t *testing.T) {
	cases := []struct {
		name, raw, want string
		wantErr         bool
	}{
		{"plain https", "https://timepad.ru/event/123", "timepad.ru", false},
		{"subdomain", "https://org.timepad.ru/event/123", "org.timepad.ru", false},
		{"uppercase host", "https://TimePad.RU/e", "timepad.ru", false},
		{"unicode idn to punycode", "https://культура.рф/afisha", "xn--80atdujec4e.xn--p1ai", false},
		{"http rejected", "http://timepad.ru/e", "", true},
		{"userinfo rejected", "https://user@timepad.ru/e", "", true},
		{"ipv4 rejected", "https://1.2.3.4/e", "", true},
		{"ipv6 rejected", "https://[2001:db8::1]/e", "", true},
		{"garbage rejected", "not a url", "", true},
		{"empty rejected", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeHost(c.raw)
			if c.wantErr {
				if !errors.Is(err, ErrBadURL) {
					t.Fatalf("want ErrBadURL, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestMatchesSuffix(t *testing.T) {
	cases := []struct {
		name, host, suffix string
		want               bool
	}{
		{"exact", "timepad.ru", "timepad.ru", true},
		{"subdomain", "org.timepad.ru", "timepad.ru", true},
		{"city subdomain", "msk.kassir.ru", "kassir.ru", true},
		{"bypass attempt", "timepad.ru.evil.com", "timepad.ru", false},
		{"partial label", "evilTimepad.ru", "timepad.ru", false},
		{"unrelated", "evil.com", "timepad.ru", false},
		{"suffix longer than host", "ru", "timepad.ru", false},
		{"deep gov path host", "afisha.yandex.ru", "afisha.yandex.ru", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MatchesSuffix(c.host, c.suffix); got != c.want {
				t.Fatalf("MatchesSuffix(%q,%q) = %v, want %v", c.host, c.suffix, got, c.want)
			}
		})
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/platforms/ -v`
Expected: FAIL — `NormalizeHost` undefined.

- [x] **Step 3: Implement**

```go
// Package platforms is the trusted external-registration platform whitelist:
// URL normalization, domain-suffix matching, and the trusted_platforms store.
// Spec: docs/superpowers/specs/2026-08-12-external-registration-design.md.
package platforms

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
)

// ErrBadURL marks an external-registration URL that fails hard validation
// (non-https, userinfo, IP host, unparseable). Everything else — including an
// unknown domain — is NOT an error; unknown domains take the moderation path.
var ErrBadURL = errors.New("bad external registration url")

// NormalizeHost parses a raw URL and returns its lowercased punycode host.
func NormalizeHost(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("%w: %q", ErrBadURL, rawURL)
	}
	if u.User != nil {
		return "", fmt.Errorf("%w: userinfo in %q", ErrBadURL, rawURL)
	}
	host := strings.ToLower(u.Hostname())
	if net.ParseIP(host) != nil {
		return "", fmt.Errorf("%w: ip host in %q", ErrBadURL, rawURL)
	}
	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return "", fmt.Errorf("%w: idna %q", ErrBadURL, rawURL)
	}
	return ascii, nil
}

// MatchesSuffix reports whether host equals suffix or ends with "."+suffix
// (dot-boundary check defeats "timepad.ru.evil.com").
func MatchesSuffix(host, suffix string) bool {
	if host == suffix {
		return true
	}
	return strings.HasSuffix(host, "."+suffix)
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/platforms/ -v`
Expected: PASS (all cases).

- [x] **Step 5: Commit**

```bash
git add backend/internal/platforms/
git commit -m "feat(platforms): url normalization + punycode suffix matcher"
```

---

### Task 3: `internal/platforms` — model, repository, service

**Files:**
- Create: `backend/internal/platforms/model.go`
- Create: `backend/internal/platforms/repository.go`
- Create: `backend/internal/platforms/service.go`
- Test: `backend/internal/platforms/service_test.go`

**Interfaces:**
- Consumes: `NormalizeHost`, `MatchesSuffix` (Task 2).
- Produces (consumed by Tasks 4, 6, 7):

```go
type TrustedPlatform struct {
	ID           uuid.UUID
	DomainSuffix string
	DisplayName  string
	Category     string
	IsActive     bool
	CreatedAt    time.Time
}
type Repository interface {
	ListActive(ctx context.Context) ([]*TrustedPlatform, error)
	List(ctx context.Context) ([]*TrustedPlatform, error) // all, admin view
	Create(ctx context.Context, p *TrustedPlatform) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
}
type Service interface {
	// Check resolves a raw URL against active platforms.
	// trusted=false with nil err means "unknown domain" (moderation path).
	// err wraps ErrBadURL only for hard-invalid URLs.
	Check(ctx context.Context, rawURL string) (displayName string, trusted bool, err error)
	ListActive(ctx context.Context) ([]*TrustedPlatform, error)
	List(ctx context.Context) ([]*TrustedPlatform, error)
	Add(ctx context.Context, domainSuffix, displayName, category string) (*TrustedPlatform, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
}
func NewService(repo Repository) Service
func NewRepository(db *pg.DB) Repository
```

- [x] **Step 1: Write the failing service tests (fake repo, no DB)**

```go
package platforms

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
)

type fakeRepo struct{ rows []*TrustedPlatform }

func (f *fakeRepo) ListActive(context.Context) ([]*TrustedPlatform, error) {
	out := []*TrustedPlatform{}
	for _, r := range f.rows {
		if r.IsActive {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeRepo) List(context.Context) ([]*TrustedPlatform, error) { return f.rows, nil }
func (f *fakeRepo) Create(_ context.Context, p *TrustedPlatform) error {
	f.rows = append(f.rows, p)
	return nil
}
func (f *fakeRepo) SetActive(_ context.Context, id uuid.UUID, active bool) error {
	for _, r := range f.rows {
		if r.ID == id {
			r.IsActive = active
			return nil
		}
	}
	return errors.New("not found")
}

func newTestService() Service {
	return NewService(&fakeRepo{rows: []*TrustedPlatform{
		{DomainSuffix: "timepad.ru", DisplayName: "TimePad", Category: "ticketing", IsActive: true},
		{DomainSuffix: "xn--80atdujec4e.xn--p1ai", DisplayName: "Культура.РФ", Category: "gov", IsActive: true},
		{DomainSuffix: "dead.ru", DisplayName: "Dead", Category: "ticketing", IsActive: false},
	}})
}

func TestCheck(t *testing.T) {
	s := newTestService()
	ctx := context.Background()

	name, trusted, err := s.Check(ctx, "https://org.timepad.ru/event/123")
	if err != nil || !trusted || name != "TimePad" {
		t.Fatalf("subdomain match: name=%q trusted=%v err=%v", name, trusted, err)
	}

	name, trusted, err = s.Check(ctx, "https://культура.рф/afisha")
	if err != nil || !trusted || name != "Культура.РФ" {
		t.Fatalf("punycode match: name=%q trusted=%v err=%v", name, trusted, err)
	}

	_, trusted, err = s.Check(ctx, "https://unknown-site.ru/e")
	if err != nil || trusted {
		t.Fatalf("unknown domain must be trusted=false, err=nil; got trusted=%v err=%v", trusted, err)
	}

	_, trusted, err = s.Check(ctx, "https://x.dead.ru/e")
	if err != nil || trusted {
		t.Fatalf("inactive row must not match; got trusted=%v err=%v", trusted, err)
	}

	if _, _, err = s.Check(ctx, "http://timepad.ru/e"); !errors.Is(err, ErrBadURL) {
		t.Fatalf("http must be ErrBadURL, got %v", err)
	}
}

func TestAddValidation(t *testing.T) {
	s := newTestService()
	if _, err := s.Add(context.Background(), "", "X", "ticketing"); err == nil {
		t.Fatal("empty suffix must fail")
	}
	if _, err := s.Add(context.Background(), "new.ru", "New", "bogus"); err == nil {
		t.Fatal("bogus category must fail")
	}
	p, err := s.Add(context.Background(), "Новый.РФ", "Новый", "afisha")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if p.DomainSuffix != "xn--b1aoke0e.xn--p1ai" {
		t.Fatalf("suffix must be stored in punycode, got %q", p.DomainSuffix)
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/platforms/ -v`
Expected: FAIL — `TrustedPlatform`, `NewService` undefined.

- [x] **Step 3: Implement model, repository, service**

`model.go`:

```go
package platforms

import (
	"time"

	"github.com/gofrs/uuid"
)

// TrustedPlatform is one whitelist row. DomainSuffix is punycode, no scheme.
type TrustedPlatform struct {
	tableName struct{} `pg:"trusted_platforms,discard_unknown_columns"` //nolint:unused

	ID           uuid.UUID `pg:"id,pk,type:uuid" json:"id"`
	DomainSuffix string    `pg:"domain_suffix,notnull" json:"domain_suffix"`
	DisplayName  string    `pg:"display_name,notnull" json:"display_name"`
	Category     string    `pg:"category,notnull" json:"category"`
	IsActive     bool      `pg:"is_active,use_zero" json:"is_active"`
	CreatedAt    time.Time `pg:"created_at,notnull,default:now()" json:"created_at"`
}
```

`repository.go` (mirror `backend/internal/moderation/repository.go` construction style):

```go
package platforms

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type Repository interface {
	ListActive(ctx context.Context) ([]*TrustedPlatform, error)
	List(ctx context.Context) ([]*TrustedPlatform, error)
	Create(ctx context.Context, p *TrustedPlatform) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
}

type pgRepository struct{ db *pg.DB }

func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) ListActive(ctx context.Context) ([]*TrustedPlatform, error) {
	var out []*TrustedPlatform
	err := r.db.ModelContext(ctx, &out).Where("is_active").Order("domain_suffix ASC").Select()
	if err != nil {
		return nil, fmt.Errorf("list active platforms: %w", err)
	}
	return out, nil
}

func (r *pgRepository) List(ctx context.Context) ([]*TrustedPlatform, error) {
	var out []*TrustedPlatform
	if err := r.db.ModelContext(ctx, &out).Order("domain_suffix ASC").Select(); err != nil {
		return nil, fmt.Errorf("list platforms: %w", err)
	}
	return out, nil
}

func (r *pgRepository) Create(ctx context.Context, p *TrustedPlatform) error {
	if _, err := r.db.ModelContext(ctx, p).Insert(); err != nil {
		return fmt.Errorf("insert platform: %w", err)
	}
	return nil
}

func (r *pgRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE trusted_platforms SET is_active = ? WHERE id = ?`, active, id)
	if err != nil {
		return fmt.Errorf("set platform active: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("platform %s: not found", id)
	}
	return nil
}
```

Note: `Create` relies on the DB `DEFAULT gen_random_uuid()`; go-pg skips zero-uuid `id` because the field has no `use_zero`. If the insert complains about a nil uuid, generate `uuid.NewV4()` in `Create` before insert instead.

`service.go`:

```go
package platforms

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
)

type Service interface {
	Check(ctx context.Context, rawURL string) (displayName string, trusted bool, err error)
	ListActive(ctx context.Context) ([]*TrustedPlatform, error)
	List(ctx context.Context) ([]*TrustedPlatform, error)
	Add(ctx context.Context, domainSuffix, displayName, category string) (*TrustedPlatform, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
}

var validCategories = map[string]bool{"ticketing": true, "afisha": true, "gov": true, "social": true}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Check(ctx context.Context, rawURL string) (string, bool, error) {
	host, err := NormalizeHost(rawURL)
	if err != nil {
		return "", false, err
	}
	rows, err := s.repo.ListActive(ctx)
	if err != nil {
		return "", false, fmt.Errorf("check platform: %w", err)
	}
	for _, row := range rows {
		if MatchesSuffix(host, row.DomainSuffix) {
			return row.DisplayName, true, nil
		}
	}
	return "", false, nil
}

func (s *service) ListActive(ctx context.Context) ([]*TrustedPlatform, error) { return s.repo.ListActive(ctx) }
func (s *service) List(ctx context.Context) ([]*TrustedPlatform, error)       { return s.repo.List(ctx) }

func (s *service) Add(ctx context.Context, domainSuffix, displayName, category string) (*TrustedPlatform, error) {
	if domainSuffix == "" || displayName == "" {
		return nil, fmt.Errorf("domain_suffix and display_name are required")
	}
	if !validCategories[category] {
		return nil, fmt.Errorf("invalid category %q", category)
	}
	// Normalize the suffix the same way hosts are normalized (punycode,
	// lowercase) so matching stays consistent.
	ascii, err := NormalizeHost("https://" + domainSuffix)
	if err != nil {
		return nil, fmt.Errorf("invalid domain_suffix %q", domainSuffix)
	}
	p := &TrustedPlatform{DomainSuffix: ascii, DisplayName: displayName, Category: category, IsActive: true}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) Deactivate(ctx context.Context, id uuid.UUID) error {
	return s.repo.SetActive(ctx, id, false)
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/platforms/ -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add backend/internal/platforms/
git commit -m "feat(platforms): trusted_platforms repo + service with Check/Add/Deactivate"
```

---

### Task 4: Events service — verification on create/update, new model fields

**Files:**
- Modify: `backend/internal/models/event.go` (struct fields, ~line 55)
- Modify: `backend/internal/events/service.go` (UpdateParams ~line 110, `Create` ~line 212, `Update` ~line 293, service struct ~line 168)
- Test: `backend/internal/events/service_test.go` (append; existing fakes live here or in the test file — reuse the file's existing fake repo)

**Interfaces:**
- Consumes: `platforms.ErrBadURL` semantics via the checker (Task 3's `Service.Check` signature).
- Produces (consumed by Tasks 6, and wiring in `application.go`):

```go
// in package events:
type PlatformChecker interface {
	Check(ctx context.Context, rawURL string) (displayName string, trusted bool, err error)
}
func (s *service) SetPlatformChecker(c PlatformChecker) // wired like SetDailyLimit
// UpdateParams gains: CapacityLimited *bool
// models.Event gains:
//   CapacityLimited      bool   `pg:"capacity_limited,use_zero"`
//   ExternalURLVerified  bool   `pg:"external_url_verified,use_zero"`
//   ExternalPlatformName string `pg:"-"` // transient, read-path (Task 6)
```

Policy (single helper, called from both Create and Update **after** field
apply/Validate and **before** the repo write):

```go
// applyExternalURLPolicy enforces the whitelist on the way to publication.
// urlChanged=true forces a re-check (create counts as changed). Never blocks
// unknown domains — it downgrades the status to pending_review instead.
func (s *service) applyExternalURLPolicy(ctx context.Context, event *models.Event, urlChanged bool) error {
	if s.platformChecker == nil { // feature not wired (tests) — no-op
		return nil
	}
	if event.SignupMode != "external" || event.Status != models.EventPublished {
		return nil
	}
	if event.ExternalURLVerified && !urlChanged {
		return nil // snapshot semantics: whitelist edits don't re-judge
	}
	_, trusted, err := s.platformChecker.Check(ctx, event.ExternalRegistrationURL)
	if err != nil {
		return fmt.Errorf("%w: некорректная ссылка для внешней регистрации (нужен https)", ErrInvalidInput)
	}
	if trusted {
		event.ExternalURLVerified = true
		return nil
	}
	event.ExternalURLVerified = false
	event.Status = models.EventPendingReview
	return nil
}
```

- [x] **Step 1: Write the failing service tests**

Append to `backend/internal/events/service_test.go`, following the file's existing fake-repo pattern (read the top of the file first; a fake `Repository` already exists there — reuse it). Add a fake checker:

```go
type fakeChecker struct {
	name    string
	trusted bool
	err     error
	calls   int
}

func (f *fakeChecker) Check(context.Context, string) (string, bool, error) {
	f.calls++
	return f.name, f.trusted, f.err
}
```

Test cases (each builds a service via the file's existing constructor helper, then `svc.(*service).SetPlatformChecker(checker)` — note `SetDailyLimit` is already called through the same type-assertion pattern in `application.go`; mirror how tests call `SetDailyLimit` if they do):

1. `TestCreateExternalTrustedPublishes` — event `SignupMode: "external"`, `Status: models.EventPublished`, `ExternalRegistrationURL: "https://x.timepad.ru/e/1"`, checker `{trusted: true, name: "TimePad"}` → after `Create`, `event.Status == models.EventPublished` and `event.ExternalURLVerified == true`.
2. `TestCreateExternalUnknownGoesPending` — checker `{trusted: false}` → after `Create`, `event.Status == models.EventPendingReview`, `event.ExternalURLVerified == false`, **no error**.
3. `TestCreateExternalBadURLRejected` — checker `{err: platforms.ErrBadURL}` → `Create` returns error wrapping `ErrInvalidInput`.
4. `TestCreateExternalDraftSkipsCheck` — `Status: models.EventDraft` → checker `calls == 0`, status stays draft.
5. `TestUpdateExternalURLChangeRechecks` — existing published external event with `ExternalURLVerified: true` in the fake repo; patch `ExternalRegistrationURL` to a new value with checker `{trusted: false}` → reloaded event has status `pending_review`.
6. `TestUpdateUnrelatedEditKeepsVerified` — same stored event; patch only `Title` with checker `{trusted: false}` → checker `calls == 0`, status stays `published` (snapshot semantics).
7. `TestUpdateCapacityLimited` — patch `CapacityLimited: ptr(true)` → persisted event has `CapacityLimited == true`.
8. `TestCreateNilCheckerNoop` — no `SetPlatformChecker` call → external published event stays published (feature off, existing behavior preserved).

- [x] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/events/ -run 'External|CapacityLimited' -v`
Expected: FAIL — `SetPlatformChecker` undefined, `CapacityLimited` unknown field.

- [x] **Step 3: Implement**

1. `models/event.go` — add to the signup block (after `ExternalRegistrationURL`):

```go
	CapacityLimited     bool `pg:"capacity_limited,use_zero"`      // external: "мест ограничено", число не хранится
	ExternalURLVerified bool `pg:"external_url_verified,use_zero"` // snapshot: URL matched whitelist when last checked

	// ExternalPlatformName is transient (not a column): the whitelist display
	// name matched at read time. Populated by the events repository.
	ExternalPlatformName string `pg:"-"`
```

2. `events/service.go`:
   - Add `PlatformChecker` interface + `platformChecker PlatformChecker` field on `service` + `SetPlatformChecker` method (next to `SetDailyLimit`, same style).
   - Add `CapacityLimited *bool` to `UpdateParams`.
   - `Create`: after the venue validation and before quota checks, call `if err := s.applyExternalURLPolicy(ctx, event, true); err != nil { return err }`.
   - `Update`: capture `originalURL := event.ExternalRegistrationURL` right after the ownership check; in the field-apply block add `if p.CapacityLimited != nil { event.CapacityLimited = *p.CapacityLimited }`; after `event.Validate()` (line ~403) add:

```go
	urlChanged := p.ExternalRegistrationURL != nil && *p.ExternalRegistrationURL != originalURL
	if err := s.applyExternalURLPolicy(ctx, event, urlChanged); err != nil {
		return nil, err
	}
```

   - Add the `applyExternalURLPolicy` helper (code above, in Interfaces).

- [x] **Step 4: Run the full events test suite**

Run: `cd backend && go test ./internal/events/ ./internal/models/ -v`
Expected: PASS, including all pre-existing tests (nil checker keeps old behavior).

- [x] **Step 5: Wire in application.go**

In `backend/internal/application.go`, find where the events service is built (grep `SetDailyLimit` — the platforms wiring goes next to it) and add:

```go
	platformsRepo := platforms.NewRepository(db)
	platformsSvc := platforms.NewService(platformsRepo)
	eventsService.(interface{ SetPlatformChecker(events.PlatformChecker) }).SetPlatformChecker(platformsSvc)
```

Adjust to the file's actual style: if `SetDailyLimit` is called via a concrete type or a declared interface, mirror that exactly. Keep `platformsSvc` in scope — Tasks 6–7 pass it to HTTP handlers.

Run: `cd backend && go build ./...`
Expected: builds clean.

- [x] **Step 6: Commit**

```bash
git add backend/internal/models/event.go backend/internal/events/ backend/internal/application.go
git commit -m "feat(events): whitelist check on publish — unknown domain routes to pending_review"
```

---

### Task 5: Moderation `Approve` + admin endpoint

**Files:**
- Modify: `backend/internal/moderation/repository.go` (add `Approve`, ~line 50)
- Modify: `backend/internal/moderation/service.go` (interface + method)
- Modify: `backend/internal/http/admin/handler.go` (route + handler)
- Test: `backend/internal/moderation/service_test.go` (or wherever existing moderation tests live — check `ls backend/internal/moderation/`), `backend/internal/http/admin/handler_test.go`

**Interfaces:**
- Produces: `moderation.Service.Approve(ctx, eventID, actorID uuid.UUID) error` — transition `pending_review → published`, action `"event.approve"`; sets `external_url_verified = true` and `published_at = COALESCE(published_at, now())`. HTTP: `POST /api/v1/admin/moderation/events/{id}/approve` → 204. Consumed by Task 11 (admin UI).

- [x] **Step 1: Write the failing repository/service test**

Follow the existing moderation test pattern (read the existing test file first). The transition helper takes fixed from/to, but Approve needs extra columns — so Approve gets its own transaction, structured like `transition`:

```go
func (r *pgRepository) Approve(ctx context.Context, eventID, actorID uuid.UUID) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE events
			    SET status = 'published',
			        external_url_verified = true,
			        published_at = COALESCE(published_at, now()),
			        updated_at = now()
			  WHERE id = ? AND status = 'pending_review'`, eventID)
		if err != nil {
			return fmt.Errorf("approve event: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrInvalidTransition
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO event_status_history (event_id, from_status, to_status, actor_user_id, reason)
			 VALUES (?, 'pending_review', 'published', ?, NULL)`, eventID, actorID); err != nil {
			return fmt.Errorf("insert status history: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'event.approve', 'event', ?, '{}'::jsonb)`, actorID, eventID); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
}
```

If existing moderation tests use a fake repo (service-level), test the service delegation + add an admin handler test; if they run against a real DB, mirror that. Admin handler test (mirror an existing `handler_test.go` case, e.g. reinstate): POST to `/api/v1/admin/moderation/events/{id}/approve` with a staff token fake → expect 204 and the fake moderation service records the call; non-staff → 403.

- [x] **Step 2: Run to verify failure**

Run: `cd backend && go test ./internal/moderation/ ./internal/http/admin/ -run Approve -v`
Expected: FAIL — `Approve` undefined.

- [x] **Step 3: Implement**

- `moderation/repository.go`: add `Approve` to the `Repository` interface + the implementation above.
- `moderation/service.go`: add `Approve(ctx context.Context, eventID, actorID uuid.UUID) error` to `Service` interface; implementation delegates: `func (s *service) Approve(ctx context.Context, eventID, actorID uuid.UUID) error { return s.repo.Approve(ctx, eventID, actorID) }`.
- `http/admin/handler.go`: register `h.mux.HandleFunc("POST /api/v1/admin/moderation/events/{id}/approve", h.staff(h.approve))`; implement `approve` mirroring the existing `reinstate` handler (parse `{id}`, call `h.deps.Moderation.Approve(r.Context(), id, actor.ID)`, map `ErrInvalidTransition` → 409, success → 204). Read `reinstate` first and copy its exact error-mapping style.

- [x] **Step 4: Run tests**

Run: `cd backend && go test ./internal/moderation/ ./internal/http/admin/ -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add backend/internal/moderation/ backend/internal/http/admin/
git commit -m "feat(moderation): approve pending_review events (whitelist moderation path)"
```

---

### Task 6: API surface — swagger fields, formatter, read-path enrichment, public /trusted-platforms

**Files:**
- Modify: `backend/api/swagger.yaml` (Event ~line 930, EventInput ~line 1033, EventPatch ~line 1100)
- Regenerate: `backend/internal/http/models/`, `backend/internal/http/server/` (via `make generate-api`)
- Modify: `backend/internal/http/formatter/event.go`
- Modify: `backend/internal/events/repository.go` (read-path enrichment)
- Create: `backend/internal/http/platformspublic/handler.go`
- Test: `backend/internal/http/platformspublic/handler_test.go`, extend `backend/internal/http/formatter` tests if present (check `ls backend/internal/http/formatter/`)
- Modify: `backend/internal/http/module.go` (mount public handler, ~line 458)

**Interfaces:**
- Consumes: `models.Event.{CapacityLimited, ExternalURLVerified, ExternalPlatformName}` (Task 4), `platforms.Service.ListActive` + `platforms.MatchesSuffix` + `platforms.NormalizeHost` (Tasks 2–3).
- Produces (consumed by frontend Tasks 8–10): Event JSON gains `external_platform_name` (string, omitted when empty), `capacity_limited` (bool), `moderation_required` (bool, true ⇔ status is `pending_review`); EventInput/EventPatch accept `capacity_limited`; `GET /api/v1/trusted-platforms` → `200 [{"domain_suffix":"timepad.ru","display_name":"TimePad","category":"ticketing"}, …]` (active only, no auth).

- [x] **Step 1: swagger.yaml — add fields**

In the `Event` definition (find the existing `signup_mode:` block ~line 930), add alongside:

```yaml
      external_platform_name:
        type: string
        description: Whitelist display name matched from external_registration_url (read-only)
      capacity_limited:
        type: boolean
        description: Seats are limited but the count is managed on the external platform
      moderation_required:
        type: boolean
        description: True when the event is pending_review (read-only)
```

In `EventInput` (~line 1033) and `EventPatch` (~line 1100), add:

```yaml
      capacity_limited:
        type: boolean
        x-nullable: true
```

(`x-nullable` in the patch gives `*bool` so "absent" ≠ "false"; in EventInput it is also `*bool` — treat nil as false.)

- [x] **Step 2: Regenerate + validate**

Run: `cd backend && make swagger-validate && make generate-api && go build ./...`
Expected: clean build. Inspect `internal/http/models/event.go` for the generated field names (`ExternalPlatformName`, `CapacityLimited`, `ModerationRequired`) — use whatever casing the generator emits.

- [x] **Step 3: Formatter mappings**

`formatter/event.go`:
- `EventToAPI` (after the `out.SignupMode = …` block ~line 121):

```go
	out.ExternalPlatformName = event.ExternalPlatformName
	out.CapacityLimited = event.CapacityLimited
	out.ModerationRequired = event.Status == domainModels.EventPendingReview
```

(If the generated fields are pointers, assign via temporaries following the `Verified` pattern above in the same file.)
- `EventFromAPIInput`: `if in.CapacityLimited != nil { event.CapacityLimited = *in.CapacityLimited }`.
- `EventPatchToUpdateParams`: `if in.CapacityLimited != nil { v := *in.CapacityLimited; p.CapacityLimited = &v }`.

- [x] **Step 4: Read-path enrichment in the events repository**

In `backend/internal/events/repository.go`, find the shared enrichment path (grep `loadOrganizers` — the function that calls it after selects). Add a `loadPlatformNames` step in the same pipeline:

```go
// loadPlatformNames resolves ExternalPlatformName for external-mode events by
// matching their registration URL against active trusted_platforms. One query
// per batch; silent skip on bad URLs (they only reach here pre-moderation).
func (r *pgRepository) loadPlatformNames(events []*models.Event) error {
	var external []*models.Event
	for _, e := range events {
		if e.SignupMode == "external" && e.ExternalRegistrationURL != "" {
			external = append(external, e)
		}
	}
	if len(external) == 0 {
		return nil
	}
	var rows []*platforms.TrustedPlatform
	if err := r.db.Model(&rows).Where("is_active").Select(); err != nil {
		return fmt.Errorf("load trusted platforms: %w", err)
	}
	for _, e := range external {
		host, err := platforms.NormalizeHost(e.ExternalRegistrationURL)
		if err != nil {
			continue
		}
		for _, row := range rows {
			if platforms.MatchesSuffix(host, row.DomainSuffix) {
				e.ExternalPlatformName = row.DisplayName
				break
			}
		}
	}
	return nil
}
```

Match the repository's actual receiver/type names (read the file first — the receiver may not be named `pgRepository`). Call it wherever `loadOrganizers` is called, with the same error handling style. Import `github.com/Pashteto/lia/internal/platforms` (no cycle: platforms imports nothing from events).

- [x] **Step 5: Public trusted-platforms handler + mount**

`backend/internal/http/platformspublic/handler.go` (mirror `backend/internal/http/geocode/handler.go` structure, but **no auth** — this list is public):

```go
// Package platformspublic serves the public read-only whitelist:
// GET /api/v1/trusted-platforms. Mounted ahead of the swagger mux.
package platformspublic

import (
	"encoding/json"
	"net/http"

	"github.com/Pashteto/lia/internal/platforms"
	"github.com/Pashteto/lia/pkg/logger"
)

type Deps struct{ Platforms platforms.Service }

type handler struct{ deps Deps }

func NewHandler(deps Deps) http.Handler { return &handler{deps: deps} }

type row struct {
	DomainSuffix string `json:"domain_suffix"`
	DisplayName  string `json:"display_name"`
	Category     string `json:"category"`
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rows, err := h.deps.Platforms.ListActive(r.Context())
	if err != nil {
		logger.Log().Errorf("list trusted platforms: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	out := make([]row, 0, len(rows))
	for _, p := range rows {
		out = append(out, row{DomainSuffix: p.DomainSuffix, DisplayName: p.DisplayName, Category: p.Category})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
```

Handler test: fake `platforms.Service` returning two rows → GET returns 200 with both; POST → 405. (Write the fake against the `platforms.Service` interface.)

Mount in `module.go` router (next to the geocode branch ~line 458):

```go
		if p == "/api/v1/trusted-platforms" {
			platformsH.ServeHTTP(w, r)
			return
		}
```

building `platformsH := platformspublic.NewHandler(platformspublic.Deps{Platforms: m.platforms})` alongside `geocodeH` — add an `m.platforms platforms.Service` field + `SetPlatforms` setter to `Module` following the existing `SetAdminUsers` pattern (~line 149), and call it from `application.go` with the `platformsSvc` built in Task 4 Step 5.

- [x] **Step 6: Run tests + build**

Run: `cd backend && go build ./... && go test ./internal/http/... ./internal/events/ -v`
Expected: PASS.

- [x] **Step 7: Commit**

```bash
git add backend/api/swagger.yaml backend/internal/http/ backend/internal/events/repository.go
git commit -m "feat(api): platform name + capacity_limited + moderation_required; public trusted-platforms list"
```

---

### Task 7: Admin CRUD for trusted platforms

**Files:**
- Modify: `backend/internal/http/admin/handler.go`
- Test: `backend/internal/http/admin/handler_test.go`

**Interfaces:**
- Consumes: `platforms.Service` (Task 3) — add `Platforms platforms.Service` to `admin.Deps`; wire from `application.go` (same `platformsSvc`).
- Produces (consumed by Task 11): `GET /api/v1/admin/trusted-platforms` → 200 full rows (including inactive, JSON tags from the model); `POST /api/v1/admin/trusted-platforms` body `{"domain_suffix":"…","display_name":"…","category":"ticketing"}` → 201 created row; `DELETE /api/v1/admin/trusted-platforms/{id}` → 204 (deactivates, does not hard-delete).

- [x] **Step 1: Write failing handler tests** — mirror the existing staff-gated handler tests: staff GET lists rows from a fake service; POST with invalid category → 422; POST valid → 201; DELETE unknown id → 404/409 per the fake's error; non-staff → 403.

- [x] **Step 2: Run to verify failure**

Run: `cd backend && go test ./internal/http/admin/ -run TrustedPlatforms -v`
Expected: FAIL.

- [x] **Step 3: Implement** — add to `NewHandler`:

```go
	h.mux.HandleFunc("GET /api/v1/admin/trusted-platforms", h.staff(h.listPlatforms))
	h.mux.HandleFunc("POST /api/v1/admin/trusted-platforms", h.staff(h.addPlatform))
	h.mux.HandleFunc("DELETE /api/v1/admin/trusted-platforms/{id}", h.staff(h.removePlatform))
```

Handlers follow the file's existing json/error style: `listPlatforms` → `deps.Platforms.List`; `addPlatform` decodes the body, calls `Add`, maps validation errors → 422 via the existing error helper; `removePlatform` parses `{id}` uuid, calls `Deactivate`, not-found → 404, success → 204. Guard `deps.Platforms == nil` → 503 (mirroring how other optional deps degrade).

- [x] **Step 4: Run tests** — `cd backend && go test ./internal/http/admin/ -v` → PASS.

- [x] **Step 5: Commit**

```bash
git add backend/internal/http/admin/ backend/internal/application.go
git commit -m "feat(admin): trusted-platforms CRUD endpoints"
```

---

### Task 8: Frontend — types, api client, client-side matcher

**Files:**
- Modify: `frontend/lib/types.ts` (LiaEvent ~line 59)
- Modify: `frontend/lib/api.ts` (`apiEventToLia` ~line 22, `CreateEventInput` ~line 307, new fetchers ~end)
- Create: `frontend/lib/platforms.ts`
- Test: `frontend/lib/__tests__/platforms.test.ts`

**Interfaces:**
- Consumes: API fields from Task 6 (`external_platform_name`, `capacity_limited`, `moderation_required`; public + admin endpoints; approve endpoint from Task 5).
- Produces (consumed by Tasks 9–11):

```ts
// types.ts — LiaEvent gains:
externalPlatformName?: string;
capacityLimited?: boolean;
moderationRequired?: boolean;
// platforms.ts:
export interface TrustedPlatform { domainSuffix: string; displayName: string; category: string }
export function matchPlatform(rawUrl: string, platforms: TrustedPlatform[]): TrustedPlatform | null;
export function hostOf(rawUrl: string): string | null; // lowercased URL host, null when unparseable
// api.ts:
export async function fetchTrustedPlatforms(): Promise<TrustedPlatform[]>;          // public
export interface AdminTrustedPlatform extends TrustedPlatform { id: string; isActive: boolean }
export async function adminListTrustedPlatforms(): Promise<AdminTrustedPlatform[]>;
export async function adminAddTrustedPlatform(input: {domainSuffix: string; displayName: string; category: string}): Promise<void>;
export async function adminRemoveTrustedPlatform(id: string): Promise<void>;
export async function approveEvent(id: string): Promise<void>; // POST …/moderation/events/{id}/approve
// CreateEventInput gains capacityLimited?: boolean → body field capacity_limited
```

- [x] **Step 1: Write failing matcher tests**

```ts
import { describe, expect, it } from "vitest";
import { hostOf, matchPlatform, type TrustedPlatform } from "@/lib/platforms";

const LIST: TrustedPlatform[] = [
  { domainSuffix: "timepad.ru", displayName: "TimePad", category: "ticketing" },
  { domainSuffix: "xn--80atdujec4e.xn--p1ai", displayName: "Культура.РФ", category: "gov" },
];

describe("hostOf", () => {
  it("extracts lowercased host", () => {
    expect(hostOf("https://Org.TimePad.ru/event/1")).toBe("org.timepad.ru");
  });
  it("returns null for garbage", () => {
    expect(hostOf("not a url")).toBeNull();
  });
});

describe("matchPlatform", () => {
  it("matches subdomains", () => {
    expect(matchPlatform("https://org.timepad.ru/e/1", LIST)?.displayName).toBe("TimePad");
  });
  it("rejects suffix-bypass hosts", () => {
    expect(matchPlatform("https://timepad.ru.evil.com/e", LIST)).toBeNull();
  });
  it("matches punycode idn (browser URL punycodes the host)", () => {
    expect(matchPlatform("https://культура.рф/afisha", LIST)?.displayName).toBe("Культура.РФ");
  });
  it("returns null for unknown domains", () => {
    expect(matchPlatform("https://unknown.ru/e", LIST)).toBeNull();
  });
});
```

- [x] **Step 2: Run to verify failure** — `cd frontend && npx vitest run lib/__tests__/platforms.test.ts` → FAIL (module missing).

- [x] **Step 3: Implement `lib/platforms.ts`**

```ts
/** Client-side mirror of backend/internal/platforms matching — used only for
 * the live form hint; the backend check is authoritative. */
export interface TrustedPlatform {
  domainSuffix: string;
  displayName: string;
  category: string;
}

export function hostOf(rawUrl: string): string | null {
  try {
    // `new URL().hostname` is already punycode + lowercase in browsers/node.
    return new URL(rawUrl).hostname.toLowerCase();
  } catch {
    return null;
  }
}

export function matchPlatform(
  rawUrl: string,
  platforms: TrustedPlatform[],
): TrustedPlatform | null {
  const host = hostOf(rawUrl);
  if (!host) return null;
  return (
    platforms.find(
      (p) => host === p.domainSuffix || host.endsWith(`.${p.domainSuffix}`),
    ) ?? null
  );
}
```

- [x] **Step 4: types + api mappings**

- `types.ts` LiaEvent: add the three optional fields (Interfaces block above).
- `api.ts` `apiEventToLia`: map `externalPlatformName: e.external_platform_name`, `capacityLimited: e.capacity_limited`, `moderationRequired: e.moderation_required` (add the snake_case fields to the `ApiEvent` type in the same file, matching how `price_min` is declared).
- `CreateEventInput`: add `capacityLimited?: boolean`; in `createEvent`/`patchEvent` body construction include `capacity_limited: input.capacityLimited` (follow how other optional fields are serialized in those functions — read them first).
- New fetchers per the Interfaces block: `fetchTrustedPlatforms` hits `${API_V1}/trusted-platforms` without auth headers and maps snake_case→camelCase; the three admin functions follow the `listModerationEvents`/`takedownEvent` style (auth headers, error on !ok); `approveEvent` mirrors `takedownEvent` with no body.

- [x] **Step 5: Run tests + typecheck** — `cd frontend && npx vitest run && npx tsc --noEmit` → PASS.

- [x] **Step 6: Commit**

```bash
git add frontend/lib/
git commit -m "feat(frontend): trusted-platform types, client matcher, api fetchers"
```

---

### Task 9: Organizer form — live hint, hidden-capacity checkbox, moderation notice

**Files:**
- Modify: `frontend/components/CreateEventForm.tsx` (schema ~line 49, payload builder ~line 192, defaults ~line 250, mutations ~line 285, signup section ~line 724)
- Test: `frontend/components/__tests__/create-event-schema.test.ts` (extend)

**Interfaces:**
- Consumes: `fetchTrustedPlatforms`, `matchPlatform` (Task 8); `moderationRequired` on the created/patched event.
- Produces: form field `capacityLimited: boolean` serialized as `capacity_limited`.

- [x] **Step 1: Extend the zod schema test** — in `create-event-schema.test.ts` add cases: external mode with `capacityLimited: true` parses; open mode ignores `capacityLimited` (payload builder omits it when mode ≠ external). Follow the file's existing test style for building form values and calling the exported schema/payload helpers.

- [x] **Step 2: Run to verify failure** — `cd frontend && npx vitest run components/__tests__/create-event-schema.test.ts` → FAIL.

- [x] **Step 3: Implement schema + payload**

- Schema (~line 55): add `capacityLimited: z.boolean().optional()`.
- Payload builder (~line 192): add `capacity_limited: v.signupMode === "external" ? (v.capacityLimited ?? false) : undefined`.
- Defaults (~line 250): `capacityLimited: initial?.capacityLimited ?? false`.

- [x] **Step 4: Implement UI**

In the signup section (~line 742):
- Load platforms once: `const [platforms, setPlatforms] = useState<TrustedPlatform[]>([]);` + `useEffect(() => { fetchTrustedPlatforms().then(setPlatforms).catch(() => {}); }, []);` — a failed load degrades to "no hint", never blocks the form.
- Under the «Ссылка для регистрации» input, add the live hint (watch the URL with `useWatch({ control, name: "externalRegistrationUrl" })`):

```tsx
{signupMode === "external" && extUrl?.trim() && (
  matched ? (
    <span className="text-[11px] text-emerald-700">Платформа: {matched.displayName}</span>
  ) : hostOf(extUrl.trim()) ? (
    <span className="text-[11px] text-amber-700">
      Неизвестная платформа — событие уйдёт на проверку модератору
    </span>
  ) : null
)}
```

where `const matched = extUrl ? matchPlatform(extUrl.trim(), platforms) : null`. Use the project's existing hint-color tokens if `text-emerald-700`/`text-amber-700` aren't already used — grep for similar success/warning inline hints and reuse their classes.
- Inside the same `hidden={signupMode !== "external"}` block add the checkbox (match the form's existing checkbox/label idiom — grep for `type="checkbox"` in the file/neighbors first; if none exists, a plain `<label className="flex items-center gap-[7px] text-[12px]"><input type="checkbox" {...register("capacityLimited")} /> Количество мест ограничено</label>` is fine):
- Paid-open warning (spec §4): in the price section (near the `priceMin` input, ~line 680), when the watched values give `priceMin > 0 && signupMode !== "external"`, render `<span className="text-[11px] text-text-dim">Посетители увидят «Оплата на месте»</span>` (watch `priceMin` via the same `useWatch` pattern as `signupMode` at line ~263).
- Moderation notice: in the create/patch `onSuccess` handlers (~lines 286–303), before routing: `if (event.moderationRequired) { router.push(`/events/${event.id}?moderation=1`); return; }` — and in `frontend/app/events/[id]/page.tsx`'s client view (or `EventDetailView`), when the query flag is present and the viewer owns the pending event, render a banner: «Событие отправлено на проверку модератору и опубликуется после одобрения». Follow the existing owner-draft banner pattern — grep `Черновик` in `EventDetailView.tsx`/`app/events/[id]` to find it and mirror.

- [x] **Step 5: Run tests + typecheck** — `cd frontend && npx vitest run && npx tsc --noEmit` → PASS.

- [x] **Step 6: Commit**

```bash
git add frontend/components/CreateEventForm.tsx frontend/components/__tests__/ frontend/app/events/
git commit -m "feat(form): platform hint, hidden-capacity checkbox, moderation notice"
```

---

### Task 10: Visitor UX — CTA texts, «Оплата на месте», «Места», layout fix

**Files:**
- Modify: `frontend/components/SignupCTA.tsx` (external branch, lines 272–291)
- Modify: `frontend/components/EventDetailView.tsx` (price rail ~line 106, mobile footer ~line 214, grid ~lines 90 and 165)
- Modify: `frontend/lib/format.ts` (`attendanceShort` ~line 119)
- Test: `frontend/lib/__tests__/format.test.ts` (extend)

**Interfaces:**
- Consumes: `event.externalPlatformName`, `event.capacityLimited` (Task 8); `priceLabel`/`priceType` already on LiaEvent.

- [x] **Step 1: Extend format tests** — `attendanceShort` with `capacityLimited: true` (and external mode) returns `"Ограничено · наличие на сайте регистрации"`; without the flag behavior is unchanged (run existing cases).

- [x] **Step 2: Run to verify failure** — `cd frontend && npx vitest run lib/__tests__/format.test.ts` → FAIL.

- [x] **Step 3: Implement `attendanceShort`** — read the function first; add the `capacityLimited` branch **before** the existing unlimited/counted branches (signature takes the event or its fields — extend per its actual shape).

- [x] **Step 4: SignupCTA external branch** — replace the static label (line 283) with:

```tsx
const isPaid = event.priceType !== "free";
const name = event.externalPlatformName;
const host = event.externalRegistrationUrl ? hostOf(event.externalRegistrationUrl) : null;
// button text:
{name ? (isPaid ? `Билеты на ${name}` : `Записаться через ${name}`) : "Записаться на сайте организатора"}
// sub-caption (replaces «Запись ведёт организатор»):
{name ? host : host ? `Переход на ${host}` : "Запись ведёт организатор"}
```

(import `hostOf` from `@/lib/platforms`; keep the existing classNames untouched).

- [x] **Step 5: «Оплата на месте»** — in `EventDetailView.tsx`: desktop price rail (after the price `<span>`, ~line 110) and mobile footer (after the price `<span>`, ~line 216), when `event.priceType !== "free" && event.signupMode !== "external"`, render `<span className="text-[10px] uppercase tracking-[0.07em] text-text-dim">Оплата на месте</span>` (reuse the `cap` utility class if it fits better — compare with the «Цена» caption styling).

- [x] **Step 6: Layout fix** — change `grid-cols-[1fr_200px]` → `grid-cols-[1fr_248px]` at **both** occurrences (~lines 90 and 165 — header CTA rail and venue address rail; both must move together or the vertical rules misalign).

- [x] **Step 7: Verify visually** — run the frontend dev server against local backend (or use existing screenshots flow), open an event page: CTA column no longer clips «Вы записаны» / «В календарь (.ics)» / «В Google-календарь» against the `border-l` rule; check an external-mode event shows the platform-branded button.

- [x] **Step 8: Run all frontend tests + typecheck** — `cd frontend && npx vitest run && npx tsc --noEmit` → PASS.

- [x] **Step 9: Commit**

```bash
git add frontend/components/SignupCTA.tsx frontend/components/EventDetailView.tsx frontend/lib/
git commit -m "feat(event-page): platform CTA texts, оплата на месте, hidden capacity, wider CTA rail"
```

---

### Task 11: Admin UI — pending queue with approve, trusted-platforms management

**Files:**
- Modify: `frontend/components/AdminModeration.tsx` (filter ~line 65, queue load ~line 82, row actions ~line 192)
- Modify: `frontend/lib/api.ts` (`listModerationEvents` status union ~line 699)
- Create: `frontend/components/AdminTrustedPlatforms.tsx`
- Modify: `frontend/app/admin/settings/page.tsx` (mount the new section — read it first; if settings is the wrong home, mount on `frontend/app/admin/moderation/events/page.tsx` below `AdminModeration`)

**Interfaces:**
- Consumes: `approveEvent`, `adminListTrustedPlatforms`, `adminAddTrustedPlatform`, `adminRemoveTrustedPlatform` (Task 8); backend endpoints (Tasks 5, 7); `listModerationEvents("pending_review")` (backend `listEvents` already accepts any valid status via `Events.List`).

- [x] **Step 1: Widen `listModerationEvents`** — change the status parameter type to `"published" | "rejected" | "pending_review"`.

- [x] **Step 2: AdminModeration pending chip** — read the component fully first. Add a third filter chip «Ссылки» (`filter === "links"`) that loads `listModerationEvents("pending_review")`; row actions for a `pending_review` row: «Опубликовать» → `approveEvent(row.id)` (then remove from queue, same optimistic pattern as reinstate ~line 192) and the existing takedown/reject flow for declines. Follow the component's existing chip/row/action idioms exactly.

- [x] **Step 3: AdminTrustedPlatforms component** — table of `adminListTrustedPlatforms()` rows (domain, name, category, active), a three-field add form (domain / название / категория select of the 4 values) calling `adminAddTrustedPlatform` then reloading, and a «Отключить» button per active row calling `adminRemoveTrustedPlatform`. Copy layout primitives from `AdminModeration.tsx`/`frontend/app/admin/settings` (read them; reuse Chip/Input components).

- [x] **Step 4: Mount + verify** — mount the component; run the dev stack, log in as staff, verify: pending event appears under «Ссылки», approve publishes it (event page shows it live), adding a domain makes the form hint in Task 9 recognize it.

- [x] **Step 5: Typecheck + tests** — `cd frontend && npx tsc --noEmit && npx vitest run` → PASS.

- [x] **Step 6: Commit**

```bash
git add frontend/components/AdminModeration.tsx frontend/components/AdminTrustedPlatforms.tsx frontend/lib/api.ts frontend/app/admin/
git commit -m "feat(admin): pending-links moderation queue + trusted-platforms management"
```

---

### Task 12: Full verification pass

- [x] **Step 1: Backend** — `cd backend && go build ./... && go test ./... && golangci-lint run` → all green.
- [x] **Step 2: Frontend** — `cd frontend && npx tsc --noEmit && npx vitest run && npm run build` → all green.
- [x] **Step 3: End-to-end smoke (local stack)** — as organizer: create external event with `https://xxx.timepad.ru/event/1` + «мест ограничено» → publishes immediately; event page shows «Билеты на TimePad» (set a price) and «Ограничено · наличие на сайте регистрации». Create one with an unknown domain → banner «Событие отправлено на проверку модератору…», feed does not show it; as admin approve it → visible. Paid open-signup event shows «Оплата на месте».
- [x] **Step 4: Commit any fixes; do not push or deploy** — deployment is a separate step (below) done with the user.

---

## Deploy notes (for the deploy session, not part of implementation)

- scp `000026_trusted_platforms.*.sql` to the box `/opt/lia/backend/db/migrations` before migrating (memory `lia-demo-deployment`); prod DB is currently at migration 019 per memory — verify actual prod version first (`schema_migrations`), migrations 020–025 may need to ride along.
- Backend image: build on Mac → save | ssh | load; `docker tag <newimg> backend-app:latest` BEFORE `up -d --force-recreate` (memory `lia-demologin-prod-bypass` gotcha); tag rollback from the RUNNING container after load.
- Frontend build needs both build-args (`NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_YANDEX_MAPS_KEY`).
- After verify: docker prune per memory `lia-deploy-image-cleanup`.
