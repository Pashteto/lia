# Organizer Hub Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gather the existing organizer tools behind one «Организаторам» hub at `/organizer`, give it a single front door (desktop header button + repurposed mobile tab), clean the attendee surfaces, and turn «Заявки участников» into a real management view showing each application-mode event with its applicant count and **named** applicants.

**Architecture:** Almost entirely a frontend information-architecture change — a new thin hub page plus an aggregated applications view that reuses the existing per-event applications endpoint and `EventApplicationsPanel`. The one backend change is additive: the applications response gains an `applicant {uuid, name}` read-model field (name only, never email), enriched in `rsvp.ListApplications` by mirroring the `loadOrganizers` batched-lookup pattern. No DB migration.

**Tech Stack:** Frontend — Next.js 16 App Router, React 19, TypeScript, Tailwind v4, `@tanstack/react-query`. Backend — Go modular monolith, go-pg, go-swagger (generated models/spec).

## Global Constraints

- **Soft framing only** — no new permission gating; anyone authed can still create events exactly as today.
- **Hub label/title is «Организаторам»** verbatim, used for the page heading, the desktop header link, and the mobile tab label.
- **Applicant email is NEVER exposed** — name only, mirroring the public-surface rule (event payloads carry organizer name but never email).
- **No DB migration.** The only backend change is one additive read-model field.
- **Swagger regen is `make generate-api` only** — never `make generate-all` (its proto step is known-broken in this repo).
- **Backend `make generate-api` must be run before backend compiles** after editing the swagger spec (it regenerates the gitignored models).
- **Frontend has no unit-test runner** — frontend tasks are verified with `pnpm exec tsc --noEmit` (typecheck) + `pnpm lint`, plus a manual browser check. Do not invent a test runner.
- Attendee routes/tab bar behaviour must be unchanged except the one repurposed 5th tab.

---

### Task 1: Backend — applicant name on the applications response

**Files:**
- Modify: `backend/internal/models/rsvp.go` (add transient `ApplicantName` field to `Rsvp`)
- Modify: `backend/internal/rsvp/repository.go` (add `LoadApplicantNames`; add to `Repository` interface at line ~28)
- Modify: `backend/internal/rsvp/service.go:166-182` (`ListApplications` calls the loader)
- Modify: `backend/api/swagger.yaml` — the `Rsvp` definition at lines 1258-1283 — then run `make generate-api`
- Modify: `backend/internal/http/formatter/rsvp.go` (map `applicant` in `RsvpToAPI`)
- Test: `backend/internal/rsvp/service_test.go` (extend) and/or `backend/internal/http/formatter/rsvp_test.go` (new) — whichever the existing test harness supports with a fake repo

**Interfaces:**
- Produces: API `Rsvp` JSON gains `applicant: { uuid, name }` (only populated on `GET /events/{id}/applications`). Frontend Task 2 consumes `applicant.uuid` + `applicant.name`.
- Produces (Go): `models.Rsvp.ApplicantName string` (transient, `pg:"-"`); `Repository.LoadApplicantNames(rows []*models.Rsvp) error`.

- [ ] **Step 1: Add the transient domain field**

In `backend/internal/models/rsvp.go`, inside `type Rsvp struct`, add below the `Event` transient field:

```go
	// Event is a transient read-model populated by joins (e.g. MyPractices).
	Event *Event `pg:"-"`

	// ApplicantName is a transient display-name read-model populated only for
	// the organizer-facing applications list. Name only — never email.
	ApplicantName string `pg:"-"`
```

- [ ] **Step 2: Add `LoadApplicantNames` to the repository interface**

In `backend/internal/rsvp/repository.go`, add to the `Repository` interface (around line 28):

```go
	// LoadApplicantNames populates ApplicantName on each row from the users
	// table in a single query (no N+1). Name only — email is excluded.
	LoadApplicantNames(rows []*models.Rsvp) error
```

- [ ] **Step 3: Implement `LoadApplicantNames` (mirrors loadOrganizers)**

Add to `backend/internal/rsvp/repository.go` (note `pg` is already imported in this file):

```go
func (r *pgRepository) LoadApplicantNames(rows []*models.Rsvp) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{})
	for _, row := range rows {
		if row.UserID != uuid.Nil {
			if _, ok := seen[row.UserID]; !ok {
				seen[row.UserID] = struct{}{}
				ids = append(ids, row.UserID)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var users []struct {
		UUID uuid.UUID `pg:"uuid"`
		Name string    `pg:"name,use_zero"`
	}
	if _, err := r.db.Query(&users,
		`SELECT uuid, name FROM users WHERE uuid IN (?)`, pg.In(ids),
	); err != nil {
		return fmt.Errorf("load applicant names: %w", err)
	}
	names := make(map[uuid.UUID]string, len(users))
	for _, u := range users {
		names[u.UUID] = u.Name
	}
	for _, row := range rows {
		row.ApplicantName = names[row.UserID]
	}
	return nil
}
```

- [ ] **Step 4: Call the loader from `ListApplications`**

In `backend/internal/rsvp/service.go`, in `ListApplications` (lines 166-182), after the `rows, err := s.repo.ListByEvent(...)` block and before `return rows, nil`:

```go
	rows, err := s.repo.ListByEvent(eventID,
		[]models.RsvpStatus{models.RsvpApplied, models.RsvpAccepted, models.RsvpDeclined})
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	if err := s.repo.LoadApplicantNames(rows); err != nil {
		return nil, fmt.Errorf("load applicant names: %w", err)
	}
	return rows, nil
```

- [ ] **Step 5: Add `applicant` to the swagger `Rsvp` definition + regenerate**

Edit `backend/api/swagger.yaml`, in the `Rsvp:` definition (lines 1258-1283), adding an additive `applicant` property after `application_answer:` (before `created_at:`). Match the existing 6-space property indentation in that file:

```yaml
      application_answer:
        type: string
      applicant:
        type: object
        readOnly: true
        properties:
          uuid:
            type: string
            format: uuid
          name:
            type: string
      created_at:
        type: string
        format: date-time
        readOnly: true
```

Then regenerate (swagger-only):

Run: `cd backend && make generate-api`
Expected: regenerates `internal/http/server/embedded_spec.go` + `internal/http/models/*` with an `Applicant` field on the `Rsvp` model. Confirm `grep -rn "Applicant" internal/http/models/rsvp.go` shows the new field.

- [ ] **Step 6: Map `applicant` in the formatter**

In `backend/internal/http/formatter/rsvp.go`, populate the new field when `ApplicantName` is set. Use the exact generated type name from Step 5 (the generated nested struct will be `apiModels.RsvpApplicant` or similar — confirm with the regen output and match it):

```go
	out := &apiModels.Rsvp{
		ID:                strfmt.UUID(r.ID.String()),
		EventID:           strfmt.UUID(r.EventID.String()),
		UserID:            strfmt.UUID(r.UserID.String()),
		Status:            string(r.Status),
		ApplicationAnswer: r.ApplicationAnswer,
		CreatedAt:         strfmt.DateTime(r.CreatedAt),
	}
	if r.ApplicantName != "" {
		out.Applicant = &apiModels.RsvpApplicant{
			UUID: strfmt.UUID(r.UserID.String()),
			Name: r.ApplicantName,
		}
	}
	if r.Event != nil {
		out.Event = EventToAPI(r.Event)
	}
```

- [ ] **Step 7: Write a test for the formatter mapping**

Add to `backend/internal/http/formatter/rsvp_test.go` (create if absent; match the package + import style of `event_test.go`):

```go
func TestRsvpToAPI_MapsApplicantName(t *testing.T) {
	uid := uuid.New()
	in := &domainModels.Rsvp{
		ID: uuid.New(), EventID: uuid.New(), UserID: uid,
		Status: domainModels.RsvpApplied, ApplicantName: "Иван Петров",
	}
	out := RsvpToAPI(in)
	if out.Applicant == nil {
		t.Fatal("expected applicant to be set")
	}
	if out.Applicant.Name != "Иван Петров" {
		t.Errorf("name = %q, want %q", out.Applicant.Name, "Иван Петров")
	}
	if out.Applicant.UUID.String() != uid.String() {
		t.Errorf("uuid = %q, want %q", out.Applicant.UUID, uid)
	}
}

func TestRsvpToAPI_OmitsApplicantWhenNameEmpty(t *testing.T) {
	out := RsvpToAPI(&domainModels.Rsvp{
		ID: uuid.New(), EventID: uuid.New(), UserID: uuid.New(),
		Status: domainModels.RsvpApplied,
	})
	if out.Applicant != nil {
		t.Error("expected applicant nil when name empty")
	}
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd backend && go test ./internal/http/formatter/... ./internal/rsvp/...`
Expected: PASS (including the two new formatter cases). If `service_test.go` has a fake `Repository`, add a no-op `LoadApplicantNames` to it so it still satisfies the interface.

- [ ] **Step 9: Build the backend**

Run: `cd backend && go build ./...`
Expected: no errors.

- [ ] **Step 10: Commit**

```bash
git add backend/internal/models/rsvp.go backend/internal/rsvp/repository.go backend/internal/rsvp/service.go backend/internal/http/formatter/rsvp.go backend/internal/http/formatter/rsvp_test.go backend/internal/http/server/embedded_spec.go
git add -A backend/internal/http/models
git commit -m "feat(rsvp): expose applicant name on applications response (name only)"
```

---

### Task 2: Frontend — Rsvp type + mapping + panel shows applicant name

**Files:**
- Modify: `frontend/lib/types.ts` (add `applicant` to `Rsvp`)
- Modify: `frontend/lib/api.ts:384-393` (`apiRsvpToLia`) + the `ApiRsvp` type it consumes
- Modify: `frontend/components/EventApplicationsPanel.tsx` (render the name)

**Interfaces:**
- Consumes: API `Rsvp.applicant` from Task 1.
- Produces: `Rsvp.applicant?: { id: string; name: string }` on the frontend type, rendered by `EventApplicationsPanel` and reused by Task 3.

- [ ] **Step 1: Add `applicant` to the `Rsvp` type**

In `frontend/lib/types.ts`, in `interface Rsvp` (after `applicationAnswer?`):

```ts
export interface Rsvp {
  id: string;
  eventId: string;
  status: RsvpStatus;
  applicationAnswer?: string;
  applicant?: { id: string; name: string };
  createdAt: string;
  event?: LiaEvent;
}
```

- [ ] **Step 2: Map it in `apiRsvpToLia`**

In `frontend/lib/api.ts`, update the `ApiRsvp` type (search for its definition) to include `applicant?: { uuid: string; name: string }`, then update `apiRsvpToLia` (lines 384-393):

```ts
function apiRsvpToLia(r: ApiRsvp): Rsvp {
  return {
    id: r.id,
    eventId: r.event_id,
    status: r.status,
    applicationAnswer: r.application_answer || undefined,
    applicant: r.applicant
      ? { id: r.applicant.uuid, name: r.applicant.name }
      : undefined,
    createdAt: r.created_at,
    event: r.event ? apiEventToLia(r.event) : undefined,
  };
}
```

- [ ] **Step 3: Render the applicant name in the panel**

In `frontend/components/EventApplicationsPanel.tsx`, inside the `.map((rsvp) => ...)` row, add a name line above the `applicationAnswer` paragraph (within the `min-w-0 flex-1 space-y-1` div):

```tsx
              <div className="min-w-0 flex-1 space-y-1">
                <p className="text-[14px] font-medium text-label">
                  {rsvp.applicant?.name || "Участник"}
                </p>
                {rsvp.applicationAnswer ? (
                  <p className="text-[14px] leading-snug">{rsvp.applicationAnswer}</p>
                ) : (
                  <p className="text-[13px] italic text-label-secondary">Ответ не указан</p>
                )}
```

- [ ] **Step 4: Typecheck + lint**

Run: `cd frontend && pnpm exec tsc --noEmit && pnpm lint`
Expected: no type errors, no lint errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/types.ts frontend/lib/api.ts frontend/components/EventApplicationsPanel.tsx
git commit -m "feat(frontend): show applicant name in applications panel"
```

---

### Task 3: Frontend — aggregated applications view at `/organizer/applications`

**Files:**
- Create: `frontend/app/organizer/applications/page.tsx`

**Interfaces:**
- Consumes: `fetchMyEvents()` (`frontend/lib/api.ts:179`), `LiaEvent` (`signupMode` field), and `EventApplicationsPanel` (Task 2).
- Produces: route `/organizer/applications` — linked from the hub card in Task 4.

- [ ] **Step 1: Create the page**

Create `frontend/app/organizer/applications/page.tsx`:

```tsx
"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";

import { EventApplicationsPanel } from "@/components/EventApplicationsPanel";
import { fetchMyEvents } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { LiaEvent } from "@/lib/types";

// Aggregated organizer view: each application-mode event the organizer owns,
// with its applicant list (named) and inline accept/decline (via the panel).
export default function OrganizerApplicationsPage() {
  const { isAuthed, ready } = useAuth();
  const { data: events = [], isLoading, isError } = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: ready && isAuthed,
  });

  if (!ready) return <div className="min-h-screen bg-bg-grouped" />;

  if (!isAuthed) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16">
        <h1 className="text-[28px] font-bold tracking-[-0.022em]">Заявки участников</h1>
        <p className="mt-2 text-label-secondary">Войдите, чтобы видеть заявки на ваши события.</p>
      </main>
    );
  }

  const applicationEvents = events.filter(
    (e: LiaEvent) => e.signupMode === "application",
  );

  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <Link href="/organizer" className="text-[14px] text-accent">← Организаторам</Link>
      <h1 className="mt-2 mb-6 text-[28px] font-bold tracking-[-0.022em]">Заявки участников</h1>

      {isLoading && <p className="text-label-secondary">Загрузка…</p>}
      {isError && <p className="text-red-500">Не удалось загрузить события.</p>}

      {!isLoading && !isError && applicationEvents.length === 0 && (
        <p className="text-label-secondary">
          У вас пока нет событий с записью по заявкам. Создайте событие с режимом
          «по заявке», чтобы принимать участников вручную.
        </p>
      )}

      <div className="space-y-6">
        {applicationEvents.map((event: LiaEvent) => (
          <section key={event.id} className="rounded-card bg-bg p-4 shadow-card-subtle">
            <div className="flex items-center justify-between gap-3">
              <Link href={`/events/${event.id}`} className="text-[17px] font-semibold">
                {event.title}
              </Link>
            </div>
            <EventApplicationsPanel eventId={event.id} eventTitle={undefined} />
          </section>
        ))}
      </div>
    </main>
  );
}
```

- [ ] **Step 2: Confirm `signupMode` exists on `LiaEvent`**

Run: `grep -n "signupMode" frontend/lib/types.ts`
Expected: a `signupMode?` field is present (it is used in `app/events/mine/page.tsx:111`). If its values differ from `"application"`, match the literal used there.

- [ ] **Step 3: Typecheck + lint**

Run: `cd frontend && pnpm exec tsc --noEmit && pnpm lint`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add frontend/app/organizer/applications/page.tsx
git commit -m "feat(frontend): aggregated applications view at /organizer/applications"
```

---

### Task 4: Frontend — the `/organizer` hub page

**Files:**
- Create: `frontend/app/organizer/page.tsx`

**Interfaces:**
- Consumes: `getMyOrganizer()` (`frontend/lib/api.ts:646`, returns `Organizer | null` with a `verification_status`), `fetchMyEvents()`, `useAuth`.
- Produces: route `/organizer` — the destination of the header link + mobile tab in Task 5.

- [ ] **Step 1: Confirm the organizer status field name**

Run: `grep -n "verification_status\|verificationStatus" frontend/lib/api.ts frontend/app/me/organizer/page.tsx`
Expected: confirm the property name on the `Organizer` returned by `getMyOrganizer()` (the spec/HANDOFF call it `verification_status` with values `draft|pending|verified|rejected`). Use the confirmed name below.

- [ ] **Step 2: Create the hub page**

Create `frontend/app/organizer/page.tsx` (uses the status field confirmed in Step 1 — shown here as `verification_status`):

```tsx
"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { fetchMyEvents, getMyOrganizer } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { LiaEvent } from "@/lib/types";

const VERIFY_LABEL: Record<string, string> = {
  draft: "Профиль не отправлен",
  pending: "На проверке",
  verified: "Подтверждённый организатор",
  rejected: "Отклонён — отредактируйте профиль",
};

const VERIFY_CLASS: Record<string, string> = {
  draft: "text-label-secondary",
  pending: "text-amber-600",
  verified: "text-green-600",
  rejected: "text-red-500",
};

export default function OrganizerHubPage() {
  const { isAuthed, ready } = useAuth();

  const { data: organizer } = useQuery({
    queryKey: ["my-organizer"],
    queryFn: getMyOrganizer,
    enabled: ready && isAuthed,
  });
  const { data: events = [] } = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: ready && isAuthed,
  });

  const drafts = events.filter((e: LiaEvent) => e.status === "draft").length;
  const published = events.filter((e: LiaEvent) => e.status === "published").length;
  const status = organizer?.verification_status ?? "draft";

  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[28px] font-bold tracking-[-0.022em]">Организаторам</h1>
          <p className="mt-1 text-[14px] text-label-secondary">
            Создавайте и ведите свои события
          </p>
        </div>
        <Link href="/events/new">
          <Button variant="tinted">+ Создать событие</Button>
        </Link>
      </div>

      {isAuthed && (
        <Link
          href="/me/organizer"
          className="mt-5 flex items-center gap-2 rounded-card bg-bg p-3.5 shadow-card-subtle"
        >
          <span className={`text-[14px] font-medium ${VERIFY_CLASS[status] ?? ""}`}>
            {VERIFY_LABEL[status] ?? "Профиль организатора"}
          </span>
          <span className="ml-auto text-[13px] text-label-secondary">
            Профиль организатора →
          </span>
        </Link>
      )}

      <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <HubCard
          href="/events/mine"
          title="Мои события"
          subtitle="Черновики, на модерации, опубликованные"
        >
          {isAuthed && (
            <div className="mt-2 flex gap-2 text-[11px]">
              <span className="rounded-control bg-amber-100 px-2 py-0.5 text-amber-800">
                {drafts} черновиков
              </span>
              <span className="rounded-control bg-green-100 px-2 py-0.5 text-green-800">
                {published} опубликовано
              </span>
            </div>
          )}
        </HubCard>

        <HubCard
          href="/organizer/applications"
          title="Заявки участников"
          subtitle="Подтвердить или отклонить запись"
        />

        <HubCard
          href="/me/organizer"
          title="Профиль организатора"
          subtitle="Название, описание, логотип, верификация"
        />

        <div className="rounded-card bg-bg p-4 opacity-50 shadow-card-subtle">
          <h3 className="text-[16px] font-semibold">Подписчики</h3>
          <p className="mt-0.5 text-[13px] text-label-secondary">
            Кто следит за вашими событиями (позже)
          </p>
        </div>
      </div>
    </main>
  );
}

function HubCard({
  href,
  title,
  subtitle,
  children,
}: {
  href: string;
  title: string;
  subtitle: string;
  children?: React.ReactNode;
}) {
  return (
    <Link href={href} className="rounded-card bg-bg p-4 shadow-card-subtle transition hover:shadow-card">
      <h3 className="text-[16px] font-semibold">{title}</h3>
      <p className="mt-0.5 text-[13px] text-label-secondary">{subtitle}</p>
      {children}
    </Link>
  );
}
```

- [ ] **Step 3: Verify the event status literals**

Run: `grep -n "status === \"draft\"\|status === \"published\"\|status:" frontend/app/events/mine/page.tsx frontend/lib/types.ts`
Expected: confirm `LiaEvent.status` uses the literals `"draft"` / `"published"`. Adjust the filters in Step 2 if the literals differ.

- [ ] **Step 4: Typecheck + lint**

Run: `cd frontend && pnpm exec tsc --noEmit && pnpm lint`
Expected: clean. (If `Button` does not accept `variant="tinted"`, match the variant used on `app/page.tsx:27`.)

- [ ] **Step 5: Commit**

```bash
git add frontend/app/organizer/page.tsx
git commit -m "feat(frontend): /organizer hub page (Организаторам)"
```

---

### Task 5: Frontend — front door (header + tab bar) and remove home create button

**Files:**
- Modify: `frontend/components/AuthButton.tsx:35-40` (header link)
- Modify: `frontend/components/ui/TabBar.tsx:13-19,18` (tab) + add `GlyphOrganizer`; review the hide-logic at lines 29-33
- Modify: `frontend/app/page.tsx:26-28` (remove the standalone create button)

**Interfaces:**
- Consumes: `/organizer` route (Task 4).
- Produces: the only entry points to the hub.

- [ ] **Step 1: Replace the header link**

In `frontend/components/AuthButton.tsx`, replace the «Мои события» `Link` (lines 35-40) with:

```tsx
        <Link
          href="/organizer"
          className="text-[14px] font-medium text-accent"
        >
          Организаторам
        </Link>
```

- [ ] **Step 2: Repurpose the 5th mobile tab + add a glyph**

In `frontend/components/ui/TabBar.tsx`, change the 5th entry (line 18) from the «Создать» plus-tab to:

```tsx
  { href: "/organizer", label: "Организаторам", icon: <GlyphOrganizer /> },
```

Then replace the `GlyphPlus` function with `GlyphOrganizer` (an ID-card / door glyph), keeping the same 22×22 `currentColor` style as the other glyphs:

```tsx
function GlyphOrganizer() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <rect x="3" y="5" width="18" height="14" rx="3" stroke="currentColor" strokeWidth="1.8" />
      <circle cx="8.5" cy="11" r="2" stroke="currentColor" strokeWidth="1.6" />
      <path d="M13 9h5M13 12.5h5M5.5 15.5c.6-1.4 4-1.4 4.6 0" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  );
}
```

- [ ] **Step 3: Fix the hide-logic for the old create route**

In `frontend/components/ui/TabBar.tsx`, the guard at lines 29-33 hides the bar on `/events/new`. That still applies (create is now reached via the hub, but `/events/new` is still a real page that should hide the bar — it has its own chrome). Leave the `/events/new` condition as-is. Confirm `/organizer` and `/organizer/applications` are NOT matched by any hide condition (they aren't: not `/events/new`, not event-detail, not `/admin`), so the bar shows with the «Организаторам» tab active on the hub.

- [ ] **Step 4: Remove the home-page create button**

In `frontend/app/page.tsx`, remove the standalone create-button block (lines 26-28):

```tsx
            <Link href="/events/new" className="hidden sm:block">
              <Button variant="tinted">Создать событие</Button>
            </Link>
```

Delete it. If removing it leaves an unused `Button`/`Link` import or an empty wrapper element, clean those up so lint passes.

- [ ] **Step 5: Typecheck + lint**

Run: `cd frontend && pnpm exec tsc --noEmit && pnpm lint`
Expected: clean (no unused-import errors from the removed button).

- [ ] **Step 6: Manual browser verification**

Run the frontend (`cd frontend && pnpm dev`) and confirm:
- Signed-out home page: no «Создать событие» button; mobile tab bar shows «Организаторам» as the 5th tab.
- Header (signed in): «Организаторам» link → `/organizer`; «Мои события» link is gone from the header.
- `/organizer`: hub renders with the four cards + verification strip; tab bar visible with «Организаторам» active.
- `/organizer/applications`: lists application-mode events; each shows the applicant list with names.
- Attendee tabs (События/Подбор/Календарь/Карта) unchanged.

- [ ] **Step 7: Commit**

```bash
git add frontend/components/AuthButton.tsx frontend/components/ui/TabBar.tsx frontend/app/page.tsx
git commit -m "feat(frontend): organizer hub front door + remove home create button"
```

---

## Final verification

- [ ] Backend: `cd backend && go build ./... && go test ./internal/rsvp/... ./internal/http/formatter/...` → PASS.
- [ ] Frontend: `cd frontend && pnpm exec tsc --noEmit && pnpm lint` → clean.
- [ ] End-to-end (against a running backend): create an application-mode event, have a second user apply, and confirm the applicant's **name** appears in `/organizer/applications` and accept/decline works.
- [ ] Confirm the applications API response contains `applicant.name` but **no email** (`curl` the endpoint as the organizer and inspect the JSON).
- [ ] Confirm no DB migration was added (`git status` shows no new file under `backend/migrations`).
