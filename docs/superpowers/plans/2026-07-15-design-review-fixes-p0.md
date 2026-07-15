# Design-Review Fixes — P0 (Pre-Pilot Gate) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the six P0 findings from the design review (`docs/superpowers/specs/2026-07-15-design-review-fixes-design.md`) that gate the pre-pilot launch: R4 status amnesia, admin revoke false-error, publish safety, multi-day range, native file inputs, native datetime inputs.

**Architecture:** Frontend is Next 16 App Router + Tailwind 4 (`frontend/`); backend is a Go modular monolith with go-swagger generated ops (`backend/`). Most tasks are frontend-only; R4 needs a small backend change (populate `my_rsvp_status` on `GET /events/{id}`). Pure functions and Go handlers are built test-first; UI wiring is verified by typecheck/build + a live smoke step, matching the repo's existing practice (only pure-logic vitest tests exist today).

**Tech Stack:** TypeScript, React 19, react-hook-form + zod, @tanstack/react-query, vitest; Go 1.x, go-pg, go-swagger.

## Global Constraints

- All user-facing copy is Russian. Never ship English UI strings or native English browser chrome (no `Choose File`, no native `confirm()`).
- Timezone is fixed to **Europe/Moscow**; all date/time formatting uses `Intl` with `timeZone: "Europe/Moscow"` (see `frontend/lib/format.ts`, `frontend/lib/calendar.ts`). Never format with the runtime zone (causes SSR hydration mismatch #418).
- No new date/HTTP libraries — the repo deliberately uses native `Date` + `Intl` and `fetch`.
- API errors are thrown as `Error` whose message contains the HTTP status (e.g. `"revoke: 409"`); detect status via `err.message.includes("409")`, matching the existing pattern in `CreateEventForm`.
- Frontend commands run from `frontend/`: typecheck `npx tsc --noEmit`, tests `npm test -- --run`, lint `npm run lint`. Backend from `backend/`: `go test ./...`, `go build ./...`.
- Commit after each task with a `fix(...)`/`feat(...)` message; branch is `feat/organizer-suite-r1-r2-r3` (or a fresh `fix/design-review-p0`).

---

### Task 1: Multi-day range formatter (pure function)

Adds `formatEventRange(event)` → "15–17 августа" / "15 июля – 3 августа" / single-day fallback "13 июня, 19:00". Foundation for Task 2.

**Files:**
- Modify: `frontend/lib/format.ts`
- Test: `frontend/lib/__tests__/format.test.ts` (create)

**Interfaces:**
- Consumes: `LiaEvent` (`startsAt: string`, `endsAt?: string`) from `@/lib/types`.
- Produces: `export function formatEventRange(event: Pick<LiaEvent, "startsAt" | "endsAt">): string`.

- [ ] **Step 1: Write the failing test**

```ts
// frontend/lib/__tests__/format.test.ts
import { describe, expect, it } from "vitest";
import { formatEventRange } from "@/lib/format";

describe("formatEventRange", () => {
  it("single-day (no end) keeps the time form", () => {
    // 13 June 2026 19:00 MSK == 16:00Z
    expect(formatEventRange({ startsAt: "2026-06-13T16:00:00Z" }))
      .toBe("13 июня, 19:00");
  });

  it("same civil day (end present) still shows the start time only", () => {
    expect(
      formatEventRange({
        startsAt: "2026-06-13T16:00:00Z",
        endsAt: "2026-06-13T18:00:00Z",
      }),
    ).toBe("13 июня, 19:00");
  });

  it("multi-day within one month → '15–17 августа'", () => {
    expect(
      formatEventRange({
        startsAt: "2026-08-15T09:00:00Z",
        endsAt: "2026-08-17T09:00:00Z",
      }),
    ).toBe("15–17 августа");
  });

  it("multi-day across months → '31 июля – 2 августа'", () => {
    expect(
      formatEventRange({
        startsAt: "2026-07-31T09:00:00Z",
        endsAt: "2026-08-02T09:00:00Z",
      }),
    ).toBe("31 июля – 2 августа");
  });

  it("treats a zero-time end as unset (open-ended)", () => {
    expect(
      formatEventRange({
        startsAt: "2026-06-13T16:00:00Z",
        endsAt: "0001-01-01T00:00:00Z",
      }),
    ).toBe("13 июня, 19:00");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run lib/__tests__/format.test.ts`
Expected: FAIL — `formatEventRange is not a function`.

- [ ] **Step 3: Write minimal implementation**

Add to `frontend/lib/format.ts` (below `formatEventDate`):

```ts
// Day + month only, e.g. "15 августа" — for the ends of a multi-day range.
const dayMonthFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  timeZone: "Europe/Moscow",
});
// Day only, e.g. "15" — used when both ends share a month.
const dayFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  timeZone: "Europe/Moscow",
});
// The Europe/Moscow civil day ("YYYY-MM-DD") and month of an instant.
function moscowParts(iso: string): { day: string; month: string } {
  // en-CA yields ISO "YYYY-MM-DD"; split off day/month for cheap comparison.
  const [y, m, d] = new Intl.DateTimeFormat("en-CA", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: "Europe/Moscow",
  })
    .format(new Date(iso))
    .split("-");
  return { day: `${y}-${m}-${d}`, month: m };
}
// A backend zero-time ("0001-01-01…") means "no end set" (see the ends_at hotfix).
function hasRealEnd(endsAt?: string): boolean {
  return !!endsAt && new Date(endsAt).getUTCFullYear() > 1;
}

/**
 * Human date for a card/detail. Single-day events keep the familiar
 * "13 июня, 19:00" form; multi-day events render a civil-day range
 * ("15–17 августа" within a month, "31 июля – 2 августа" across months).
 */
export function formatEventRange(
  event: Pick<LiaEvent, "startsAt" | "endsAt">,
): string {
  if (!hasRealEnd(event.endsAt)) return formatEventDate(event.startsAt);
  const start = moscowParts(event.startsAt);
  const end = moscowParts(event.endsAt as string);
  if (start.day === end.day) return formatEventDate(event.startsAt);
  if (start.month === end.month) {
    // Same month → "15–17 августа" (end carries the month word).
    return `${dayFmt.format(new Date(event.startsAt))}–${dayMonthFmt.format(
      new Date(event.endsAt as string),
    )}`;
  }
  // Cross-month → "31 июля – 2 августа" (spaced en-dash, both months).
  return `${dayMonthFmt.format(new Date(event.startsAt))} – ${dayMonthFmt.format(
    new Date(event.endsAt as string),
  )}`;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --run lib/__tests__/format.test.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/format.ts frontend/lib/__tests__/format.test.ts
git commit -m "feat(format): formatEventRange for multi-day events"
```

---

### Task 2: Render the range on every surface (feed, detail, list, calendar)

Applies `formatEventRange` on the four surfaces the review flagged, and spans multi-day events across all their days in the calendar.

**Files:**
- Modify: `frontend/components/ui/EventCard.tsx:4,54`
- Modify: `frontend/components/EventDetailView.tsx:9,68`
- Modify: `frontend/app/me/practices/page.tsx:12,67`
- Modify: `frontend/app/me/applications/page.tsx:11,66` (event date only — leave `rsvp.createdAt` at line 70 as `formatEventDate`)
- Modify: `frontend/app/me/calendar/page.tsx:132-141` (byDay bucketing)
- Test: `frontend/lib/__tests__/calendar-span.test.ts` (create)

**Interfaces:**
- Consumes: `formatEventRange` (Task 1), `moscowDayKey`/`civil`/`addDays`/`civilKey` from `@/lib/calendar`.
- Produces: `export function eventDayKeys(startsAt: string, endsAt: string | undefined): string[]` in `frontend/lib/calendar.ts`.

- [ ] **Step 1: Write the failing test for the calendar-span helper**

```ts
// frontend/lib/__tests__/calendar-span.test.ts
import { describe, expect, it } from "vitest";
import { eventDayKeys } from "@/lib/calendar";

describe("eventDayKeys", () => {
  it("single-day → one key", () => {
    expect(eventDayKeys("2026-08-15T09:00:00Z", undefined)).toEqual(["2026-08-15"]);
  });
  it("multi-day → every civil day inclusive", () => {
    expect(eventDayKeys("2026-08-15T09:00:00Z", "2026-08-17T09:00:00Z")).toEqual([
      "2026-08-15",
      "2026-08-16",
      "2026-08-17",
    ]);
  });
  it("zero-time end → single day", () => {
    expect(eventDayKeys("2026-08-15T09:00:00Z", "0001-01-01T00:00:00Z")).toEqual([
      "2026-08-15",
    ]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run lib/__tests__/calendar-span.test.ts`
Expected: FAIL — `eventDayKeys is not a function`.

- [ ] **Step 3: Add `eventDayKeys` to `frontend/lib/calendar.ts`**

Append (uses the existing `moscowDayKey`, `civil`, `addDays`, `civilKey`):

```ts
/**
 * The Europe/Moscow civil day keys ("YYYY-MM-DD") an event occupies, inclusive
 * of start and end. Single-day (or open-ended, i.e. zero-time end) → one key.
 * Used to span multi-day events across every calendar cell they cover.
 */
export function eventDayKeys(startsAt: string, endsAt: string | undefined): string[] {
  const startKey = moscowDayKey(new Date(startsAt));
  const realEnd = endsAt && new Date(endsAt).getUTCFullYear() > 1 ? endsAt : undefined;
  if (!realEnd) return [startKey];
  const endKey = moscowDayKey(new Date(realEnd));
  if (endKey <= startKey) return [startKey];
  const [sy, sm, sd] = startKey.split("-").map(Number);
  const keys: string[] = [];
  let cursor = civil(sy, sm - 1, sd);
  while (civilKey(cursor) <= endKey) {
    keys.push(civilKey(cursor));
    cursor = addDays(cursor, 1);
  }
  return keys;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --run lib/__tests__/calendar-span.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Wire the calendar bucketing to span days**

In `frontend/app/me/calendar/page.tsx`, replace the single-day bucketing in the `byDay` useMemo (currently ~lines 132-141):

```tsx
  const byDay = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const ev of events) {
      for (const key of eventDayKeys(ev.startsAt, ev.endsAt)) {
        const list = map.get(key) ?? [];
        list.push(ev);
        map.set(key, list);
      }
    }
    for (const list of map.values()) {
      list.sort((a, b) => a.startsAt.localeCompare(b.startsAt));
    }
    return map;
  }, [events]);
```

Add `eventDayKeys` to the existing `@/lib/calendar` import at the top of the file (join the list that already imports `moscowDayKey`).

- [ ] **Step 6: Swap the date formatter on the three text surfaces**

In each file, change the import to add `formatEventRange` and replace the call:

- `frontend/components/ui/EventCard.tsx`: import `formatEventRange`; line 54 `{formatEventDate(event.startsAt)}` → `{formatEventRange(event)}`.
- `frontend/components/EventDetailView.tsx`: import `formatEventRange`; line 68 `value={formatEventDate(event.startsAt)}` → `value={formatEventRange(event)}`.
- `frontend/app/me/practices/page.tsx`: import `formatEventRange`; line 67 `{formatEventDate(event.startsAt)}` → `{formatEventRange(event)}`.
- `frontend/app/me/applications/page.tsx`: import `formatEventRange`; line 66 `{formatEventDate(event.startsAt)}` → `{formatEventRange(event)}`. **Leave line 70** (`{formatEventDate(rsvp.createdAt)}`) unchanged — that's the submission timestamp, not the event window.

Keep `formatEventDate` imported where line 70 still uses it (applications page keeps both).

- [ ] **Step 7: Typecheck**

Run: `cd frontend && npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 8: Commit**

```bash
git add frontend/lib/calendar.ts frontend/lib/__tests__/calendar-span.test.ts \
  frontend/components/ui/EventCard.tsx frontend/components/EventDetailView.tsx \
  frontend/app/me/practices/page.tsx frontend/app/me/applications/page.tsx \
  frontend/app/me/calendar/page.tsx
git commit -m "feat(events): render multi-day range on feed/detail/list/calendar"
```

---

### Task 3: Publish safety — draft-by-default + styled confirm modal

Two changes: (a) new events default to Черновик; (b) a reusable styled `ConfirmModal` replaces the native `confirm()` in `PublishEventButton`.

**Files:**
- Create: `frontend/components/ui/ConfirmModal.tsx`
- Modify: `frontend/components/CreateEventForm.tsx:105`
- Modify: `frontend/components/PublishEventButton.tsx`
- Test: `frontend/components/__tests__/create-event-schema.test.ts` (extend — assert default)

**Interfaces:**
- Produces: `ConfirmModal` — `{ title: string; body?: string; confirmLabel: string; cancelLabel?: string; danger?: boolean; onConfirm: () => void; onClose: () => void }`.

- [ ] **Step 1: Change the create-form default status**

In `frontend/components/CreateEventForm.tsx`, line ~105, change the default and its comment:

```tsx
      // Default to Черновик so an accidental "Сохранить" never publishes a
      // half-built event. Publishing is an explicit choice (status control, or
      // the "Опубликовать" action on /events/mine behind a confirm).
      status: initial?.status ?? "draft",
```

- [ ] **Step 2: Create the reusable ConfirmModal**

```tsx
// frontend/components/ui/ConfirmModal.tsx
"use client";

import { Button } from "@/components/ui/Button";

/**
 * Centered, dimmed confirmation modal — the styled Russian replacement for the
 * native window.confirm(). Backdrop click and Отмена both dismiss.
 */
export function ConfirmModal({
  title,
  body,
  confirmLabel,
  cancelLabel = "Отмена",
  danger = false,
  onConfirm,
  onClose,
}: {
  title: string;
  body?: string;
  confirmLabel: string;
  cancelLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm rounded-card bg-bg p-5 shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-1 text-[17px] font-semibold">{title}</h2>
        {body && <p className="mb-3 text-[15px] text-label-secondary">{body}</p>}
        <div className="mt-4 flex items-center justify-end gap-2">
          <button
            type="button"
            className="px-3 py-2 text-[15px] text-label"
            onClick={onClose}
          >
            {cancelLabel}
          </button>
          <Button
            type="button"
            variant="filled"
            onClick={onConfirm}
            className={danger ? "bg-red-500 hover:opacity-90" : undefined}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Use ConfirmModal in PublishEventButton**

Rewrite `frontend/components/PublishEventButton.tsx` to hold an `open` state and render `ConfirmModal` instead of calling `window.confirm`:

```tsx
"use client";

import { useState } from "react";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { getToken } from "@/lib/auth";

// Self-contained so it does not depend on the (concurrently edited) lib/api.ts.
const API_V1 = `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/v1`;

/**
 * Publishes a draft event via PATCH /events/{id} with {status:"published"},
 * behind a styled confirmation modal (never native confirm()). Publishing is
 * one-way: the backend locks a published event from further edits. On success,
 * invalidates the "my-events" query so the card re-renders without its badge.
 */
export function PublishEventButton({ eventId }: { eventId: string }) {
  const qc = useQueryClient();
  const [confirming, setConfirming] = useState(false);

  const mutation = useMutation({
    mutationFn: async () => {
      const token = getToken();
      if (!token) throw new Error("not authenticated");
      const res = await fetch(`${API_V1}/events/${eventId}`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ status: "published" }),
      });
      if (!res.ok) {
        throw new Error(`publish failed: ${res.status} ${await res.text().catch(() => "")}`);
      }
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["my-events"] }),
  });

  return (
    <div className="mt-1">
      <button
        type="button"
        onClick={() => setConfirming(true)}
        disabled={mutation.isPending}
        className="flex w-full items-center justify-center gap-1 rounded-control px-2 py-1.5 text-[13px] font-medium text-accent hover:bg-accent/8 transition disabled:opacity-50"
      >
        {mutation.isPending ? "Публикация…" : "Опубликовать"}
      </button>
      {mutation.isError && (
        <p className="px-2 text-[12px] text-red-500">Не удалось опубликовать.</p>
      )}
      {confirming && (
        <ConfirmModal
          title="Опубликовать событие?"
          body="После публикации изменить его будет нельзя."
          confirmLabel="Опубликовать"
          onConfirm={() => {
            setConfirming(false);
            mutation.mutate();
          }}
          onClose={() => setConfirming(false)}
        />
      )}
    </div>
  );
}
```

- [ ] **Step 4: Extend the schema test to lock the default**

Add to `frontend/components/__tests__/create-event-schema.test.ts`:

```ts
import { CreateEventForm } from "@/components/CreateEventForm";
// (Import path check: eventFormSchema is exported from CreateEventForm.)

it("new-event form defaults status to draft", () => {
  // Guard against a regression to publish-by-default. The default lives in the
  // component's defaultValues; assert via a light structural check.
  const src = CreateEventForm.toString();
  expect(src).toContain('initial?.status ?? "draft"');
});
```

If `CreateEventForm.toString()` is unreliable under the bundler, instead assert on the source file with `fs.readFileSync` of `components/CreateEventForm.tsx` containing `initial?.status ?? "draft"`. Pick whichever runs green.

- [ ] **Step 5: Run tests + typecheck**

Run: `cd frontend && npm test -- --run && npx tsc --noEmit`
Expected: PASS; no type errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/components/ui/ConfirmModal.tsx frontend/components/CreateEventForm.tsx \
  frontend/components/PublishEventButton.tsx frontend/components/__tests__/create-event-schema.test.ts
git commit -m "fix(publish): draft-by-default + styled confirm modal (no native confirm)"
```

---

### Task 4: Admin action buttons — in-flight guard + 409-as-success

Fixes the confirmed revoke false-error. Disable while in flight; on success refetch; treat `409` ("already in that state") as success, not a red error. Apply to revoke + auto-verify on `/admin/organizers`; extend the same shape to reject and takedown.

**Files:**
- Modify: `frontend/app/admin/organizers/page.tsx:19-71,163-170`
- Modify: `frontend/app/admin/moderation/organizers/page.tsx` (reject handler ~57-62)
- Modify: `frontend/app/admin/complaints/page.tsx` (takedown/dismiss handler ~53-58)

**Interfaces:**
- Consumes: `revokeOrganizer`, `setOrganizerAutoVerify` (throw `Error` with status in message).
- Produces: local `acting` boolean state guarding each handler; a shared `is409` helper inline.

- [ ] **Step 1: Add in-flight state + 409-tolerant revoke to `/admin/organizers`**

In `frontend/app/admin/organizers/page.tsx`, add an `acting` state and a helper, and rewrite `onRevoke`/`onToggleAuto`:

```tsx
  const [acting, setActing] = useState(false);
```

```tsx
  // A 409 means the org is already in the target state — the mutation the user
  // wanted has effectively happened. Treat it as success (refetch), not error.
  function is409(err: unknown): boolean {
    return err instanceof Error && err.message.includes("409");
  }

  async function onRevoke() {
    if (!selected || acting) return;
    if (!revokeReason.trim()) {
      setActionError("Укажите причину отзыва");
      return;
    }
    setActing(true);
    try {
      await revokeOrganizer(selected.id, revokeReason.trim());
      setRevokeReason("");
      setActionError("");
      await open(selected.id);
    } catch (err) {
      if (is409(err)) {
        setRevokeReason("");
        setActionError("");
        await open(selected.id);
      } else {
        setActionError("Не удалось отозвать подтверждение");
      }
    } finally {
      setActing(false);
    }
  }

  async function onToggleAuto() {
    if (!selected || acting) return;
    setActing(true);
    try {
      setActionError("");
      await setOrganizerAutoVerify(selected.id, !selected.auto_verify);
      await open(selected.id);
    } catch (err) {
      if (is409(err)) {
        await open(selected.id);
      } else {
        setActionError("Не удалось изменить авто-подтверждение");
      }
    } finally {
      setActing(false);
    }
  }
```

- [ ] **Step 2: Disable the revoke button + auto-verify checkbox while acting**

In the same file: the revoke `<Button>` (line ~163) — change `disabled={!revokeReason.trim()}` to `disabled={acting || !revokeReason.trim()}` and its label to reflect progress:

```tsx
              <Button
                variant="tinted"
                disabled={acting || !revokeReason.trim()}
                onClick={onRevoke}
                className="text-red-500 hover:bg-red-500/10 disabled:opacity-40"
              >
                {acting ? "Отзываем…" : "Отозвать подтверждение"}
              </Button>
```

And the auto-verify checkbox (line ~136): add `disabled={acting}` to the `<input type="checkbox" …>`.

- [ ] **Step 3: Apply the same guard to reject (moderation/organizers)**

In `frontend/app/admin/moderation/organizers/page.tsx`, guard the reject handler (~line 57): add an `acting` state (or reuse existing pending state if present), wrap `await rejectOrganizer(...)` so a re-fire is ignored while in flight, and treat a `409` as success (refetch the queue) rather than surfacing `setActionError`. Disable the reject confirm button while acting (the button at line ~188 already has `disabled={!reason.trim()}` — change to `disabled={acting || !reason.trim()}`).

```tsx
  const [acting, setActing] = useState(false);
  // ...
  async function onReject() {
    if (!rejectTarget || acting) return;
    if (!reason.trim()) return;
    setActing(true);
    try {
      await rejectOrganizer(rejectTarget.id, reason.trim());
      // ...existing success path (close modal, refetch)...
    } catch (err) {
      if (err instanceof Error && err.message.includes("409")) {
        // already rejected — refetch, no error banner
        // ...existing success path...
      } else {
        setActionError("Не удалось отклонить организатора");
      }
    } finally {
      setActing(false);
    }
  }
```

(Adapt to the file's existing state names — read them first; keep the existing success/refetch statements verbatim, only adding the `acting` guard and the `409` branch.)

- [ ] **Step 4: Apply the same guard to takedown + dismiss (complaints)**

In `frontend/app/admin/complaints/page.tsx`, guard the takedown handler (~line 53) and the dismiss handler the same way: an `acting` boolean prevents double-fire; a `409` on takedown (event already taken down) refetches instead of erroring. Disable the "Снять и закрыть жалобы" and "Отклонить" buttons while `acting`.

- [ ] **Step 5: Typecheck + build**

Run: `cd frontend && npx tsc --noEmit && npm run build`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/app/admin/organizers/page.tsx \
  frontend/app/admin/moderation/organizers/page.tsx \
  frontend/app/admin/complaints/page.tsx
git commit -m "fix(admin): in-flight guard + 409-as-success on revoke/reject/takedown/dismiss"
```

---

### Task 5: R4 backend — populate `my_rsvp_status` on `GET /events/{id}`

The domain `Event.MyRsvpStatus` and the API formatter already exist (`internal/http/formatter/event.go:132`); the field is just never set on the single-event path. Resolve the caller and set it.

**Files:**
- Modify: `backend/internal/rsvp/service.go` (add `StatusForUser` to the `Service` interface + impl)
- Modify: `backend/internal/http/handlers/events.go` (GetEventByID: inject rsvp service, set `MyRsvpStatus`)
- Modify: `backend/internal/http/module.go:258` (wire the rsvp service into `NewGetEventByID`)
- Test: `backend/internal/rsvp/service_test.go` (add a `StatusForUser` case) or `backend/internal/http/handlers/events_test.go`

**Interfaces:**
- Consumes: `Repository.GetUserRsvp(eventID, userID) (*models.Rsvp, error)` (already exists), `pg.ErrNoRows` via `isNoRows`.
- Produces: `StatusForUser(ctx context.Context, eventID, userID uuid.UUID) (models.RsvpStatus, error)` on `rsvp.Service` — returns `""` when the user has no RSVP.

- [ ] **Step 1: Write the failing service test**

Add to `backend/internal/rsvp/service_test.go` (mirror the existing test style / fake repo in that file):

```go
func TestStatusForUser(t *testing.T) {
	eventID, userID := uuid.NewV4(), uuid.NewV4()

	t.Run("returns the user's status", func(t *testing.T) {
		repo := &fakeRepo{ // existing test double in this package
			userRsvp: &models.Rsvp{Status: models.RsvpStatusGoing},
		}
		svc := NewService(repo)
		got, err := svc.StatusForUser(context.Background(), eventID, userID)
		require.NoError(t, err)
		require.Equal(t, models.RsvpStatusGoing, got)
	})

	t.Run("no rsvp → empty status, no error", func(t *testing.T) {
		repo := &fakeRepo{userRsvpErr: pg.ErrNoRows}
		svc := NewService(repo)
		got, err := svc.StatusForUser(context.Background(), eventID, userID)
		require.NoError(t, err)
		require.Equal(t, models.RsvpStatus(""), got)
	})
}
```

(Read the existing `service_test.go` first to match its fake-repo field names — `userRsvp`/`userRsvpErr` are illustrative; adapt to the real double. If the file uses a mock without a `GetUserRsvp` field, add one.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/rsvp/ -run TestStatusForUser`
Expected: FAIL — `svc.StatusForUser undefined`.

- [ ] **Step 3: Add `StatusForUser` to the interface + impl**

In `backend/internal/rsvp/service.go`, add to the `Service` interface (after `Cancel`):

```go
	// StatusForUser returns the caller's RSVP status on an event, or "" when
	// they have none. Used to render the correct join/apply state on reload.
	StatusForUser(ctx context.Context, eventID, userID uuid.UUID) (models.RsvpStatus, error)
```

And the implementation:

```go
func (s *service) StatusForUser(_ context.Context, eventID, userID uuid.UUID) (models.RsvpStatus, error) {
	if eventID == uuid.Nil || userID == uuid.Nil {
		return "", nil
	}
	r, err := s.repo.GetUserRsvp(eventID, userID)
	if err != nil {
		if isNoRows(err) {
			return "", nil
		}
		return "", err
	}
	return r.Status, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/rsvp/ -run TestStatusForUser`
Expected: PASS.

- [ ] **Step 5: Inject the rsvp service into GetEventByID and set the field**

In `backend/internal/http/handlers/events.go`, extend the `GetEventByID` struct and constructor, and set `MyRsvpStatus` before formatting:

```go
type GetEventByID struct {
	events    eventsdomain.Service
	rsvp      rsvpdomain.Service // optional; nil → my_rsvp_status stays ""
	checkAuth func(string) (*apimodels.User, error)
}

func NewGetEventByID(
	svc eventsdomain.Service,
	rsvp rsvpdomain.Service,
	checkAuth func(string) (*apimodels.User, error),
) *GetEventByID {
	return &GetEventByID{events: svc, rsvp: rsvp, checkAuth: checkAuth}
}
```

In `Handle`, after the not-published owner check and before the OK return, populate the caller's status:

```go
	// Populate my_rsvp_status for the authenticated caller so the detail page
	// renders the correct join/apply state on reload (design-review R4).
	if h.rsvp != nil && h.checkAuth != nil {
		if u, err := h.checkAuth(params.HTTPRequest.Header.Get("Authorization")); err == nil && u != nil {
			if uid, err := uuid.FromString(u.UUID.String()); err == nil {
				if st, err := h.rsvp.StatusForUser(params.HTTPRequest.Context(), params.ID, uid); err == nil {
					event.MyRsvpStatus = string(st)
				}
			}
		}
	}

	return eventsops.NewGetEventByIDOK().WithPayload(formatter.EventToAPI(event))
```

Add the `rsvpdomain "github.com/Pashteto/lia/internal/rsvp"` import if not already present in this file, and ensure `uuid` is imported (it is used elsewhere in the package).

- [ ] **Step 6: Update the wiring in module.go**

In `backend/internal/http/module.go`, line ~258, pass `m.rsvp`:

```go
	api.EventsGetEventByIDHandler = handlers.NewGetEventByID(m.events, m.rsvp, m.auth.CheckAuth)
```

`m.rsvp` may be nil if `SetRsvpService` wasn't called — the handler guards for that (`if h.rsvp != nil`), so ordering is safe.

- [ ] **Step 7: Build + full backend test**

Run: `cd backend && go build ./... && go test ./internal/rsvp/ ./internal/http/...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/rsvp/service.go backend/internal/rsvp/service_test.go \
  backend/internal/http/handlers/events.go backend/internal/http/module.go
git commit -m "feat(events): populate my_rsvp_status on GET /events/{id} (R4 backend)"
```

---

### Task 6: R4 frontend — trust `myRsvpStatus` on load

With the backend now returning it, remove the "do NOT trust" workaround so a reload renders the correct state. Depends on Task 5.

**Files:**
- Modify: `frontend/components/SignupCTA.tsx:94-99`

- [ ] **Step 1: Remove the workaround comment**

In `frontend/components/SignupCTA.tsx`, replace the comment block at lines 94-96 (the `useState` initializer at line 97-99 already reads `event.myRsvpStatus ?? ""` and stays as-is):

```tsx
  // Local state — seeded from the server's my_rsvp_status (populated on
  // GET /events/{id}) and authoritative after any user action this session.
  // A reload therefore renders the correct joined/applied state.
  const [localStatus, setLocalStatus] = useState<LocalStatus>(
    event.myRsvpStatus ?? "",
  );
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Live verification (the R4 acceptance test)**

Run backend + frontend locally (or against the box). As a logged-in user: sign up / apply to an event on `/events/{id}`, then hard-reload. Expected: the CTA shows "Вы записаны" / "Заявка отправлена" (not the join CTA). Log out and reload the same event: back to the join CTA (anonymous → empty status).

- [ ] **Step 4: Commit**

```bash
git add frontend/components/SignupCTA.tsx
git commit -m "fix(events): render join/apply status on reload (R4 frontend)"
```

---

### Task 7: Styled Russian file-upload control (cover + logo)

One reusable component replaces the raw `<input type=file>` ("Choose File / No file chosen") on the event cover and the organizer logo. Keeps the existing `uploadFile` flow and preview.

**Files:**
- Create: `frontend/components/ui/ImageUpload.tsx`
- Modify: `frontend/components/CreateEventForm.tsx:245-275` (cover block)
- Modify: `frontend/app/me/organizer/page.tsx` (logo block ~178-182)

**Interfaces:**
- Consumes: `uploadFile(file) => Promise<{ id: string; url: string }>` from `@/lib/api`.
- Produces: `ImageUpload` — `{ label: string; previewUrl?: string; uploading?: boolean; error?: string; onFile: (file: File) => void; disabled?: boolean }`. Renders a styled RU button ("Загрузить обложку"/"Заменить"), a hidden native input, an optional thumbnail, and progress/error text.

- [ ] **Step 1: Create the component**

```tsx
// frontend/components/ui/ImageUpload.tsx
"use client";

import { useRef } from "react";

/**
 * Styled Russian image upload. Wraps a visually-hidden native file input behind
 * a themed button so the UI never shows the browser's English "Choose File".
 * Shows a thumbnail once a preview URL is available, plus progress/error text.
 * Upload itself is the caller's concern (via onFile → uploadFile).
 */
export function ImageUpload({
  label,
  previewUrl,
  uploading = false,
  error,
  onFile,
  disabled = false,
}: {
  label: string;
  previewUrl?: string;
  uploading?: boolean;
  error?: string;
  onFile: (file: File) => void;
  disabled?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  return (
    <div className="rounded-card bg-bg-secondary p-4">
      {previewUrl && (
        <div className="relative mb-3 aspect-[3/2] w-full overflow-hidden rounded-[10px] bg-fill">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={previewUrl} alt="Предпросмотр" className="h-full w-full object-cover" />
        </div>
      )}
      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        className="sr-only"
        disabled={disabled || uploading}
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) onFile(file);
          e.target.value = ""; // allow re-picking the same file
        }}
      />
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        disabled={disabled || uploading}
        className="rounded-control bg-fill px-4 py-2 text-[15px] font-medium text-label transition hover:bg-fill-secondary disabled:opacity-60"
      >
        {uploading ? "Загрузка…" : previewUrl ? `Заменить ${label.toLowerCase()}` : `Загрузить ${label.toLowerCase()}`}
      </button>
      {error && <p className="mt-2 text-[13px] text-red-500">{error}</p>}
    </div>
  );
}
```

- [ ] **Step 2: Use it for the event cover**

In `frontend/components/CreateEventForm.tsx`, replace the inner `<div className="rounded-card bg-bg-secondary p-4">…</div>` (the block containing the `<input type="file">`, lines ~249-273) with:

```tsx
            <ImageUpload
              label="обложку"
              previewUrl={coverPreviewUrl}
              uploading={coverUploading}
              error={coverError}
              onFile={handleCoverFile}
            />
```

Change `handleCoverChange` (currently takes a `ChangeEvent`) to take a `File` directly:

```tsx
  const handleCoverFile = async (file: File) => {
    setCoverError(undefined);
    setCoverUploading(true);
    try {
      const { id, url } = await uploadFile(file);
      setCoverFileId(id);
      setCoverPreviewUrl(url);
    } catch (err) {
      setCoverError(err instanceof Error ? err.message : "Ошибка загрузки");
    } finally {
      setCoverUploading(false);
    }
  };
```

Add `import { ImageUpload } from "@/components/ui/ImageUpload";` and keep the outer `<label>`'s "Обложка" caption span.

- [ ] **Step 3: Use it for the organizer logo**

In `frontend/app/me/organizer/page.tsx`, replace the `<input type="file" …>` logo block (~lines 179-182, under the "Логотип" label) with an `<ImageUpload label="логотип" … />`, adapting to that page's existing logo state (`logoFileId`, the `uploadFile` call at line ~109). Add a `logoUploading`/`logoError` state pair if not present and a preview URL from the upload result, mirroring Step 2.

- [ ] **Step 4: Typecheck + build + lint**

Run: `cd frontend && npx tsc --noEmit && npm run build && npm run lint`
Expected: no errors (watch for unused `React.ChangeEvent` import after the handler change).

- [ ] **Step 5: Commit**

```bash
git add frontend/components/ui/ImageUpload.tsx frontend/components/CreateEventForm.tsx \
  frontend/app/me/organizer/page.tsx
git commit -m "feat(upload): styled Russian image upload for cover + logo"
```

---

### Task 8: Styled date-time field with explicit МСК label

Replaces the two raw `datetime-local` inputs (English `dd/mm/yyyy`, keyboard-entry friction, no timezone cue). P0 deliverable: a themed, self-contained `DateTimeField` built on split native `date` + `time` inputs with an explicit **(МСК)** label — decisive, accessible, and removes the ambiguity. (A fully custom calendar-popover widget is out of P0 scope.)

**Files:**
- Create: `frontend/components/ui/DateTimeField.tsx`
- Test: `frontend/lib/__tests__/datetime.test.ts` (create — round-trip helpers)
- Modify: `frontend/components/CreateEventForm.tsx:363-368` (Начало/Окончание fields)

**Interfaces:**
- Produces: `splitLocal(value: string): { date: string; time: string }` and `joinLocal(date: string, time: string): string` (both operating on the `"YYYY-MM-DDTHH:mm"` datetime-local string the form already uses via `toDatetimeLocalValue`), and a `DateTimeField` component — `{ value: string; onChange: (v: string) => void; }`.

- [ ] **Step 1: Write the failing round-trip test**

```ts
// frontend/lib/__tests__/datetime.test.ts
import { describe, expect, it } from "vitest";
import { joinLocal, splitLocal } from "@/components/ui/DateTimeField";

describe("datetime-local split/join", () => {
  it("splits a datetime-local string", () => {
    expect(splitLocal("2026-08-15T18:30")).toEqual({ date: "2026-08-15", time: "18:30" });
  });
  it("handles empty", () => {
    expect(splitLocal("")).toEqual({ date: "", time: "" });
  });
  it("joins date + time", () => {
    expect(joinLocal("2026-08-15", "18:30")).toBe("2026-08-15T18:30");
  });
  it("join returns empty when date missing", () => {
    expect(joinLocal("", "18:30")).toBe("");
  });
  it("join defaults a missing time to 00:00 when date present", () => {
    expect(joinLocal("2026-08-15", "")).toBe("2026-08-15T00:00");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run lib/__tests__/datetime.test.ts`
Expected: FAIL — module/exports not found.

- [ ] **Step 3: Create `DateTimeField`**

```tsx
// frontend/components/ui/DateTimeField.tsx
"use client";

// Split/join the native datetime-local string ("YYYY-MM-DDTHH:mm") the create
// form already round-trips via toDatetimeLocalValue. Kept as pure functions so
// they're unit-testable without a DOM.
export function splitLocal(value: string): { date: string; time: string } {
  const [date = "", time = ""] = value.split("T");
  return { date, time };
}
export function joinLocal(date: string, time: string): string {
  if (!date) return "";
  return `${date}T${time || "00:00"}`;
}

const cls =
  "rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none focus:ring-2 focus:ring-accent";

/**
 * Themed date + time entry with an explicit «(Мск)» label. Two native controls
 * (type=date shows a locale calendar; type=time a locale clock) avoid the
 * ambiguous dd/mm/yyyy keyboard friction of a single datetime-local, and the
 * label makes the fixed Moscow zone unmistakable. Emits the same
 * "YYYY-MM-DDTHH:mm" value the form consumes.
 */
export function DateTimeField({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  const { date, time } = splitLocal(value);
  return (
    <div className="flex flex-wrap items-center gap-2">
      <input
        type="date"
        className={cls}
        value={date}
        onChange={(e) => onChange(joinLocal(e.target.value, time))}
      />
      <input
        type="time"
        className={cls}
        value={time}
        onChange={(e) => onChange(joinLocal(date, e.target.value))}
      />
      <span className="text-[13px] text-label-secondary">Мск</span>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --run lib/__tests__/datetime.test.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Wire it into the create/edit form**

In `frontend/components/CreateEventForm.tsx`, replace the two `datetime-local` inputs (lines 363-368) with `Controller`-driven `DateTimeField`s (the form uses react-hook-form; the plain `datetime-local` used `register`, but `DateTimeField` needs `value`/`onChange`):

```tsx
          <Field label="Начало" error={errors.startsAt?.message}>
            <Controller
              control={control}
              name="startsAt"
              render={({ field }) => (
                <DateTimeField value={field.value ?? ""} onChange={field.onChange} />
              )}
            />
          </Field>
          <Field label="Окончание">
            <Controller
              control={control}
              name="endsAt"
              render={({ field }) => (
                <DateTimeField value={field.value ?? ""} onChange={field.onChange} />
              )}
            />
          </Field>
```

Add `import { DateTimeField } from "@/components/ui/DateTimeField";`. `Controller` is already imported. The `onSubmit` conversion `new Date(v.startsAt).toISOString()` still works because `DateTimeField` emits the same `"YYYY-MM-DDTHH:mm"` string.

- [ ] **Step 6: Typecheck + build + edit-prefill check**

Run: `cd frontend && npx tsc --noEmit && npm run build`
Expected: no errors. Manually confirm on `/events/[id]/edit` that a seeded event's Начало pre-fills both the date and time sub-fields (the `initial.startsAt` still flows through `toDatetimeLocalValue`).

- [ ] **Step 7: Commit**

```bash
git add frontend/components/ui/DateTimeField.tsx frontend/lib/__tests__/datetime.test.ts \
  frontend/components/CreateEventForm.tsx
git commit -m "feat(form): themed date+time field with explicit МСК label"
```

---

## Final verification (whole P0)

- [ ] **Frontend green:** `cd frontend && npm test -- --run && npx tsc --noEmit && npm run build && npm run lint` — all pass.
- [ ] **Backend green:** `cd backend && go build ./... && go test ./...` — all pass.
- [ ] **Live smoke** (local or box, per `docs/qa`): (1) create event → defaults to Черновик; (2) quick-publish shows the styled modal, not a native dialog; (3) sign up + reload → status persists (R4); (4) admin revoke twice → committed `rejected`, no red error; (5) a 15–17 Aug event shows the range on feed/detail/list and spans all three calendar days; (6) cover + logo show the Russian upload button; (7) Начало/Окончание show themed date+time + «Мск».
- [ ] **Deploy note:** no new migration in P0. Backend change is code-only (Task 5). Frontend rebuild uses `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru` per the deploy runbook.

## Notes for P1 / P2

Not in this plan (separate plans): owner-view on own event, feed upcoming-first sort, all-8 category filters, venue pre-fill on edit, calendar pending-vs-confirmed, verification explainer, organizer-hub cards, dark map tiles, public org bio/logo (P1); plural `жалоб`, dismiss guard, review pre-gating, terminology consolidation, and the grab-bag polish (P2). See the spec's phase sections.
