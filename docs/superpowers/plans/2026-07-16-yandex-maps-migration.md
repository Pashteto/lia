# Yandex Maps Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Leaflet + OpenStreetMap tiles + OSM Nominatim geocoding with Yandex Maps (JS API v3) + an auth-gated backend Yandex Geocoder proxy, so the app renders legally-compliant maps for the Russian deployment.

**Architecture:** A new `YandexMap` React component preserves the existing `LeafletMap` prop interface, so the three map consumers change only their import. Geocoding moves server-side: a new Go `internal/geocode` client calls the Yandex Geocoder HTTP API, wrapped by an auth-gated `GET /api/v1/geocode` hand-mounted handler; the frontend calls that proxy with a Bearer token. Leaflet + Nominatim are removed.

**Tech Stack:** Next.js 16 / React 19 / TypeScript, vitest (node env), Tailwind v4; Go backend (go-swagger mux + hand-mounted `http.Handler`s, spf13/viper config), Yandex JS API v3, Yandex Geocoder HTTP API 1.x.

## Global Constraints

- Go module path: `github.com/Pashteto/lia` (verify against `backend/go.mod`; use the module's actual import prefix for internal packages).
- **Coordinate order differs by system:** Yandex uses `[longitude, latitude]`; the existing component props and `MapPin` use `[latitude, longitude]` (Leaflet order). Keep the component's public props as `[lat, lon]` and convert internally. Do NOT change consumer call sites' coordinate order.
- Yandex Geocoder `Point.pos` is a `"lon lat"` space-separated string.
- `NEXT_PUBLIC_*` env vars are inlined by Next.js **at build time** — the frontend build-arg on the deploy box must include `NEXT_PUBLIC_YANDEX_MAPS_KEY`.
- Frontend package manager is **pnpm** (`frontend/pnpm-lock.yaml`).
- UI copy stays in Russian, matching existing components.
- **Yandex ToS requires visible attribution** — the JS API renders its own © control; do not hide it.
- **API key values never enter the repo.** Reference them by variable name. The JS API key (Yandex cabinet Key #3) goes in `frontend/.env.local` (gitignored) + deploy build-arg; the Geocoder key (Key #4) goes in `YANDEX_GEOCODER_KEY` backend env only.
- Frontend authed requests use `Authorization: Bearer ${getToken()}` (localStorage token), NOT cookies.

---

### Task 1: Backend Yandex Geocoder client

**Files:**
- Create: `backend/internal/geocode/client.go`
- Test: `backend/internal/geocode/client_test.go`

**Interfaces:**
- Produces:
  - `type Result struct { Lat float64 `json:"lat"`; Lon float64 `json:"lon"`; Label string `json:"label"` }`
  - `func NewClient(apiKey string) *Client`
  - `func (c *Client) Geocode(ctx context.Context, q string) ([]Result, error)`
  - unexported field `c.endpoint` (tests in-package override it)

- [ ] **Step 1: Write the failing test**

Create `backend/internal/geocode/client_test.go`:

```go
package geocode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeocodeParsesFeatureMembers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("geocode"); got != "Москва" {
			t.Errorf("geocode param = %q, want Москва", got)
		}
		if got := r.URL.Query().Get("apikey"); got != "test-key" {
			t.Errorf("apikey = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"GeoObjectCollection":{"featureMember":[
			{"GeoObject":{"metaDataProperty":{"GeocoderMetaData":{"text":"Россия, Москва"}},"Point":{"pos":"37.617635 55.755814"}}}
		]}}}`))
	}))
	defer srv.Close()

	c := NewClient("test-key")
	c.endpoint = srv.URL

	got, err := c.Geocode(context.Background(), "Москва")
	if err != nil {
		t.Fatalf("Geocode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Label != "Россия, Москва" {
		t.Errorf("label = %q", got[0].Label)
	}
	if got[0].Lat != 55.755814 || got[0].Lon != 37.617635 {
		t.Errorf("coords = %v,%v want 55.755814,37.617635", got[0].Lat, got[0].Lon)
	}
}

func TestGeocodeBlankQuerySkipsRequest(t *testing.T) {
	c := NewClient("k")
	got, err := c.Geocode(context.Background(), "   ")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/geocode/`
Expected: FAIL — `undefined: NewClient` (package has no implementation yet).

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/geocode/client.go`:

```go
// Package geocode is the backend proxy client for the Yandex Geocoder HTTP API.
// It is the first outbound HTTP client in the backend — there is no prior
// pattern to mirror.
package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultEndpoint = "https://geocode-maps.yandex.ru/1.x/"

// Result is one geocoded address, in [lat, lon] terms for frontend consumption.
type Result struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Label string  `json:"label"`
}

// Client calls the Yandex Geocoder HTTP API.
type Client struct {
	apiKey   string
	endpoint string
	http     *http.Client
}

// NewClient builds a geocoder client bound to apiKey (YANDEX_GEOCODER_KEY).
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: defaultEndpoint,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

// yandexResponse mirrors the subset of the Yandex Geocoder 1.x JSON we read.
type yandexResponse struct {
	Response struct {
		GeoObjectCollection struct {
			FeatureMember []struct {
				GeoObject struct {
					MetaDataProperty struct {
						GeocoderMetaData struct {
							Text string `json:"text"`
						} `json:"GeocoderMetaData"`
					} `json:"metaDataProperty"`
					Point struct {
						Pos string `json:"pos"` // "lon lat"
					} `json:"Point"`
				} `json:"GeoObject"`
			} `json:"featureMember"`
		} `json:"GeoObjectCollection"`
	} `json:"response"`
}

// Geocode returns up to 5 matches for q. A blank query yields an empty slice
// without an HTTP call.
func (c *Client) Geocode(ctx context.Context, q string) ([]Result, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []Result{}, nil
	}
	if c.apiKey == "" {
		return nil, errors.New("geocode: api key not configured")
	}
	params := url.Values{
		"apikey":  {c.apiKey},
		"geocode": {q},
		"format":  {"json"},
		"lang":    {"ru_RU"},
		"results": {"5"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocode: upstream status %d", res.StatusCode)
	}
	var yr yandexResponse
	if err := json.NewDecoder(res.Body).Decode(&yr); err != nil {
		return nil, err
	}
	members := yr.Response.GeoObjectCollection.FeatureMember
	out := make([]Result, 0, len(members))
	for _, m := range members {
		lon, lat, ok := parsePos(m.GeoObject.Point.Pos)
		if !ok {
			continue
		}
		out = append(out, Result{
			Lat:   lat,
			Lon:   lon,
			Label: m.GeoObject.MetaDataProperty.GeocoderMetaData.Text,
		})
	}
	return out, nil
}

// parsePos parses a Yandex "lon lat" string into (lon, lat).
func parsePos(pos string) (lon, lat float64, ok bool) {
	parts := strings.Fields(pos)
	if len(parts) != 2 {
		return 0, 0, false
	}
	lon, err1 := strconv.ParseFloat(parts[0], 64)
	lat, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return lon, lat, true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/geocode/`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/geocode/
git commit -m "feat(geocode): Yandex Geocoder HTTP client"
```

---

### Task 2: Backend auth-gated `/geocode` handler

**Files:**
- Create: `backend/internal/http/geocode/handler.go`
- Test: `backend/internal/http/geocode/handler_test.go`

**Interfaces:**
- Consumes (Task 1): `geo.Result`, `geo.NewClient` (via the `Geocoder` interface below).
- Produces:
  - `type Geocoder interface { Geocode(ctx context.Context, q string) ([]geo.Result, error) }`
  - `type Deps struct { Authenticate func(token string) (*domain.User, error); Client Geocoder }`
  - `func NewHandler(deps Deps) http.Handler` — serves `GET /api/v1/geocode?q=...`

> NOTE: Copy the exact `domain` import path from `backend/internal/http/follows/handler.go`'s import block — it imports the package that provides `*domain.User`. The `principal(r)` helper below is identical to the one in that file.

- [ ] **Step 1: Write the failing test**

Create `backend/internal/http/geocode/handler_test.go` (replace `DOMAIN_IMPORT` with the real path used by follows/handler.go):

```go
package geocode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	geo "github.com/Pashteto/lia/backend/internal/geocode"
	domain "DOMAIN_IMPORT"
)

type fakeGeocoder struct{ results []geo.Result }

func (f fakeGeocoder) Geocode(_ context.Context, _ string) ([]geo.Result, error) {
	return f.results, nil
}

func TestGeocodeRejectsUnauthenticated(t *testing.T) {
	h := NewHandler(Deps{
		Authenticate: func(string) (*domain.User, error) { return nil, nil },
		Client:       fakeGeocoder{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/geocode?q=x", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rr.Code)
	}
}

func TestGeocodeReturnsResultsForAuthed(t *testing.T) {
	h := NewHandler(Deps{
		Authenticate: func(string) (*domain.User, error) { return &domain.User{}, nil },
		Client:       fakeGeocoder{results: []geo.Result{{Lat: 55.7, Lon: 37.6, Label: "Москва"}}},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/geocode?q=Москва", nil)
	req.Header.Set("Authorization", "Bearer t")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
	var got []geo.Result
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Label != "Москва" {
		t.Fatalf("got = %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/geocode/`
Expected: FAIL — `undefined: NewHandler`.

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/http/geocode/handler.go` (replace `DOMAIN_IMPORT` with the real path):

```go
// Package geocode is the auth-gated HTTP proxy in front of the Yandex Geocoder.
// It mirrors the hand-mounted handler shape of internal/http/follows.
package geocode

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	geo "github.com/Pashteto/lia/backend/internal/geocode"
	domain "DOMAIN_IMPORT"
)

// Geocoder is the subset of *geo.Client the handler needs (for test injection).
type Geocoder interface {
	Geocode(ctx context.Context, q string) ([]geo.Result, error)
}

// Deps are the handler's injected dependencies.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Client       Geocoder
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler builds the /api/v1/geocode handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps}
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("GET /api/v1/geocode", h.geocode)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// principal reads the Bearer token and resolves the current user, or nil.
func (h *handler) principal(r *http.Request) *domain.User {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	u, err := h.deps.Authenticate(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil || u == nil {
		return nil
	}
	return u
}

func (h *handler) geocode(w http.ResponseWriter, r *http.Request) {
	if h.principal(r) == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	results, err := h.deps.Client.Geocode(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "geocode_failed")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/geocode/`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/geocode/
git commit -m "feat(geocode): auth-gated /geocode HTTP handler"
```

---

### Task 3: Backend config + wiring (mount `/geocode`)

**Files:**
- Modify: `backend/config/scheme.go` (add `GeocoderConfig` + field on `Scheme`)
- Modify: `backend/config/init.go` (default + `BindEnv`)
- Modify: `backend/internal/application.go` (pass key into the HTTP module — mirror `SetInvitations`/`SetFollows`)
- Modify: `backend/internal/http/module.go` (store key, build client + handler, add router branch — mirror the follows mount at ~L339-348 and the router dispatch at ~L386-418)

**Interfaces:**
- Consumes (Tasks 1–2): `geo.NewClient`, `geocodehttp.NewHandler`, `geocodehttp.Deps`, `m.auth.Authenticate`.
- Produces: live route `GET /api/v1/geocode?q=...`.

- [ ] **Step 1: Add config field**

In `backend/config/scheme.go`, next to the other secret sub-structs (e.g. `SMTPConfig`), add:

```go
// GeocoderConfig holds the Yandex Geocoder HTTP API credentials.
type GeocoderConfig struct {
	Key string `mapstructure:"key"`
}
```

Add a field to the top-level `Scheme` struct (alongside `SMTP`, `S3`, etc.):

```go
	Geocoder GeocoderConfig `mapstructure:"geocoder"`
```

- [ ] **Step 2: Bind the env var**

In `backend/config/init.go`, in `setDefaults()` next to the other `viper.BindEnv` calls (e.g. the `SMTP_*` block), add:

```go
	viper.SetDefault("geocoder.key", "")
	viper.BindEnv("geocoder.key", "YANDEX_GEOCODER_KEY")
```

- [ ] **Step 3: Verify config compiles**

Run: `cd backend && go build ./config/`
Expected: no output (success).

- [ ] **Step 4: Add a `SetGeocoder` injector on the HTTP module**

In `backend/internal/http/module.go`, add a field to the module struct (near `follows`, `invitations`, etc.):

```go
	geocoderKey string
```

Add a setter, mirroring the existing `SetFollows`/`SetInvitations` methods:

```go
// SetGeocoder injects the Yandex Geocoder API key.
func (m *Module) SetGeocoder(key string) { m.geocoderKey = key }
```

- [ ] **Step 5: Build + mount the handler**

In `backend/internal/http/module.go`, where the other hand-mounted handlers are constructed (near the `followsH := ...` block, ~L339), add — importing the two new packages at the top (`geo "github.com/Pashteto/lia/backend/internal/geocode"` and `geocodehttp "github.com/Pashteto/lia/backend/internal/http/geocode"`; adjust prefix to match `go.mod`):

```go
	geocodeH := geocodehttp.NewHandler(geocodehttp.Deps{
		Authenticate: m.auth.Authenticate,
		Client:       geo.NewClient(m.geocoderKey),
	})
```

In the router dispatch `http.HandlerFunc` (the `p := r.URL.Path` block, ~L386-418), add a branch next to the `/metrics`/follows branches:

```go
		if p == "/api/v1/geocode" {
			geocodeH.ServeHTTP(w, r)
			return
		}
```

- [ ] **Step 6: Call the injector at startup**

In `backend/internal/application.go`, immediately after the HTTP module is created and the other `SetXxx` injectors are called (~L240-256), add:

```go
	httpModule.SetGeocoder(app.config.Geocoder.Key)
```

(Use the actual module variable name in that file — match the surrounding `SetInvitations(...)` call.)

- [ ] **Step 7: Build the whole backend**

Run: `cd backend && go build ./... && go test ./internal/geocode/ ./internal/http/geocode/`
Expected: build succeeds; tests PASS.

- [ ] **Step 8: Manual smoke (optional, needs a real key + running backend)**

With `YANDEX_GEOCODER_KEY` set and the backend running, and a valid bearer token in `$TOKEN`:

```bash
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8080/api/v1/geocode?q=Москва" | head
# Expected: JSON array of {lat,lon,label}. Without the header: {"error":"unauthorized"} (401).
```

- [ ] **Step 9: Commit**

```bash
git add backend/config/scheme.go backend/config/init.go backend/internal/application.go backend/internal/http/module.go
git commit -m "feat(geocode): wire YANDEX_GEOCODER_KEY + mount /api/v1/geocode"
```

---

### Task 4: Frontend coordinate helper

**Files:**
- Create: `frontend/lib/coords.ts`
- Test: `frontend/lib/__tests__/coords.test.ts`

**Interfaces:**
- Produces: `export function toLngLat(latLon: [number, number]): [number, number]` — converts `[lat, lon]` → `[lon, lat]` for Yandex.

- [ ] **Step 1: Write the failing test**

Create `frontend/lib/__tests__/coords.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { toLngLat } from "@/lib/coords";

describe("toLngLat", () => {
  it("flips [lat, lon] to [lon, lat] for Yandex", () => {
    expect(toLngLat([55.7558, 37.6173])).toEqual([37.6173, 55.7558]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && pnpm test -- --run coords`
Expected: FAIL — cannot resolve `@/lib/coords`.

- [ ] **Step 3: Write minimal implementation**

Create `frontend/lib/coords.ts`:

```ts
// Yandex Maps uses [longitude, latitude]; our component props use [lat, lon]
// (Leaflet order). Convert at the boundary.
export function toLngLat(latLon: [number, number]): [number, number] {
  return [latLon[1], latLon[0]];
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && pnpm test -- --run coords`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/coords.ts frontend/lib/__tests__/coords.test.ts
git commit -m "feat(map): coordinate helper for Yandex [lon,lat] order"
```

---

### Task 5: Frontend geocode lib (backend proxy)

**Files:**
- Create: `frontend/lib/geocode.ts` (replaces `frontend/lib/nominatim.ts`, deleted in Task 7)
- Test: `frontend/lib/__tests__/geocode.test.ts`

**Interfaces:**
- Consumes: `API_BASE` from `@/lib/api`, `getToken` from `@/lib/auth`.
- Produces:
  - `export interface GeoResult { lat: number; lon: number; label: string }`
  - `export async function geocodeAddress(q: string): Promise<GeoResult[]>`

- [ ] **Step 1: Write the failing test**

Create `frontend/lib/__tests__/geocode.test.ts`:

```ts
import { describe, expect, it, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth", () => ({ getToken: () => "test-token" }));

import { geocodeAddress } from "@/lib/geocode";

describe("geocodeAddress", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("returns [] for a blank query without calling fetch", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    expect(await geocodeAddress("   ")).toEqual([]);
    expect(f).not.toHaveBeenCalled();
  });

  it("sends a bearer token and returns the backend results", async () => {
    const f = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify([{ lat: 55.7, lon: 37.6, label: "Москва" }]), {
        status: 200,
      }),
    );
    const out = await geocodeAddress("Москва");
    expect(out).toEqual([{ lat: 55.7, lon: 37.6, label: "Москва" }]);
    const [url, init] = f.mock.calls[0];
    expect(String(url)).toContain("/api/v1/geocode?q=");
    expect((init as RequestInit).headers).toMatchObject({
      Authorization: "Bearer test-token",
    });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && pnpm test -- --run geocode`
Expected: FAIL — cannot resolve `@/lib/geocode`.

- [ ] **Step 3: Write minimal implementation**

Create `frontend/lib/geocode.ts`:

```ts
import { API_BASE } from "@/lib/api";
import { getToken } from "@/lib/auth";

export interface GeoResult {
  lat: number;
  lon: number;
  label: string;
}

// Forward geocoding via the auth-gated backend Yandex proxy.
// The backend never exposes the geocoder key to the browser.
export async function geocodeAddress(q: string): Promise<GeoResult[]> {
  const query = q.trim();
  if (query === "") return [];
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const res = await fetch(
    `${API_BASE}/api/v1/geocode?q=${encodeURIComponent(query)}`,
    {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    },
  );
  if (!res.ok) throw new Error(`geocode failed: ${res.status}`);
  return (await res.json()) as GeoResult[];
}
```

> If `API_BASE` is not exported from `frontend/lib/api.ts`, export it there (`export const API_BASE = ...` already exists per the current file). Do not duplicate the constant.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && pnpm test -- --run geocode`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/geocode.ts frontend/lib/__tests__/geocode.test.ts
git commit -m "feat(geocode): frontend geocode lib via backend proxy"
```

---

### Task 6: Frontend `YandexMap` component

**Files:**
- Create: `frontend/components/map/YandexMap.tsx`

**Interfaces:**
- Consumes: `toLngLat` from `@/lib/coords`; `window.ymaps3` (JS API v3, loaded at runtime); `process.env.NEXT_PUBLIC_YANDEX_MAPS_KEY`.
- Produces: `export interface MapPin { id: string; lat: number; lon: number; label?: string; href?: string }` and `export function YandexMap({...})` with the SAME prop shape as the old `LeafletMap`: `center: [number, number]` (lat,lon), `zoom?`, `marker?: [number, number]` (lat,lon), `draggableMarker?`, `onMarkerMove?: (lat, lon) => void`, `pins?: MapPin[]`, `className?`.

> This component is DOM/JS-API heavy; there is no jsdom in the test setup, so it is verified by typecheck + lint + `pnpm build` + manual browser (Task 8 verification), not a unit test. The `ready` state flag below is essential: the JS API loads asynchronously, so marker/pin effects must re-run once the map exists.

- [ ] **Step 1: Write the component**

Create `frontend/components/map/YandexMap.tsx`:

```tsx
"use client";

import { useEffect, useRef, useState } from "react";
import { toLngLat } from "@/lib/coords";

export interface MapPin {
  id: string;
  lat: number;
  lon: number;
  label?: string;
  href?: string;
}

const KEY = process.env.NEXT_PUBLIC_YANDEX_MAPS_KEY ?? "";
const PIN_CLASS =
  "block h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-accent shadow";

// Load the JS API v3 script exactly once across every map instance on the page.
let loaderPromise: Promise<void> | null = null;
function loadYmaps(): Promise<void> {
  const w = window as unknown as { ymaps3?: { ready: Promise<void> } };
  if (w.ymaps3) return w.ymaps3.ready;
  if (loaderPromise) return loaderPromise;
  loaderPromise = new Promise<void>((resolve, reject) => {
    const script = document.createElement("script");
    script.src = `https://api-maps.yandex.ru/v3/?apikey=${KEY}&lang=ru_RU`;
    script.onload = () => w.ymaps3!.ready.then(() => resolve(), reject);
    script.onerror = () => reject(new Error("yandex maps failed to load"));
    document.head.appendChild(script);
  });
  return loaderPromise;
}

export function YandexMap({
  center,
  zoom = 13,
  marker,
  draggableMarker = false,
  onMarkerMove,
  pins,
  className = "h-64 w-full rounded-control",
}: {
  center: [number, number];
  zoom?: number;
  marker?: [number, number];
  draggableMarker?: boolean;
  onMarkerMove?: (lat: number, lon: number) => void;
  pins?: MapPin[];
  className?: string;
}) {
  const elRef = useRef<HTMLDivElement>(null);
  // ymaps objects are untyped (the JS API ships no bundled TS types).
  const mapRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const markerRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const pinRefs = useRef<any[]>([]); // eslint-disable-line @typescript-eslint/no-explicit-any
  const onMoveRef = useRef(onMarkerMove);
  onMoveRef.current = onMarkerMove;
  const [ready, setReady] = useState(false);

  // init once
  useEffect(() => {
    if (!KEY) return;
    let cancelled = false;
    loadYmaps()
      .then(() => {
        if (cancelled || !elRef.current || mapRef.current) return;
        const ymaps3 = (window as any).ymaps3; // eslint-disable-line @typescript-eslint/no-explicit-any
        const { YMap, YMapDefaultSchemeLayer, YMapDefaultFeaturesLayer } = ymaps3;
        const map = new YMap(elRef.current, {
          location: { center: toLngLat(center), zoom },
        });
        map.addChild(new YMapDefaultSchemeLayer());
        map.addChild(new YMapDefaultFeaturesLayer());
        mapRef.current = map;
        setReady(true);
      })
      .catch(() => {
        /* leave placeholder; page stays up */
      });
    return () => {
      cancelled = true;
      mapRef.current?.destroy?.();
      mapRef.current = null;
      setReady(false);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // recenter
  useEffect(() => {
    if (!ready) return;
    mapRef.current?.update?.({ location: { center: toLngLat(center), zoom } });
  }, [ready, center, zoom]);

  // single marker (static or draggable)
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps3 = (window as any).ymaps3; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps3) return;
    const { YMapMarker } = ymaps3;
    if (markerRef.current) {
      map.removeChild(markerRef.current);
      markerRef.current = null;
    }
    if (marker) {
      const el = document.createElement("div");
      el.className = PIN_CLASS;
      const m = new YMapMarker(
        {
          coordinates: toLngLat(marker),
          draggable: draggableMarker,
          onDragEnd: (coords: [number, number]) =>
            onMoveRef.current?.(coords[1], coords[0]),
        },
        el,
      );
      map.addChild(m);
      markerRef.current = m;
    }
  }, [ready, marker, draggableMarker]);

  // multi-pin layer
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps3 = (window as any).ymaps3; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps3) return;
    const { YMapMarker } = ymaps3;
    pinRefs.current.forEach((m) => map.removeChild(m));
    pinRefs.current = [];
    (pins ?? []).forEach((p) => {
      const el = document.createElement(p.href ? "a" : "div");
      if (p.href) (el as HTMLAnchorElement).href = p.href;
      el.title = p.label ?? "";
      el.className = PIN_CLASS;
      const m = new YMapMarker({ coordinates: [p.lon, p.lat] }, el);
      map.addChild(m);
      pinRefs.current.push(m);
    });
  }, [ready, pins]);

  if (!KEY) {
    return (
      <div
        className={`${className} flex items-center justify-center bg-fill text-[13px] text-label-secondary`}
      >
        Карта недоступна
      </div>
    );
  }
  return <div ref={elRef} className={className} />;
}
```

- [ ] **Step 2: Typecheck + lint**

Run: `cd frontend && pnpm lint && pnpm exec tsc --noEmit`
Expected: no errors. (If `tsc` isn't a configured script, `pnpm exec tsc --noEmit` uses the local TypeScript.)

- [ ] **Step 3: Commit**

```bash
git add frontend/components/map/YandexMap.tsx
git commit -m "feat(map): YandexMap component on JS API v3"
```

---

### Task 7: Swap consumers, remove Leaflet + Nominatim

**Files:**
- Modify: `frontend/components/VenueMap.tsx`
- Modify: `frontend/components/MapBrowse.tsx`
- Modify: `frontend/components/VenueGeoModal.tsx`
- Delete: `frontend/components/map/LeafletMap.tsx`
- Delete: `frontend/lib/nominatim.ts`
- Modify: `frontend/package.json` (remove `leaflet`, `@types/leaflet`)

**Interfaces:**
- Consumes (Tasks 5–6): `YandexMap`, `MapPin` from `@/components/map/YandexMap`; `geocodeAddress`, `GeoResult` from `@/lib/geocode`.

- [ ] **Step 1: Swap `VenueMap.tsx`**

Replace the dynamic import target. In `frontend/components/VenueMap.tsx`, change:

```tsx
const LeafletMap = dynamic(() => import("@/components/map/LeafletMap").then((m) => m.LeafletMap), {
  ssr: false,
});
```
to:
```tsx
const YandexMap = dynamic(() => import("@/components/map/YandexMap").then((m) => m.YandexMap), {
  ssr: false,
});
```
and rename the JSX usage `<LeafletMap ... />` → `<YandexMap ... />` (props unchanged).

- [ ] **Step 2: Swap `MapBrowse.tsx`**

Same change in `frontend/components/MapBrowse.tsx`: dynamic import → `YandexMap`, and the `<LeafletMap center={center} pins={pins} .../>` usage → `<YandexMap .../>` (props unchanged).

- [ ] **Step 3: Swap `VenueGeoModal.tsx`**

In `frontend/components/VenueGeoModal.tsx`:
- Change the geocode import:
  ```tsx
  import { geocodeAddress, type GeoResult } from "@/lib/geocode";
  ```
  (was `@/lib/nominatim`).
- Change the dynamic map import + usage to `YandexMap` (as in Step 1).
- Replace the attribution note text:
  ```tsx
  <p className="mt-1 text-[12px] text-label-secondary">
    Поиск адресов — © Яндекс. Перетащите метку для точности.
  </p>
  ```
  (was "© OpenStreetMap / Nominatim").
- Change the debounce comment (currently references the Nominatim usage policy) to a neutral note, e.g. `// Debounce address lookups to ~1 req/s.`

- [ ] **Step 4: Delete the dead files**

```bash
git rm frontend/components/map/LeafletMap.tsx frontend/lib/nominatim.ts
```

- [ ] **Step 5: Remove Leaflet dependencies**

Run:
```bash
cd frontend && pnpm remove leaflet @types/leaflet
```
Expected: `leaflet` and `@types/leaflet` gone from `package.json` dependencies/devDependencies and from `pnpm-lock.yaml`.

- [ ] **Step 6: Verify no Leaflet/Nominatim references remain**

Run:
```bash
cd frontend && grep -rniE "leaflet|nominatim|openstreetmap|tile\.openstreetmap" --include=*.ts --include=*.tsx . | grep -v node_modules
```
Expected: no matches.

- [ ] **Step 7: Build + test + lint**

Run:
```bash
cd frontend && pnpm test -- --run && pnpm lint && pnpm build
```
Expected: tests PASS, lint clean, production build succeeds. (Build needs `NEXT_PUBLIC_YANDEX_MAPS_KEY` in the environment or `.env.local` to exercise the real map; without it the component renders the "Карта недоступна" placeholder, which still builds.)

- [ ] **Step 8: Commit**

```bash
git add frontend/components/VenueMap.tsx frontend/components/MapBrowse.tsx frontend/components/VenueGeoModal.tsx frontend/package.json frontend/pnpm-lock.yaml
git commit -m "feat(map): switch consumers to YandexMap; remove Leaflet + Nominatim"
```

---

### Task 8: Config, docs, and end-to-end verification

**Files:**
- Modify: `frontend/.env.example` (add `NEXT_PUBLIC_YANDEX_MAPS_KEY=`)
- Modify: `backend/.env.prod.example` (add `YANDEX_GEOCODER_KEY=`)
- Modify: `docs/HANDOFF.md` (map notes: Leaflet/OSM → Yandex; new `/geocode` endpoint; env vars; build-arg)
- Create (local only, gitignored — do NOT commit): `frontend/.env.local` entry `NEXT_PUBLIC_YANDEX_MAPS_KEY=<JS API key from Yandex cabinet Key #3>`

- [ ] **Step 1: Add env examples**

Append to `frontend/.env.example`:
```
# Yandex Maps JS API key (public; restrict by HTTP Referer in the Yandex cabinet)
NEXT_PUBLIC_YANDEX_MAPS_KEY=
```
Append to `backend/.env.prod.example`:
```
# Yandex Geocoder HTTP API key (server-side secret)
YANDEX_GEOCODER_KEY=
```

- [ ] **Step 2: Set the local key (not committed)**

Add to `frontend/.env.local` (gitignored): `NEXT_PUBLIC_YANDEX_MAPS_KEY=<JS API key>`. Set `YANDEX_GEOCODER_KEY=<geocoder key>` in the backend's local/dev environment. Both key values come from the Yandex Developer Cabinet — never write them into a committed file.

- [ ] **Step 3: Update HANDOFF.md**

In `docs/HANDOFF.md`, update the map references: replace "Leaflet map"/"Nominatim" mentions with "Yandex JS API v3 map" and "backend Yandex Geocoder proxy (`GET /api/v1/geocode`, auth-gated)". Note the two env vars and that `NEXT_PUBLIC_YANDEX_MAPS_KEY` must be a frontend **build-arg** on the deploy box.

- [ ] **Step 4: Manual end-to-end verification (real keys, both services running)**

With backend running (`YANDEX_GEOCODER_KEY` set) and frontend running (`NEXT_PUBLIC_YANDEX_MAPS_KEY` set):
1. Open `/map` → a Yandex map renders with event pins; a pin click navigates to `/events/{id}`. Borders render per Russian convention (Crimea as RU); no "Support Ukraine" text.
2. Open an event detail page with a located venue → single Yandex pin at the venue.
3. As a logged-in organizer, open the venue picker → type an address → suggestions appear (from `/api/v1/geocode`); pick one → marker moves; drag the marker → coords update; Save persists.
4. Network tab: geocode requests go to your backend `/api/v1/geocode` with a Bearer token — NOT to `nominatim.openstreetmap.org`; the geocoder key is absent from all browser traffic.

Use the `verify` skill / browser automation to drive these and capture screenshots.

- [ ] **Step 5: Commit**

```bash
git add frontend/.env.example backend/.env.prod.example docs/HANDOFF.md
git commit -m "docs(maps): env examples + HANDOFF for Yandex migration"
```

- [ ] **Step 6: Post-migration ops (manual, outside the repo)**

- In the Yandex cabinet, restrict BOTH keys by HTTP Referer / allowed domains (`presence.tarski.ru`, `api.presence.tarski.ru`, `localhost`).
- Consider enabling the backend per-IP rate limiter in production (`RateLimitConfig`) to cap geocoder usage.
- On the deploy box, add `NEXT_PUBLIC_YANDEX_MAPS_KEY` to the frontend build-arg and `YANDEX_GEOCODER_KEY` to the backend env before the next deploy.

---

## Self-Review

**Spec coverage:**
- YandexMap on JS API v3 → Task 6. ✓
- Backend geocode proxy (auth-gated) → Tasks 1–3. ✓
- Frontend geocode lib via backend → Task 5. ✓
- Remove Leaflet + Nominatim, update attribution → Task 7. ✓
- Config `NEXT_PUBLIC_YANDEX_MAPS_KEY` + `YANDEX_GEOCODER_KEY`, docs, build-arg → Task 8. ✓
- Empty-state when key absent → Task 6 ("Карта недоступна"). ✓
- Coordinate-order handling ([lat,lon] props vs [lon,lat] Yandex) → Task 4 + used in Task 6. ✓
- Error handling (401 unauth, 503 upstream, script-load failure) → Tasks 2, 6. ✓
- Testing (client, handler, geocode lib, coords; manual browser) → Tasks 1,2,4,5,8. ✓
- Security (key restriction, rate limit, geocoder key server-side) → Tasks 2/8. ✓

**Type consistency:** `Result{Lat,Lon,Label}` (Go) ↔ `GeoResult{lat,lon,label}` (TS) match via JSON tags. `Geocoder` interface method matches `*Client.Geocode` signature. `MapPin` and `YandexMap` props exactly match the removed `LeafletMap` interface, so consumers are drop-in. `toLngLat` used consistently.

**Placeholder scan:** `DOMAIN_IMPORT` in Tasks 1–2 is an explicit, called-out substitution (the existing `domain` package path from follows/handler.go), not an unresolved TODO — every other step has concrete code/commands.
