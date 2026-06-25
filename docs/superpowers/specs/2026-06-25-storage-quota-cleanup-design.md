# Design — Presence.Tarski: auth finish, storage, event quota, file cleanup

_2026-06-25. Branch `feat/presence-tarski-storage-quota` (off `feat/dark-mode-theme` tip).
Continues the auth slice (spec `2026-06-25-auth-gatekeeper-design.md`) and adds three new
features. Decisions confirmed with the user in brainstorming — see "Decisions" below._

## Goal

Five workstreams, in build order:

0. **Rename** the product display name `Lia` → `Presence.Tarski` everywhere user-facing.
1. **Finish auth** (Phase C): clear the GateGuard `SignInOAuth` panic that blocks
   demo-login, verify the full login → token → `POST /events` round trip, rebuild +
   redeploy live (auth already enforced via `HTTP_MOCK_AUTH=false`).
2. **Storage abstraction** ("S3 analogue"): a local-disk blob store now, swappable to
   any RU-zone S3-compatible provider by config. Authenticated uploads for event cover
   images and user avatars.
3. **Event-creation quota**: ≤10 events per calendar month per user.
4. **Cleanup cron**: a daily job that deletes orphaned (unreferenced) uploaded files.

## Decisions (from brainstorming)

- **Storage scope**: images + other attachments — a generic `files` blob model wired to
  two concrete uses now (event cover image, user avatar); extensible to more later.
- **Quota**: 10 per **calendar month** per user (resets on the 1st), counts published +
  draft, limit configurable via env (default 10).
- **Cleanup**: orphaned uploads only (referenced by no event/user) **and** older than 24h
  grace; runs daily; logs counts.
- **Deploy**: verify the panic fix, keep auth enforced, rebuild + redeploy live; document
  the access-control posture for ISO 27001 / Vanta.
- **Branch**: `feat/presence-tarski-storage-quota` off the `feat/dark-mode-theme` tip
  (carries the committed auth backend + dark mode + the uncommitted frontend auth/rename).
- **S3 backend depth**: code-complete via `minio-go` v7 **and** smoke-tested against a
  local MinIO (put/get/delete) so the swap is proven, not just plausible.

## Workstream 0 — Rename (mechanical)

Product display name becomes **`Presence.Tarski`** in every user-facing place: HTML
`<title>`/metadata, `GlassNav` titles, README prose, web-app manifest/meta, and remaining
source comments that name the product.

**Out of scope (intentionally):** internal identifiers stay `lia` — the Go module
`github.com/Pashteto/lia`, Postgres table/DB names, Docker container names, and the
`lia.pashteto.com` / `api.lia.pashteto.com` domains. These are not user-facing; renaming
them is high-risk churn with no demo value, and `Presence.Tarski` is not a valid Go module
path. The frontend `page.tsx`/`layout.tsx` and home `GlassNav` were already updated in the
in-flight work; this workstream finishes the sweep.

## Workstream 1 — Auth finish (Phase C)

**Current live state:** prod runs `HTTP_MOCK_AUTH=false` → auth is **enforced** (`POST
/events` returns 401 without a token, verified). The only blocker is that
`POST /auth/demo-login` 503s because GateGuard's `SignInOAuth` panics with
`index out of range [2] with length 2`. The latest commit (`87116e0`) applied the
runbook's suspected fix (send explicit `Status=UserActive`, `Role=UserRoleCommon` on the
proto user) but it was **never redeployed/verified on the box**.

**Plan:**
1. Redeploy the backend `app` with the Status+Role fix; curl the repro
   (`POST /auth/demo-login {"email":"x@y.z","name":"X"}`).
2. **If the panic clears:** verify the round trip — `demo-login` → token →
   `POST /events` with `Authorization: Bearer <token>` → **201** (proves `CheckAuth` end
   to end). Then verify the frontend (`pnpm lint`/`build`) and redeploy it.
3. **If it still panics:** apply `systematic-debugging` — get the stack trace by adding
   `debug.Stack()` to GateGuard's gRPC recovery interceptor (or run GateGuard locally
   under grpcurl), localize the index, fix, redeploy. Do not guess further.

Frontend Phase C is already built (`lib/auth.ts`, `lib/auth-context.tsx`,
`components/AuthButton.tsx`, `api.ts` `demoLogin` + Bearer attach, `CreateEventForm`
gate, providers wired) — this workstream verifies and ships it, plus surfaces the new
429 quota error (Workstream 3) in the login/create UI.

## Workstream 2 — Storage abstraction + uploads

### `internal/storage` package

```go
type Storage interface {
    Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    URL(key string) string
    Exists(ctx context.Context, key string) (bool, error)
}
```

- **`localStorage`** (default, demo): writes blobs under `STORAGE_LOCAL_DIR`
  (e.g. `/data/uploads`, a Docker volume). `URL(key)` returns a backend-served route
  `<PUBLIC_BASE>/api/v1/files/{key}`; a `GET /api/v1/files/{key}` handler streams the
  object via `Get` (open read — public read of public event imagery, no PII).
- **`s3Storage`** (config-gated, RU-swappable): a `minio-go` v7 client with a custom
  endpoint. Works unchanged against Yandex Object Storage (`storage.yandexcloud.net`),
  Selectel (`s3.ru-1.storage.selcloud.ru`), VK Cloud, Cloud.ru/SberCloud, or self-hosted
  MinIO — all S3-API-compatible with `region=us-east-1`. `URL(key)` returns the object's
  public/presigned URL. Selected when `STORAGE_BACKEND=s3`.

**Config** (`config` package, all env): `STORAGE_BACKEND` (`local`|`s3`, default `local`),
`STORAGE_LOCAL_DIR`, `STORAGE_PUBLIC_BASE`; for S3 — `S3_ENDPOINT`, `S3_REGION`,
`S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_USE_SSL`. **Secrets via env only**
(never committed; in prod they live in `.env.prod`, chmod 600, git-ignored — consistent
with the existing demo exception to the Secrets-Manager standard).

### `files` table + uploads

Migration adds:
```
files(
  id            uuid pk,
  storage_key   text not null,
  content_type  text not null,
  size          bigint not null,
  owner_user_id uuid not null,          -- the authenticated uploader
  created_at    timestamptz not null default now()
)
```
- `POST /api/v1/uploads` (multipart `file`): **auth-required** (behind `CheckAuth`).
  Validates MIME against an **image allowlist** (`image/png`, `image/jpeg`, `image/webp`)
  and a **5 MB** size cap. Generates `storage_key = uploads/<uuid>.<ext>`, stores the
  blob via `Storage.Put`, inserts a `files` row owned by the principal, returns
  `{ "id": <uuid>, "url": <Storage.URL(key)> }`. This is the only write surface — no open
  upload, no arbitrary content types.
- **References:** `events.cover_file_id uuid` (zero-uuid = none, `NOT NULL DEFAULT`,
  matching the project's no-NULL-uuid convention) and `users.avatar_file_id uuid`
  (same). On serialization, the API computes `cover_url` / `avatar_url` from the
  referenced file's `storage_key` via `Storage.URL`. Clients send `cover_file_id` /
  `avatar_file_id` (the upload id), never raw URLs.

## Workstream 3 — Event-creation quota

In the `POST /events` service path, after the principal user resolves (organizer =
principal, per the auth slice):

```
count = SELECT count(*) FROM events
        WHERE organizer_id = <principal>
          AND created_at >= <start of current calendar month in Europe/Moscow>
if count >= EVENTS_MONTHLY_LIMIT:   // env, default 10
    return 429 Too Many Requests
```

- Counts **published + draft** (calendar-month semantics chosen for explainability).
- Month boundary computed in **Europe/Moscow** to match the app's pinned timezone
  (consistent with `formatEventDate`).
- 429 body carries a clear Russian message (e.g. «Достигнут лимит: 10 событий в месяц.
  Лимит обновится 1-го числа.»). Frontend surfaces it on the create form (reuses the
  existing mutation error path).
- Enforced server-side (the source of truth). The frontend may optionally show remaining
  quota later — not required for this slice (YAGNI).

## Workstream 4 — Cleanup cron

A **daily in-process ticker** started in the `serve` command (guarded by
`FILE_CLEANUP_ENABLED`, default `true`; interval configurable via
`FILE_CLEANUP_INTERVAL`, default `24h`), **plus** a `lia files:cleanup` CLI subcommand so
ops can run it manually or from OS cron.

**Predicate (narrow, by design):** delete a `files` row + its blob iff
- it is referenced by **no** `events.cover_file_id` and **no** `users.avatar_file_id`, AND
- `created_at < now() - 24h` (grace, so a file uploaded mid-form isn't reaped before the
  event/avatar that will reference it is saved).

Each run logs the candidate count and the deleted count (audit-aware; deletions are
logged). Blob delete then row delete; idempotent and safe to re-run. Blast radius =
only unreferenced demo uploads older than a day — never a referenced image, never event
or user rows.

## Architecture / data flow

```
upload:   client → POST /uploads (auth) → validate(mime,size) → Storage.Put → files row → {id,url}
attach:   client → POST/PATCH /events {cover_file_id} (auth, quota) → events.cover_file_id
serve:    client ← GET /events/{id} {cover_url=Storage.URL(key)} ; img ← GET /files/{key} → Storage.Get
cleanup:  ticker/CLI → files unreferenced & >24h → Storage.Delete + delete row (logged)
swap:     STORAGE_BACKEND=s3 + S3_* env → same Storage interface, minio-go to RU provider
```

Each unit is independently testable: `Storage` via an in-memory/temp-dir fake; the
uploads handler via a fake `Storage`; the quota check via a repo count seam; the cleanup
job via a fake `Storage` + seeded `files`/`events`.

## Error handling

- Upload: 401 (no auth), 415 (bad MIME), 413 (too large), 500 (storage error → logged, no
  secret leakage).
- Quota: 429 with the message above; never silently drops a create.
- Cleanup: per-file errors logged and skipped (one bad blob doesn't abort the run).
- S3: dial/credential errors surface at startup (fail fast) when `STORAGE_BACKEND=s3`.

## Testing

- Go: `Storage` local impl (temp dir), uploads handler (fake Storage; MIME/size limits),
  quota boundary (9→ok, 10→ok, 11→429; month rollover), cleanup predicate (referenced kept,
  orphan>24h deleted, orphan<24h kept). `go build/vet/test ./...` + golangci-lint v1.
- S3: smoke test against a local MinIO container — put/get/delete/URL round trip.
- Frontend: `pnpm lint`/`build`; cover-image upload + quota-429 surfaced; auth gate.
- Live: demo-login round trip (Workstream 1) and a create-with-cover end to end.

## Compliance / security notes (ISO 27001 / Vanta)

- **Access control:** uploads and event/avatar writes require a valid GateGuard token;
  auth is enforced in prod (`HTTP_MOCK_AUTH=false`). Document the enforced posture +
  the demo-login non-production control (mints a token for any email — gated, never real
  prod, like the old `MockAuth`).
- **Upload surface:** authenticated, MIME-allowlisted, size-capped — not an open write.
- **Destructive cron:** narrow predicate + 24h grace + logged; blast radius stated above.
- **Secrets:** S3 credentials via env only; never committed, never echoed in logs.
- **Data handling:** stored blobs are public event imagery / avatars — no PII beyond a
  user-chosen avatar; no customer data. Local blobs stay on the box; S3 swap targets
  RU-zone providers (data-residency friendly).

## Out of scope (YAGNI)

- Renaming internal identifiers / module path / domains.
- Real Google OAuth (later upgrade; needs Google creds).
- RSVP, AI-search (separate slices).
- Image resizing/thumbnails, multi-image galleries, presigned client-direct uploads.
- Per-user quota UI / remaining-count display.
