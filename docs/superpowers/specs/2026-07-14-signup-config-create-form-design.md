# Signup Configuration in Create-Event Form (R1) — Design

_Design spec · 2026-07-14 · frontend + backend validation (no migration)_

## Problem

The create-event form (`frontend/components/CreateEventForm.tsx`) exposes only content + venue + dates + price + status. It does **not** expose `signup_mode`, `capacity`, `curator_question`, or `external_registration_url`. Every event created through the UI is therefore `signup_mode="open"` with **unlimited capacity**.

The backend fully supports the other modes (`internal/http/formatter/event.go:160-166` maps them at create; `internal/models/event.go:52,83` defines and validates them), but they are reachable only via a direct `POST /api/v1/events`.

**Concrete blocker (from the first-clients QA):** Tarsky's «Тур в Геленджик» is a curated trip **limited to 10 people, by selection**. That is `signup_mode="application"` + `capacity=10` + a curator question — none of which the organizer can set from the product today.

## Goals

- Let an organizer choose the **signup mode** (open / application / external) when creating an event.
- Let them set a **capacity** (seat limit) for open and application modes.
- Let them set a **curator question** (application) and an **external registration URL** (external).
- Harden the backend's create-time validation messages (Russian, human-readable).

## Non-goals

- No DB migration — the columns already exist.
- No change to RSVP runtime behaviour (waitlist/promotion/decision logic is untouched).
- Editing these fields on an already-created event is out of scope here (draft edit already works; published edit incl. capacity is **spec R2**, a separate document).
- No new signup modes beyond the three the backend already supports.

## Decisions (from brainstorming)

| Decision | Choice |
| --- | --- |
| Which modes in UI | **All three** — Открытая / По заявке / Внешняя ссылка |
| Capacity field | Exposed for open + application; empty = unlimited |
| Curator question | Shown + required only for application mode |
| External URL | Shown + required only for external mode |
| Default mode | **open** (unchanged current behaviour) |

## Design

### Frontend — `CreateEventForm.tsx`

Add a **«Запись»** section (after price, before status). Progressive disclosure driven by the selected mode:

- **Segmented control** `signup_mode`: `Открытая запись` (open) · `По заявке` (application) · `Внешняя ссылка` (external). Default `open`.
- **Лимит мест** (`capacity`, integer input) — visible for `open` + `application`. Placeholder/help: «Оставьте пустым — без ограничения». Empty → omit `capacity` (unlimited). Must be `> 0` when set.
- **Вопрос кандидату** (`curator_question`, textarea) — visible + **required** only for `application`. Help: «Покажется в форме заявки».
- **Ссылка для регистрации** (`external_registration_url`, url input) — visible + **required** only for `external`. When external is chosen, hide capacity + curator question (no local RSVP rows exist for external).

Zod schema gains a **discriminated/refined** validation:
- `application` ⇒ `curator_question` non-empty.
- `external` ⇒ `external_registration_url` valid URL.
- `capacity` (when present) ⇒ integer `> 0`.
All messages in Russian.

The submit payload (`lib/api.ts` create call) forwards the new fields; when mode is `open` and capacity empty, the payload matches today's behaviour exactly (backwards compatible).

### Backend — validation hardening only

No handler/route changes (create already maps the fields). Confirm and, where missing, sharpen the create-path checks in `internal/http/formatter/event.go` / `internal/models/event.go:83`:
- `application` without `curator_question` → 422 «Для режима «по заявке» нужен вопрос кандидату».
- `external` without `external_registration_url` → 422 «Укажите ссылку для внешней регистрации».
- `capacity <= 0` → 422 «Лимит мест должен быть больше нуля».
Keep the existing `use_zero` default guard: create formatter must default `signup_mode` to `open` (never `''`) — see the known `events_signup_mode_check` 503 gotcha.

## Data flow

Form (mode + conditional fields) → Zod validate → `POST /api/v1/events` (existing) → formatter maps signup fields → event row created with the chosen mode/capacity. RSVP behaviour then follows the existing `rsvp` domain for that mode.

## Error handling

- Client-side Zod blocks submit with inline Russian messages before the request.
- Server-side 422 messages (above) are surfaced in the form's error area (reuse the existing error rendering that shows the quota 429 copy).

## Testing

- **Unit (frontend):** Zod schema — application requires question; external requires URL; capacity>0; open+empty capacity passes with no `capacity` key.
- **Integration (backend):** `POST /events` with each mode; missing conditional field → 422; capacity 0 → 422; happy application create → event has `signup_mode=application`, `capacity=10`.
- **Manual (the Gelendzhik case):** create an application event, cap 10, with a curator question, entirely from the form; verify it appears with «осталось N мест» and the application flow works end-to-end (ties into RSVP spec, `EventApplicationsPanel`).

## Rollout

Frontend + a possible small backend validation tweak. No migration, schema stays 018. Deploy via the standard build-on-Mac→`save|ssh|load` path.
