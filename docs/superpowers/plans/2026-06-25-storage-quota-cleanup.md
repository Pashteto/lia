# Presence.Tarski — Storage + Quota + Cleanup + Auth Finish — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the auth slice and ship it live, rename the product to Presence.Tarski, add a swappable blob-storage layer with authenticated image uploads (event covers + user avatars), cap event creation at 10/calendar-month/user, and reap orphaned uploads daily.

**Architecture:** A new `internal/storage` package defines a `Storage` interface with a local-disk impl (default) and a `minio-go` S3 impl (config-gated, swappable to any RU-zone S3 provider). A new `internal/files` domain tracks every uploaded blob in a `files` table. Plain `net/http` handlers (mounted ahead of the go-swagger mux) serve upload/download — the two cases go-swagger handles poorly. Event/avatar references and the quota check ride the existing swagger `POST /events` path. A daily in-process `cleanup` module + a `lia files:cleanup` CLI reap unreferenced blobs older than 24h.

**Tech Stack:** Go 1.x modular monolith (`github.com/Pashteto/lia`), go-pg v10, gofrs/uuid, go-swagger generated server, cobra CLI, `minio-go/v7` (new dep), PostgreSQL+PostGIS. Frontend: Next.js App Router + TS + Tailwind + pnpm.

## Global Constraints

- **Module path stays `github.com/Pashteto/lia`.** Never rename it; "Presence.Tarski" is a display name only.
- **Internal identifiers stay `lia`** — table/DB names, container names, `lia.pashteto.com` domain. Rename only user-facing copy.
- **No-NULL-uuid rule:** uuid columns are `NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'`; zero-uuid (`uuid.Nil`) means "unset". Never scan SQL `NULL` into `uuid.UUID`. Avoid `Returning("*")` on inserts with optional uuid columns.
- **go-pg raw `Query` skips hooks:** if you scan events via raw SQL, call `e.AfterSelect(ctx)` per row so `Status` is populated.
- **swagger codegen:** after editing `backend/api/swagger.yaml`, run `make generate-all` (gitignored generated code) before `go build`. Generated server lives under `internal/http/server/`.
- **golangci-lint is v1** (`.golangci.yml` is v1 format) — do not migrate to v2.
- **Secrets via env only.** S3 creds (`S3_ACCESS_KEY`/`S3_SECRET_KEY`) never committed, never logged. Prod uses `.env.prod` (chmod 600, git-ignored).
- **Box (vds-ru215) is flaky** (1.9 GB RAM, broken IPv6). Run long Docker builds detached + poll. Force IPv4 in any Dockerfile that `curl`s github.
- **Auth is an access-control change** — document the enforced posture for ISO 27001 / Vanta in the HANDOFF when flipping/verifying.
- **TDD:** failing test → run-fail → minimal impl → run-pass → commit. Commit messages end with the Co-Authored-By trailer for Claude Opus 4.8.

---

## Phase 0 — Rename to Presence.Tarski

### Task 0.1: Finish the user-facing rename sweep

**Files:**
- Modify: `frontend/app/globals.css:4` (comment), `frontend/lib/types.ts:1,66` (comments)
- Modify: `README.md` (prose occurrences of "Lia" as product name)
- Verify: `frontend/app/layout.tsx` (`<title>` already "Presence.Tarski — События"), `frontend/app/page.tsx` (GlassNav `title="Presence.Tarski"` already done)
- Check: `frontend/app/manifest.ts` / any `apple-web-app`/`og:` metadata (create-only if present)

**Interfaces:** none (copy-only).

- [ ] **Step 1: Enumerate remaining product-name occurrences**

Run:
```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
grep -rn "Lia" frontend README.md --include="*.tsx" --include="*.ts" --include="*.css" --include="*.md" 2>/dev/null | grep -v "node_modules\|\.next" | grep -vi "LiaEvent\|apiEventToLia\|LiaAPIAPI"
```
Expected: matches in `globals.css:4`, `types.ts:1,66`, and README prose. (Code identifiers `LiaEvent`/`apiEventToLia` are NOT renamed — they're internal types.)

- [ ] **Step 2: Replace product-name occurrences**

Edit each match so the *product name* reads "Presence.Tarski" while keeping identifiers. Examples:
- `globals.css:4`: `* Presence.Tarski design tokens — Apple HIG.`
- `types.ts:1`: `// Domain types for the Presence.Tarski frontend. These mirror the backend domain model`
- `types.ts:66`: `/** Shape returned by the backend GET /api/v1/events (Presence.Tarski API Event model). */`
- README: replace "Lia" product references with "Presence.Tarski"; keep code/paths/module/domain as-is.

- [ ] **Step 3: Verify no user-facing "Lia" remains (identifiers excepted)**

Run the Step 1 grep again. Expected: only `LiaEvent`/`apiEventToLia`/`LiaAPIAPI` identifiers (acceptable) remain; no prose/title "Lia".

- [ ] **Step 4: Commit**

```bash
git add frontend/app/globals.css frontend/lib/types.ts README.md
git commit -m "chore(rename): finish Lia → Presence.Tarski user-facing sweep

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase 1 — Finish auth (verify panic fix, ship frontend)

> This phase is verification/ops-heavy, not TDD. The backend signer fix is committed (`87116e0`); the goal is to confirm it clears the GateGuard `index out of range [2]` panic on the box, then verify and ship the already-built frontend.

### Task 1.1: Commit the in-flight frontend auth work

**Files (already in working tree, uncommitted):**
- `frontend/lib/auth.ts`, `frontend/lib/auth-context.tsx`, `frontend/components/AuthButton.tsx` (new)
- `frontend/lib/api.ts`, `frontend/app/providers.tsx`, `frontend/app/page.tsx`, `frontend/components/CreateEventForm.tsx` (modified)

- [ ] **Step 1: Lint + build the frontend**

Run:
```bash
cd frontend && pnpm install && pnpm lint && pnpm build
```
Expected: lint clean, build succeeds. Fix any type errors before committing.

- [ ] **Step 2: Commit the frontend auth Phase C**

```bash
cd /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia
git add frontend/lib/auth.ts frontend/lib/auth-context.tsx frontend/components/AuthButton.tsx \
        frontend/lib/api.ts frontend/app/providers.tsx frontend/app/page.tsx frontend/components/CreateEventForm.tsx
git commit -m "feat(auth): frontend demo-login (Войти modal, Bearer attach, create-event gate)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 1.2: Verify the GateGuard panic fix on the box

**Files:** none (deploy + verify). Reference: `docs/superpowers/runbooks/2026-06-25-gateguard-signin-panic-HANDOFF.md`.

- [ ] **Step 1: Redeploy the backend `app` with the committed signer fix**

The fix (`backend/internal/http/auth/signer.go`, sets `Status: UserActive`, `Role: UserRoleCommon`) is committed but unverified on the box. Rsync backend + rebuild:
```bash
ssh vdska2 'cd /opt/lia/backend && docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.gateguard.yml up -d --build app'
```
(If rsync of the source is needed first, follow the deploy runbook `2026-06-23-vds-ru215-deploy.md`. Run detached + poll — the box is flaky.)

- [ ] **Step 2: Reproduce the demo-login call**

Run:
```bash
curl -s -X POST https://api.lia.pashteto.com/api/v1/auth/demo-login \
  -H 'Content-Type: application/json' -d '{"email":"demo@presence.test","name":"Demo"}'
```
Expected (fixed): `{"token":"<jwt>"}` (HTTP 200). If still `{"code":503,...}` → go to Task 1.3.

- [ ] **Step 3: Verify the full auth round trip**

```bash
TOKEN=$(curl -s -X POST https://api.lia.pashteto.com/api/v1/auth/demo-login \
  -H 'Content-Type: application/json' -d '{"email":"demo@presence.test","name":"Demo"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
# anon → 401
curl -s -o /dev/null -w "%{http_code}\n" -X POST https://api.lia.pashteto.com/api/v1/events \
  -H 'Content-Type: application/json' -d '{"title":"t","starts_at":"2026-09-01T18:00:00Z"}'
# authed → 201
curl -s -o /dev/null -w "%{http_code}\n" -X POST https://api.lia.pashteto.com/api/v1/events \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"title":"auth roundtrip ok","status":"draft","starts_at":"2026-09-01T18:00:00Z"}'
```
Expected: `401` then `201`. This proves `CheckAuth` + signer end to end.

### Task 1.3: (Only if Step 2 still 503s) Debug the panic with a stack trace

> Use superpowers:systematic-debugging. Do NOT guess further fixes.

- [ ] **Step 1: Add `debug.Stack()` to GateGuard's gRPC recovery interceptor**

On the box, locate GateGuard source (`~/gateway_fm/appstore/gateguard` or `/opt/gateguard`). In its gRPC recovery interceptor, log `string(debug.Stack())` on panic. Rebuild GateGuard, repro the curl from Task 1.2 Step 2, read the trace from `docker logs backend-gateguard-1`.

- [ ] **Step 2: Localize the `index out of range [2]` and fix at the source**

The trace names the file:line. Likely suspects (per HANDOFF): redis storage key path (`go-redis` v6 `REDIS_ADDRESS` parsing), a logging/sentry interceptor, or `repository.CreateUser` column building. Fix the indexing bug there, rebuild, re-verify Task 1.2 Steps 2–3.

- [ ] **Step 3: Record findings in the panic runbook**

Append root cause + fix to `docs/superpowers/runbooks/2026-06-25-gateguard-signin-panic-HANDOFF.md`. Commit (docs only).

---

## Phase 2 — Storage abstraction (`internal/storage`)

### Task 2.1: Define the `Storage` interface + local-disk implementation

**Files:**
- Create: `backend/internal/storage/storage.go` (interface + errors)
- Create: `backend/internal/storage/local.go` (local-disk impl)
- Test: `backend/internal/storage/local_test.go`

**Interfaces:**
- Produces:
  ```go
  package storage
  type Storage interface {
      Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
      Get(ctx context.Context, key string) (io.ReadCloser, error)
      Delete(ctx context.Context, key string) error
      URL(key string) string
      Exists(ctx context.Context, key string) (bool, error)
  }
  var ErrNotFound = errors.New("storage: object not found")
  func NewLocal(baseDir, publicBase string) (Storage, error) // publicBase e.g. "https://api.lia.pashteto.com/api/v1/files"
  ```

- [ ] **Step 1: Write the failing test**

`backend/internal/storage/local_test.go`:
```go
package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestLocal_PutGetExistsDelete(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(dir, "https://x/api/v1/files")
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	ctx := context.Background()
	body := []byte("hello-image-bytes")
	if err := s.Put(ctx, "uploads/a.png", bytes.NewReader(body), int64(len(body)), "image/png"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	ok, err := s.Exists(ctx, "uploads/a.png")
	if err != nil || !ok {
		t.Fatalf("Exists: ok=%v err=%v", ok, err)
	}
	rc, err := s.Get(ctx, "uploads/a.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, body) {
		t.Fatalf("Get body mismatch: %q", got)
	}
	if s.URL("uploads/a.png") != "https://x/api/v1/files/uploads/a.png" {
		t.Fatalf("URL: %q", s.URL("uploads/a.png"))
	}
	if err := s.Delete(ctx, "uploads/a.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := s.Exists(ctx, "uploads/a.png"); ok {
		t.Fatalf("Exists after delete = true")
	}
}

func TestLocal_GetMissing_ReturnsErrNotFound(t *testing.T) {
	s, _ := NewLocal(t.TempDir(), "https://x/f")
	if _, err := s.Get(context.Background(), "nope.png"); err == nil {
		t.Fatal("expected error for missing object")
	}
}

func TestLocal_RejectsPathTraversal(t *testing.T) {
	s, _ := NewLocal(t.TempDir(), "https://x/f")
	if err := s.Put(context.Background(), "../escape.png", nil, 0, "image/png"); err == nil {
		t.Fatal("expected path-traversal rejection")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/storage/...`
Expected: FAIL (package/functions not defined).

- [ ] **Step 3: Implement interface + local impl**

`backend/internal/storage/storage.go`:
```go
// Package storage is a swappable blob store. The local impl backs the demo;
// the s3 impl (config-gated) targets any S3-compatible RU-zone provider.
package storage

import (
	"context"
	"errors"
	"io"
)

// ErrNotFound is returned by Get/Exists when an object is absent.
var ErrNotFound = errors.New("storage: object not found")

// Storage is a content-addressed-by-key blob store.
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	// URL returns a publicly fetchable URL for key.
	URL(key string) string
	Exists(ctx context.Context, key string) (bool, error)
}
```

`backend/internal/storage/local.go`:
```go
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type local struct {
	baseDir    string
	publicBase string // no trailing slash
}

// NewLocal returns a Storage that writes blobs under baseDir and serves them
// at publicBase/<key> (a backend route — see the files HTTP handler).
func NewLocal(baseDir, publicBase string) (Storage, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("storage: baseDir is required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("storage: mkdir %q: %w", baseDir, err)
	}
	return &local{baseDir: baseDir, publicBase: strings.TrimRight(publicBase, "/")}, nil
}

// resolve guards against path traversal and returns the on-disk path for key.
func (l *local) resolve(key string) (string, error) {
	clean := filepath.Clean("/" + key) // force absolute, collapses .. 
	p := filepath.Join(l.baseDir, clean)
	if !strings.HasPrefix(p, filepath.Clean(l.baseDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return p, nil
}

func (l *local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	p, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("storage: mkdir: %w", err)
	}
	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("storage: create: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("storage: write: %w", err)
	}
	return nil
}

func (l *local) Get(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := l.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	return f, nil
}

func (l *local) Delete(_ context.Context, key string) error {
	p, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: remove: %w", err)
	}
	return nil
}

func (l *local) Exists(_ context.Context, key string) (bool, error) {
	p, err := l.resolve(key)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *local) URL(key string) string {
	return l.publicBase + "/" + strings.TrimLeft(key, "/")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/storage/...`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/storage/storage.go backend/internal/storage/local.go backend/internal/storage/local_test.go
git commit -m "feat(storage): Storage interface + local-disk impl

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 2.2: Add the `minio-go` S3 implementation (config-gated)

**Files:**
- Create: `backend/internal/storage/s3.go`
- Modify: `backend/go.mod` / `backend/go.sum` (add `github.com/minio/minio-go/v7`)
- Test (smoke, build-tagged): `backend/internal/storage/s3_smoke_test.go`

**Interfaces:**
- Produces:
  ```go
  type S3Config struct {
      Endpoint, Region, Bucket, AccessKey, SecretKey, PublicBase string
      UseSSL bool
  }
  func NewS3(cfg S3Config) (Storage, error)
  ```

- [ ] **Step 1: Add the dependency**

Run: `cd backend && go get github.com/minio/minio-go/v7@latest`
Expected: go.mod/go.sum updated.

- [ ] **Step 2: Write the S3 impl**

`backend/internal/storage/s3.go`:
```go
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config configures an S3-compatible backend. Works against AWS S3, MinIO,
// and RU-zone providers (Yandex Object Storage, Selectel, VK Cloud, Cloud.ru):
// set Endpoint to the provider host and Region to "us-east-1".
type S3Config struct {
	Endpoint   string
	Region     string
	Bucket     string
	AccessKey  string
	SecretKey  string
	PublicBase string // optional; if empty, URL() derives endpoint/bucket/key
	UseSSL     bool
}

type s3store struct {
	client     *minio.Client
	bucket     string
	publicBase string
}

// NewS3 dials the S3-compatible endpoint and verifies the bucket exists.
func NewS3(cfg S3Config) (Storage, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, fmt.Errorf("storage(s3): endpoint and bucket are required")
	}
	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("storage(s3): new client: %w", err)
	}
	ok, err := cli.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage(s3): bucket check: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("storage(s3): bucket %q does not exist", cfg.Bucket)
	}
	base := strings.TrimRight(cfg.PublicBase, "/")
	if base == "" {
		scheme := "https"
		if !cfg.UseSSL {
			scheme = "http"
		}
		base = fmt.Sprintf("%s://%s/%s", scheme, cfg.Endpoint, cfg.Bucket)
	}
	return &s3store{client: cli, bucket: cfg.Bucket, publicBase: base}, nil
}

func (s *s3store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("storage(s3): put %q: %w", key, err)
	}
	return nil
}

func (s *s3store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage(s3): get %q: %w", key, err)
	}
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		var e minio.ErrorResponse
		if errors.As(err, &e) && e.Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("storage(s3): stat %q: %w", key, err)
	}
	return obj, nil
}

func (s *s3store) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("storage(s3): delete %q: %w", key, err)
	}
	return nil
}

func (s *s3store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		var e minio.ErrorResponse
		if errors.As(err, &e) && e.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("storage(s3): stat %q: %w", key, err)
	}
	return true, nil
}

func (s *s3store) URL(key string) string {
	return s.publicBase + "/" + strings.TrimLeft(key, "/")
}
```

- [ ] **Step 3: Write a build-tagged smoke test (run against local MinIO)**

`backend/internal/storage/s3_smoke_test.go`:
```go
//go:build s3smoke

package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

// Run with a local MinIO:
//   docker run -d -p 9000:9000 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin minio/minio server /data
//   docker run --rm --network host minio/mc alias set m http://127.0.0.1:9000 minioadmin minioadmin && mc mb m/testbucket
//   S3_ENDPOINT=127.0.0.1:9000 go test -tags s3smoke ./internal/storage/ -run TestS3Smoke -v
func TestS3Smoke(t *testing.T) {
	ep := os.Getenv("S3_ENDPOINT")
	if ep == "" {
		t.Skip("S3_ENDPOINT not set")
	}
	s, err := NewS3(S3Config{
		Endpoint: ep, Region: "us-east-1", Bucket: "testbucket",
		AccessKey: "minioadmin", SecretKey: "minioadmin", UseSSL: false,
	})
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}
	ctx := context.Background()
	body := []byte("s3-smoke")
	if err := s.Put(ctx, "uploads/smoke.png", bytes.NewReader(body), int64(len(body)), "image/png"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, err := s.Get(ctx, "uploads/smoke.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, body) {
		t.Fatalf("body mismatch")
	}
	if err := s.Delete(ctx, "uploads/smoke.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := s.Exists(ctx, "uploads/smoke.png"); ok {
		t.Fatal("exists after delete")
	}
}
```

- [ ] **Step 4: Run the regular build + the smoke test against MinIO**

```bash
cd backend && go build ./... && go vet ./internal/storage/...
# stand up MinIO + bucket (commands in the test comment), then:
S3_ENDPOINT=127.0.0.1:9000 go test -tags s3smoke ./internal/storage/ -run TestS3Smoke -v
```
Expected: build ok; smoke test PASS (proves the swap works end to end).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/storage/s3.go backend/internal/storage/s3_smoke_test.go backend/go.mod backend/go.sum
git commit -m "feat(storage): minio-go S3 backend (RU-zone swappable), MinIO smoke test

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 2.3: Add storage config + factory

**Files:**
- Modify: `backend/config/scheme.go` (add `StorageConfig` + field on `Scheme`)
- Modify: `backend/config/init.go` (defaults/env binding — follow existing pattern)
- Create: `backend/internal/storage/factory.go`
- Test: `backend/internal/storage/factory_test.go`

**Interfaces:**
- Produces:
  ```go
  // config
  type StorageConfig struct {
      Backend    string `mapstructure:"backend"`     // "local" | "s3"
      LocalDir   string `mapstructure:"local_dir"`
      PublicBase string `mapstructure:"public_base"`
      S3         *S3StorageConfig `mapstructure:"s3"`
  }
  type S3StorageConfig struct {
      Endpoint, Region, Bucket, AccessKey, SecretKey string `mapstructure:"..."`
      UseSSL bool `mapstructure:"use_ssl"`
  }
  // storage
  func New(cfg StorageSettings) (Storage, error) // StorageSettings is a plain struct mirroring config, to avoid an import cycle
  ```

- [ ] **Step 1: Write the failing factory test**

`backend/internal/storage/factory_test.go`:
```go
package storage

import "testing"

func TestNew_LocalBackend(t *testing.T) {
	s, err := New(StorageSettings{Backend: "local", LocalDir: t.TempDir(), PublicBase: "https://x/f"})
	if err != nil || s == nil {
		t.Fatalf("New local: s=%v err=%v", s, err)
	}
}

func TestNew_UnknownBackend(t *testing.T) {
	if _, err := New(StorageSettings{Backend: "weird"}); err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestNew_DefaultsToLocal(t *testing.T) {
	if _, err := New(StorageSettings{Backend: "", LocalDir: t.TempDir(), PublicBase: "https://x/f"}); err != nil {
		t.Fatalf("empty backend should default to local: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `cd backend && go test ./internal/storage/ -run TestNew`
Expected: FAIL (`New`/`StorageSettings` undefined).

- [ ] **Step 3: Implement the factory**

`backend/internal/storage/factory.go`:
```go
package storage

import "fmt"

// StorageSettings is a transport-agnostic snapshot of storage config. The config
// package converts its StorageConfig into this to avoid a config→storage import cycle.
type StorageSettings struct {
	Backend    string
	LocalDir   string
	PublicBase string
	S3         S3Config
}

// New builds the configured Storage. Backend "" or "local" → local disk; "s3" → minio-go.
func New(cfg StorageSettings) (Storage, error) {
	switch cfg.Backend {
	case "", "local":
		return NewLocal(cfg.LocalDir, cfg.PublicBase)
	case "s3":
		s := cfg.S3
		if s.PublicBase == "" {
			s.PublicBase = cfg.PublicBase
		}
		return NewS3(s)
	default:
		return nil, fmt.Errorf("storage: unknown backend %q", cfg.Backend)
	}
}
```

- [ ] **Step 4: Add config struct + defaults**

In `backend/config/scheme.go` add `StorageConfig`/`S3StorageConfig` structs and a `Storage *StorageConfig` field on `Scheme`. In `backend/config/init.go` set viper defaults following the existing pattern: `storage.backend=local`, `storage.local_dir=/data/uploads`, `storage.public_base` (derive from HTTP or env), and bind `S3_*` envs. (Match the file's existing default/env-binding style — read it first.)

- [ ] **Step 5: Run tests + build**

Run: `cd backend && go test ./internal/storage/... && go build ./...`
Expected: PASS + build ok.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/storage/factory.go backend/internal/storage/factory_test.go backend/config/scheme.go backend/config/init.go
git commit -m "feat(storage): config + backend factory (local default, s3 opt-in)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase 3 — Files domain + upload/serve handlers

### Task 3.1: `files` table migration

**Files:**
- Create: `backend/db/migrations/000010_files_table.up.sql`
- Create: `backend/db/migrations/000010_files_table.down.sql`

- [ ] **Step 1: Write the migration**

`000010_files_table.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS files (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    storage_key   text NOT NULL UNIQUE,
    content_type  text NOT NULL,
    size          bigint NOT NULL DEFAULT 0,
    owner_user_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_files_owner ON files (owner_user_id);
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files (created_at);
```

`000010_files_table.down.sql`:
```sql
DROP TABLE IF EXISTS files;
```

- [ ] **Step 2: Apply + verify**

Run (against local compose Postgres or host workaround per HANDOFF):
```bash
cd backend && docker compose up -d --build  # runs migrate
docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "\d files"
```
Expected: `files` table with the columns above.

- [ ] **Step 3: Commit**

```bash
git add backend/db/migrations/000010_files_table.up.sql backend/db/migrations/000010_files_table.down.sql
git commit -m "feat(files): migration 000010 — files table

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 3.2: `internal/files` domain (model, repository, service)

**Files:**
- Create: `backend/internal/models/file.go`
- Create: `backend/internal/files/repository.go`
- Create: `backend/internal/files/service.go`
- Test: `backend/internal/files/service_test.go`

**Interfaces:**
- Produces:
  ```go
  // models
  type File struct {
      ID uuid.UUID; StorageKey, ContentType string; Size int64; OwnerUserID uuid.UUID; CreatedAt time.Time
  }
  // files
  type Repository interface {
      Create(f *models.File) error
      GetByID(id uuid.UUID) (*models.File, error)
      ListOrphansOlderThan(d time.Duration) ([]*models.File, error) // unreferenced by events.cover_file_id / users.avatar_file_id
      Delete(id uuid.UUID) error
  }
  type Service interface {
      Register(ctx, key, contentType string, size int64, owner uuid.UUID) (*models.File, error)
      Get(ctx, id uuid.UUID) (*models.File, error)
  }
  func NewService(repo Repository, store storage.Storage) Service
  func NewRepository(db *pg.DB) Repository
  var ErrNotFound = errors.New("file not found")
  ```

- [ ] **Step 1: Write the failing service test (with a fake repo + fake storage)**

`backend/internal/files/service_test.go`:
```go
package files

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/Pashteto/lia/internal/models"
)

type fakeRepo struct{ created *models.File }

func (f *fakeRepo) Create(file *models.File) error { f.created = file; return nil }
func (f *fakeRepo) GetByID(id uuid.UUID) (*models.File, error) {
	if f.created != nil && f.created.ID == id {
		return f.created, nil
	}
	return nil, ErrNotFound
}
func (f *fakeRepo) ListOrphansOlderThan(_ interface{ Hours() float64 }) ([]*models.File, error) { return nil, nil } // not used here

func TestRegister_PersistsAndReturnsFile(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)
	owner := uuid.Must(uuid.NewV4())
	f, err := svc.Register(context.Background(), "uploads/a.png", "image/png", 123, owner)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if f.StorageKey != "uploads/a.png" || f.OwnerUserID != owner || f.Size != 123 {
		t.Fatalf("unexpected file: %+v", f)
	}
	if repo.created == nil {
		t.Fatal("repo.Create not called")
	}
}
```
(Adjust the fake's `ListOrphansOlderThan` signature to match the final interface — use `time.Duration`. Keep the fake minimal.)

- [ ] **Step 2: Run to verify fail**

Run: `cd backend && go test ./internal/files/...`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement model, repository, service**

`backend/internal/models/file.go`:
```go
package models

import (
	"time"
	"github.com/gofrs/uuid"
)

// File is an uploaded blob's metadata row (the bytes live in storage.Storage).
type File struct {
	tableName   struct{}  `pg:"files,discard_unknown_columns"` //nolint:unused
	ID          uuid.UUID `pg:"id,pk,type:uuid"`
	StorageKey  string    `pg:"storage_key,notnull"`
	ContentType string    `pg:"content_type,notnull"`
	Size        int64     `pg:"size,use_zero"`
	OwnerUserID uuid.UUID `pg:"owner_user_id,type:uuid,use_zero"`
	CreatedAt   time.Time `pg:"created_at,notnull,default:now()"`
}
```

`backend/internal/files/repository.go` — go-pg impl. `ListOrphansOlderThan(d time.Duration)`:
```go
// orphans: files referenced by no event cover and no user avatar, older than the grace window.
const orphanQuery = `
SELECT f.* FROM files f
WHERE f.created_at < (now() - ?0::interval)
  AND NOT EXISTS (SELECT 1 FROM events e WHERE e.cover_file_id = f.id)
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.avatar_file_id = f.id)
`
```
(Bind the interval as e.g. `fmt.Sprintf("%d seconds", int(d.Seconds()))`. `Create` sets `ID = uuid.NewV4()` if nil and inserts without `Returning("*")`.)

`backend/internal/files/service.go`:
```go
package files

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/storage"
)

var ErrNotFound = errors.New("file not found")

type Service interface {
	Register(ctx context.Context, key, contentType string, size int64, owner uuid.UUID) (*models.File, error)
	Get(ctx context.Context, id uuid.UUID) (*models.File, error)
}

type service struct {
	repo  Repository
	store storage.Storage
}

func NewService(repo Repository, store storage.Storage) Service { return &service{repo: repo, store: store} }

func (s *service) Register(_ context.Context, key, ct string, size int64, owner uuid.UUID) (*models.File, error) {
	f := &models.File{ID: uuid.Must(uuid.NewV4()), StorageKey: key, ContentType: ct, Size: size, OwnerUserID: owner}
	if err := s.repo.Create(f); err != nil {
		return nil, fmt.Errorf("register file: %w", err)
	}
	return f, nil
}

func (s *service) Get(_ context.Context, id uuid.UUID) (*models.File, error) {
	return s.repo.GetByID(id)
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `cd backend && go test ./internal/files/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/models/file.go backend/internal/files/
git commit -m "feat(files): domain model + repository + service

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 3.3: Upload + serve HTTP handlers (plain net/http, mounted ahead of swagger)

**Files:**
- Create: `backend/internal/http/uploads/handler.go` (upload + serve + bearer-auth middleware)
- Test: `backend/internal/http/uploads/handler_test.go`
- Modify: `backend/internal/http/module.go` (inject storage+files+auth; mount in alice chain)

**Interfaces:**
- Consumes: `storage.Storage`, `files.Service`, and an auth principal extractor. Reuse `auth.Auth.CheckAuth(token string) (*apimodels.User, error)` (the existing JwtAuth func) to resolve the bearer.
- Produces:
  ```go
  func NewHandler(store storage.Storage, files files.Service, authFn func(token string) (*apimodels.User, error)) http.Handler
  // Routes (relative to mount at /api/v1):
  //   POST /api/v1/uploads     multipart field "file"; auth required → 201 {"id","url"}
  //   GET  /api/v1/files/{key} stream blob (open read)
  ```

- [ ] **Step 1: Write the failing handler test**

`backend/internal/http/uploads/handler_test.go` — table tests with `httptest`:
```go
package uploads

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	"github.com/Pashteto/lia/internal/storage"
)

func okAuth(string) (*apimodels.User, error) {
	id := strfmtUUID() // helper returning a valid apimodels principal with a UUID
	return id, nil
}

func TestUpload_RejectsAnonymous(t *testing.T) {
	h := NewHandler(memStore(t), &fakeFiles{}, func(string) (*apimodels.User, error) { return nil, http.ErrNoCookie })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestUpload_RejectsNonImage(t *testing.T) {
	body, ct := multipartFile(t, "file", "x.txt", "text/plain", []byte("nope"))
	h := NewHandler(memStore(t), &fakeFiles{}, okAuth)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("want 415, got %d", rec.Code)
	}
}

func TestUpload_AcceptsImage_Returns201WithURL(t *testing.T) {
	body, ct := multipartFile(t, "file", "a.png", "image/png", []byte{0x89, 0x50, 0x4e, 0x47})
	h := NewHandler(memStore(t), &fakeFiles{}, okAuth)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	// body contains "id" and "url"
}
```
(Provide `multipartFile`, `memStore` (uses `storage.NewLocal(t.TempDir(), ...)`), `fakeFiles` (implements `files.Service`), and `strfmtUUID` helpers in the test file. The detected content type uses the multipart part header + a sniff of the first 512 bytes via `http.DetectContentType`; allowlist = png/jpeg/webp.)

- [ ] **Step 2: Run to verify fail**

Run: `cd backend && go test ./internal/http/uploads/...`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement the handler**

`backend/internal/http/uploads/handler.go` — a `http.ServeMux`-based handler:
- `POST /api/v1/uploads`: require `Authorization: Bearer <t>` → call `authFn`; on error → 401. Parse multipart (`r.ParseMultipartForm(maxBytes)`, `maxBytes = 5<<20`). Read the part, sniff content type via `http.DetectContentType(first512)`, validate against `{image/png, image/jpeg, image/webp}` → 415 if not. Enforce 5 MB → 413. Generate `key = "uploads/" + uuid.NewV4().String() + ext(ct)`. `store.Put(...)`. `files.Register(...)` with owner = principal UUID. Respond 201 `{"id": <fileID>, "url": store.URL(key)}`.
- `GET /api/v1/files/{key...}`: strip the `/api/v1/files/` prefix → key. `store.Get(ctx, key)` → stream with the stored content type (look up via files repo by key is optional; for the demo, set `Content-Type` from `http.DetectContentType` of the first bytes or default `application/octet-stream`). 404 on `storage.ErrNotFound`.

Wire into `backend/internal/http/module.go`: add `SetStorage(storage.Storage)` + `SetFilesService(files.Service)` setters; in `initAPI`, build the uploads handler and put it first in the alice chain via a path-routing wrapper:
```go
base := api.Serve(nil)
mounted := uploads.NewHandler(m.storage, m.files, m.auth.CheckAuth)
router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    p := r.URL.Path
    if strings.HasPrefix(p, "/api/v1/uploads") || strings.HasPrefix(p, "/api/v1/files/") {
        mounted.ServeHTTP(w, r)
        return
    }
    base.ServeHTTP(w, r)
})
handler := alice.New(middlewares.Recovery(), middlewares.Logger(), middlewares.Cors(m.config.CORS), middlewares.RateLimit(m.config.RateLimit)).Then(router)
```

- [ ] **Step 4: Run tests to verify pass**

Run: `cd backend && go test ./internal/http/uploads/... && go build ./...`
Expected: PASS + build ok.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/uploads/ backend/internal/http/module.go
git commit -m "feat(uploads): authenticated image upload + blob serve handlers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 3.4: Wire storage + files into application.go

**Files:**
- Modify: `backend/internal/application.go` (build storage, files service; inject into HTTP module)

- [ ] **Step 1: Build storage + files service and inject**

In `registerModules`, after the `repoModule != nil` domain wiring block, add:
```go
var filesSvc filesdomain.Service
var blobStore storage.Storage
if repoModule != nil && app.config.Storage != nil {
    bs, err := storage.New(toStorageSettings(app.config.Storage)) // helper maps config→storage.StorageSettings
    if err != nil {
        return fmt.Errorf("init storage: %w", err)
    }
    blobStore = bs
    filesSvc = filesdomain.NewService(filesdomain.NewRepository(repoModule.DB()), bs)
    logger.Log().Infof("storage backend %q wired", app.config.Storage.Backend)
}
```
Then in the HTTP block: `httpModule.SetStorage(blobStore); httpModule.SetFilesService(filesSvc)`. Add a `toStorageSettings` helper (in `application.go` or `config`) mapping `*config.StorageConfig` → `storage.StorageSettings` (avoids the import cycle). Default `PublicBase` to `https://<api-host>/api/v1/files` or the configured value.

- [ ] **Step 2: Build + run, smoke the upload locally**

```bash
cd backend && go build ./... && go vet ./...
# run app (host or compose), then:
TOKEN=...   # from demo-login (or MockAuth)
curl -s -X POST localhost:8080/api/v1/uploads -H "Authorization: Bearer $TOKEN" -F file=@some.png
# → {"id":"...","url":"http://localhost:8080/api/v1/files/uploads/<uuid>.png"}
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/api/v1/files/uploads/<uuid>.png   # → 200
```
Expected: 201 with id+url; GET returns 200.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/application.go backend/config/
git commit -m "feat(storage): wire storage + files service into application

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase 4 — Attach cover image (events) + avatar (users)

### Task 4.1: Migration — `events.cover_file_id` + `users.avatar_file_id`

**Files:**
- Create: `backend/db/migrations/000011_cover_avatar.up.sql` / `.down.sql`

- [ ] **Step 1: Write the migration**

`000011_cover_avatar.up.sql`:
```sql
ALTER TABLE events ADD COLUMN IF NOT EXISTS cover_file_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';
ALTER TABLE users  ADD COLUMN IF NOT EXISTS avatar_file_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';
```
`.down.sql`:
```sql
ALTER TABLE events DROP COLUMN IF EXISTS cover_file_id;
ALTER TABLE users  DROP COLUMN IF EXISTS avatar_file_id;
```

- [ ] **Step 2: Apply + verify, then commit** (as Task 3.1 Steps 2–3 pattern).

### Task 4.2: Model + swagger + formatter — `cover_file_id` in, `cover_url` out

**Files:**
- Modify: `backend/internal/models/event.go` (add `CoverFileID uuid.UUID` + transient `CoverURL string \`pg:"-"\``)
- Modify: `backend/api/swagger.yaml` (EventInput: `cover_file_id` string uuid; Event: `cover_url` read-only string) → `make generate-all`
- Modify: `backend/internal/http/formatter/*.go` (EventFromAPIInput reads cover_file_id; EventToAPI sets cover_url via storage.URL using the loaded file)
- Modify: `backend/internal/events/repository.go` (load cover file key for events; compute URL) OR resolve in service
- Test: `backend/internal/events/service_test.go` (cover_file_id round-trips), formatter test if present

**Interfaces:**
- Consumes: `storage.Storage.URL`, `files.Service.Get`.
- Produces: events carry `CoverFileID`; API responses include `cover_url` (empty when unset).

- [ ] **Step 1: Write/extend failing tests**

Add to `backend/internal/events/service_test.go` a test that a created event preserves a non-zero `CoverFileID`, and (if cover resolution lives in the events service) that `CoverURL` is populated from a fake file resolver + fake storage. Keep the events service's new dependency behind a small local interface:
```go
type CoverResolver interface { // satisfied by files.Service + storage.Storage wrapper
    URLFor(ctx context.Context, fileID uuid.UUID) (string, error)
}
```

- [ ] **Step 2: Run to verify fail.** `cd backend && go test ./internal/events/...` → FAIL.

- [ ] **Step 3: Implement**
  - `models/event.go`: add `CoverFileID uuid.UUID \`pg:"cover_file_id,type:uuid,use_zero"\`` and `CoverURL string \`pg:"-"\``.
  - `swagger.yaml`: add `cover_file_id` (string, format uuid) to the create input and `cover_url` (string, read-only) to the Event response; run `make generate-all`.
  - formatter: `EventFromAPIInput` parses `cover_file_id` (zero-uuid when empty); `EventToAPI` emits `cover_url` from `event.CoverURL`.
  - events service `Create`: if `CoverFileID != uuid.Nil`, optionally validate it exists via the resolver (non-fatal if absent — keep loose like venue_id). On load (GetByID/List), populate `CoverURL` for events with a non-zero `CoverFileID` (single query: join `files`, then `storage.URL(key)`), mirroring `loadVenues`.

- [ ] **Step 4: Run tests + build.** `cd backend && go test ./... && go build ./...` → PASS.

- [ ] **Step 5: Commit.**
```bash
git add backend/internal/models/event.go backend/api/swagger.yaml backend/internal/http/formatter backend/internal/events backend/db/migrations/000011_*
git commit -m "feat(events): cover_file_id input + cover_url output via storage

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 4.3: Frontend — cover image upload in CreateEventForm + render on cards/detail

**Files:**
- Modify: `frontend/lib/api.ts` (add `uploadFile(file): Promise<{id,url}>`; add `cover_file_id` to `CreateEventInput`; map `cover_url`→`coverUrl` in `apiEventToLia`)
- Modify: `frontend/lib/types.ts` (`LiaEvent.coverUrl?: string`, `ApiEvent.cover_url?`)
- Modify: `frontend/components/CreateEventForm.tsx` (file input → upload → set cover_file_id; preview)
- Modify: `frontend/components/DiscoveryFeed.tsx` + event detail (render `coverUrl` when present)

- [ ] **Step 1: Add `uploadFile` to `lib/api.ts`**
```ts
export async function uploadFile(file: File): Promise<{ id: string; url: string }> {
  const token = getToken();
  const fd = new FormData();
  fd.append("file", file);
  const res = await fetch(`${API_V1}/uploads`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: fd,
  });
  if (!res.ok) throw new Error(`upload failed: ${res.status} ${await res.text().catch(() => "")}`);
  return (await res.json()) as { id: string; url: string };
}
```
Add `cover_file_id?: string` to `CreateEventInput`; in `apiEventToLia` add `coverUrl: e.cover_url`.

- [ ] **Step 2: Wire the file input in CreateEventForm** — on change, call `uploadFile`, store returned `id` into form state, show a preview from the returned `url`; submit `cover_file_id`.

- [ ] **Step 3: Render cover on feed + detail** — `<img src={event.coverUrl}>` with `max-width:100%`, only when present.

- [ ] **Step 4: Lint + build.** `cd frontend && pnpm lint && pnpm build` → clean.

- [ ] **Step 5: Commit.**
```bash
git add frontend/lib/api.ts frontend/lib/types.ts frontend/components/CreateEventForm.tsx frontend/components/DiscoveryFeed.tsx
git commit -m "feat(frontend): event cover image upload + render

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

> Avatar upload (users.avatar_file_id) reuses `uploadFile` + a `PATCH /users` field. Ship event covers first; avatar UI is a thin follow-on within this task's pattern — implement it only if the user wants it in this slice (the column + storage already support it).

---

## Phase 5 — Event-creation quota (10/calendar-month/user)

### Task 5.1: Repository count + config

**Files:**
- Modify: `backend/config/scheme.go` (`EventsMonthlyLimit int \`mapstructure:"events_monthly_limit"\`` on HTTPConfig or a new `QuotaConfig`; default 10 in `init.go`)
- Modify: `backend/internal/events/repository.go` (add `CountByOrganizerSince(organizer uuid.UUID, since time.Time) (int, error)`)
- Modify: `backend/internal/events/service.go` (interface + Create enforcement)
- Test: `backend/internal/events/service_test.go`

**Interfaces:**
- Produces:
  ```go
  // events.Repository
  CountByOrganizerSince(organizer uuid.UUID, since time.Time) (int, error)
  // events package
  var ErrQuotaExceeded = errors.New("monthly event limit reached")
  // NewService gains a limit param:
  func NewService(repo Repository, categories CategoryValidator, venues VenueValidator, monthlyLimit int) Service
  ```

- [ ] **Step 1: Write the failing quota test (fake repo returns the count)**

In `service_test.go`, extend the fake events repo with `CountByOrganizerSince` returning a settable value. Tests:
```go
func TestCreate_UnderLimit_OK(t *testing.T)     // count=9, limit=10 → no error
func TestCreate_AtLimit_ReturnsErrQuota(t *testing.T) // count=10, limit=10 → errors.Is(err, ErrQuotaExceeded)
func TestCreate_LimitZero_Unlimited(t *testing.T)     // limit<=0 → never blocks
```
Each builds a valid `*models.Event` with a non-nil `OrganizerID`, fake category/venue validators returning empty/ok.

- [ ] **Step 2: Run to verify fail.** `cd backend && go test ./internal/events/...` → FAIL.

- [ ] **Step 3: Implement**
  - `events.go`: add `ErrQuotaExceeded`.
  - service: store `monthlyLimit`; in `Create`, before persisting, if `monthlyLimit > 0 && event.OrganizerID != uuid.Nil`:
    ```go
    since := startOfMonthMoscow(time.Now())
    n, err := s.repo.CountByOrganizerSince(event.OrganizerID, since)
    if err != nil { return fmt.Errorf("quota check: %w", err) }
    if n >= s.monthlyLimit { return fmt.Errorf("%w: %d/%d this month", ErrQuotaExceeded, n, s.monthlyLimit) }
    ```
    Add `startOfMonthMoscow(t time.Time) time.Time` using `time.LoadLocation("Europe/Moscow")` → first day 00:00 of that month.
  - repository: `CountByOrganizerSince` → `db.Model((*models.Event)(nil)).Where("organizer_id = ?", organizer).Where("created_at >= ?", since).Count()`.
  - Update `NewService` callers (application.go) to pass the configured limit.

- [ ] **Step 4: Run tests + build.** PASS.

- [ ] **Step 5: Commit.**
```bash
git add backend/internal/events backend/config backend/internal/application.go
git commit -m "feat(events): 10/calendar-month creation quota per user

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 5.2: Map quota error → HTTP 429

**Files:**
- Modify: `backend/internal/http/handlers/events.go` (CreateEvent: map `ErrQuotaExceeded` → 429)
- Modify: `backend/api/swagger.yaml` (CreateEvent: add 429 response) → `make generate-all`
- Test: `backend/internal/http/handlers/events_test.go`

- [ ] **Step 1: Write failing handler test** — fake events service returns `events.ErrQuotaExceeded`; assert the responder is the 429 variant with the Russian message in the payload.

- [ ] **Step 2: Run to verify fail.** FAIL.

- [ ] **Step 3: Implement** — in `CreateEvent.Handle`, add a case:
```go
case errors.Is(err, eventsdomain.ErrQuotaExceeded):
    return eventsops.NewCreateEventTooManyRequests().
        WithPayload(DefaultError(http.StatusTooManyRequests, errors.New("Достигнут лимит: 10 событий в месяц. Лимит обновится 1-го числа."), nil))
```
Add the `429` response to `CreateEvent` in `swagger.yaml`, regenerate so `NewCreateEventTooManyRequests` exists.

- [ ] **Step 4: Run tests + build.** PASS.

- [ ] **Step 5: Commit.**
```bash
git add backend/internal/http/handlers/events.go backend/internal/http/handlers/events_test.go backend/api/swagger.yaml
git commit -m "feat(events): 429 on monthly quota exceeded

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 5.3: Frontend — surface the 429 quota message

**Files:**
- Modify: `frontend/components/CreateEventForm.tsx` (the mutation error path already exists; show the backend's 429 message verbatim)

- [ ] **Step 1:** In the create mutation's `onError`, if the error message includes `429`, show the Russian quota message; otherwise the generic error. (The backend message is already in `error.message` from `createEvent`'s thrown text.)
- [ ] **Step 2:** `pnpm lint && pnpm build` → clean.
- [ ] **Step 3: Commit.**
```bash
git add frontend/components/CreateEventForm.tsx
git commit -m "feat(frontend): surface monthly quota (429) on create form

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase 6 — Cleanup cron (orphaned uploads, 24h grace)

### Task 6.1: Cleanup service (delete orphans older than 24h)

**Files:**
- Create: `backend/internal/files/cleanup.go` (a `Cleaner` that lists orphans + deletes blob+row, logged)
- Test: `backend/internal/files/cleanup_test.go`

**Interfaces:**
- Consumes: `Repository.ListOrphansOlderThan`, `Repository.Delete`, `storage.Storage.Delete`.
- Produces:
  ```go
  type Cleaner struct { /* repo, store, grace time.Duration */ }
  func NewCleaner(repo Repository, store storage.Storage, grace time.Duration) *Cleaner
  func (c *Cleaner) Run(ctx context.Context) (deleted int, err error) // logs candidate + deleted counts
  ```

- [ ] **Step 1: Write the failing test** — fake repo returns 2 orphans; fake storage records deletes. Assert `Run` deletes both blobs + both rows and returns `deleted=2`. A second test: a per-file storage error is logged and skipped, not fatal (returns `deleted=1, err=nil` with one failure).

- [ ] **Step 2: Run to verify fail.** FAIL.

- [ ] **Step 3: Implement** — `Run` calls `ListOrphansOlderThan(grace)`, logs the candidate count, then for each: `store.Delete(key)` (log+skip on error), `repo.Delete(id)`. Returns the deleted count. Logs the final deleted count (audit-aware).

- [ ] **Step 4: Run tests.** PASS.

- [ ] **Step 5: Commit.**
```bash
git add backend/internal/files/cleanup.go backend/internal/files/cleanup_test.go
git commit -m "feat(files): orphan cleanup service (24h grace, logged)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 6.2: Cleanup module (daily in-process ticker) + config

**Files:**
- Create: `backend/internal/cleanup/module.go` (implements `module.Module`; ticker)
- Modify: `backend/config/scheme.go` + `init.go` (`Cleanup *CleanupConfig`: `Enabled bool` default true, `Interval string` default "24h", `Grace string` default "24h")
- Modify: `backend/internal/application.go` (register the cleanup module when files service is wired + enabled)
- Test: `backend/internal/cleanup/module_test.go` (Start runs one tick quickly with a short interval + a fake Cleaner)

**Interfaces:**
- Consumes: `*files.Cleaner`, `CleanupConfig`.
- Produces: a `module.Module` (`Name()="cleanup"`, `Init/Start/Stop`). `Start` runs an initial `Run` then ticks every `Interval`; `Stop` cancels the goroutine.

- [ ] **Step 1: Write the failing module test** — inject a fake cleaner; `Start` with `Interval=20ms`; assert the cleaner's `Run` is called ≥1 within 100ms; `Stop` halts further calls.

- [ ] **Step 2: Run to verify fail.** FAIL.

- [ ] **Step 3: Implement** the module: a context-cancellable goroutine with `time.NewTicker`. Guard registration in `application.go` behind `app.config.Cleanup.Enabled && filesSvc != nil`, constructing `files.NewCleaner(filesRepo, blobStore, graceDuration)`. (Expose the files repo or add `files.Service` method to get a Cleaner — simplest: build the cleaner in application.go from the same repo+store used for filesSvc.)

- [ ] **Step 4: Run tests + build.** PASS + build ok.

- [ ] **Step 5: Commit.**
```bash
git add backend/internal/cleanup backend/config backend/internal/application.go
git commit -m "feat(cleanup): daily in-process orphan-file cleanup module

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 6.3: `lia files:cleanup` CLI subcommand

**Files:**
- Create: `backend/cmd/cleanup/cleanup.go` (cobra command that builds the app's storage+files, runs `Cleaner.Run` once, prints the deleted count)
- Modify: `backend/cmd/root/root.go` (register the subcommand)
- Test: `backend/cmd/cleanup/cleanup_test.go` (command wiring; flag parsing)

- [ ] **Step 1: Write a minimal failing test** for command construction (the `cobra.Command` has `Use: "files:cleanup"` and a `RunE`). 
- [ ] **Step 2: Run to verify fail.** FAIL.
- [ ] **Step 3: Implement** — mirror `cmd/serve/serve.go`'s config bootstrap, build storage+files repo, `files.NewCleaner(...).Run(ctx)`, log `deleted=N`. Register under root.
- [ ] **Step 4: Run tests + build + manual run.**
```bash
cd backend && go test ./cmd/... && go build -o /tmp/lia ./cmd/lia.go
/tmp/lia files:cleanup   # with DATABASE_*/STORAGE_* env → prints deleted=N
```
- [ ] **Step 5: Commit.**
```bash
git add backend/cmd/cleanup backend/cmd/root/root.go
git commit -m "feat(cleanup): lia files:cleanup CLI subcommand

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase 7 — Integration, deploy, docs

### Task 7.1: Full local verification

- [ ] **Step 1: Backend gate**
```bash
cd backend && go build ./... && go vet ./... && go test ./... 
golangci-lint run   # v1, must exit 0
```
Expected: all pass.

- [ ] **Step 2: Frontend gate**
```bash
cd frontend && pnpm lint && pnpm build
```
Expected: clean.

- [ ] **Step 3: End-to-end local smoke** — bring up compose (or host workaround), then: demo-login → token → upload a PNG → create event with `cover_file_id` → GET event shows `cover_url` → GET the file 200 → create 10 events to hit 429 on the 11th → run `lia files:cleanup` (upload an orphan, backdate it, confirm it's reaped; a fresh upload survives the 24h grace).

### Task 7.2: Deploy to vds-ru215 + verify live

> Access-control & infra change — document for ISO 27001 / Vanta (Task 7.3). The box is flaky: build detached, poll.

- [ ] **Step 1: Backend** — rsync `backend/` → `/opt/lia/backend`; ensure `.env.prod` has `STORAGE_BACKEND=local`, `STORAGE_LOCAL_DIR=/data/uploads`, `STORAGE_PUBLIC_BASE=https://api.lia.pashteto.com/api/v1/files`, `EVENTS_MONTHLY_LIMIT=10`, `FILE_CLEANUP_ENABLED=true`; add a Docker volume for `/data/uploads` in `docker-compose.prod.yml`. Run migrate + `up -d --build app`.
- [ ] **Step 2: Frontend** — rsync `frontend/` → `/opt/lia/frontend`, rebuild image with `NEXT_PUBLIC_API_URL=https://api.lia.pashteto.com`, `docker rm -f && docker run`.
- [ ] **Step 3: Live verification** — repeat Task 7.1 Step 3's curls against `https://api.lia.pashteto.com`: demo-login 200, upload 201, create-with-cover 201, `cover_url` resolves (GET file 200), 11th create → 429.
- [ ] **Step 4: Confirm cleanup** — check `docker logs backend-app-1` for a cleanup run log line; optionally `docker compose exec app /app/lia files:cleanup`.

### Task 7.3: Update HANDOFF + memory + docs

- [ ] **Step 1: Update `docs/HANDOFF.md`** — auth Phase C done + verified live (demo-login fixed; round trip 401/201); storage layer (local now, S3 swappable via `STORAGE_BACKEND=s3` with RU endpoints); event quota (10/calendar-month, 429); cleanup cron (daily, 24h grace, `lia files:cleanup`); product renamed to Presence.Tarski (display only). **Document the access-control posture** (auth enforced; demo-login non-prod control) for ISO 27001 / Vanta. Note the new `/data/uploads` volume as a hand-managed-infra exception.
- [ ] **Step 2: Update memory** — refresh `lia-project-state.md` (Presence.Tarski rename, storage/quota/cleanup shipped) and `lia-demo-deployment.md` (uploads volume, new env vars). Add a `reference` memory for the RU-zone S3 swap recipe (endpoints + `region=us-east-1` + `STORAGE_BACKEND=s3`).
- [ ] **Step 3: Commit docs.**
```bash
git add docs/HANDOFF.md
git commit -m "docs: HANDOFF — Presence.Tarski auth/storage/quota/cleanup shipped

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 7.4: Finish the branch

- [ ] Use superpowers:finishing-a-development-branch to decide merge/PR. Open a PR `feat/presence-tarski-storage-quota` → `main` summarizing the five workstreams, the access-control change, and the new env vars / `/data/uploads` volume.

---

## Self-Review

**Spec coverage:**
- WS0 rename → Phase 0 ✅
- WS1 auth finish (verify panic, frontend, deploy) → Phase 1 + Phase 7.2 ✅
- WS2 storage interface + local + S3 + MinIO smoke + config → Phase 2 ✅
- Uploads + files table + serve → Phase 3 ✅
- Cover image + avatar (avatar noted as thin follow-on; column shipped) → Phase 4 ✅
- WS3 quota 10/calendar-month + 429 + frontend → Phase 5 ✅
- WS4 cleanup cron + CLI + 24h grace → Phase 6 ✅
- Compliance/security notes (auth posture, upload allowlist, destructive-cron blast radius, secrets) → Global Constraints + Phase 7.3 ✅

**Placeholder scan:** No "TBD/TODO/handle edge cases" left as work items — config default-binding and formatter edits reference the existing file's pattern (read-first) rather than inventing line numbers, which is accurate given generated/edited files. Avatar UI is explicitly deferred with rationale, not a silent gap.

**Type consistency:** `Storage` methods, `StorageSettings`/`S3Config`, `files.Service` (`Register`/`Get`), `Repository` (`Create/GetByID/ListOrphansOlderThan/Delete`), `events.NewService(..., monthlyLimit int)`, `ErrQuotaExceeded`, and `Cleaner.Run` are used consistently across tasks. `ListOrphansOlderThan(d time.Duration)` is the canonical signature (the Task 3.2 fake comment notes aligning to it).

**Scope:** Cohesive single branch, phased; each phase ends with build/test green and most with a deployable increment.
