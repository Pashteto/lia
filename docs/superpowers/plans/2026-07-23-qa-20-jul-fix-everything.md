# QA-прогон 20 июля — план «починить абсолютно всё»

> ## ✅ STATUS — EXECUTED + DEPLOYED LIVE (2026-07-23/24)
> Executed via subagent-driven development (fresh implementer + reviewer per task). **All tasks complete except Task 7 (backend calendar fan-out collapse) — intentionally DEFERRED** (perf-only; Task 6 mitigates client-side). Merged to `main` (`bafd694`) + **DEPLOYED LIVE** on `presence.tarski.ru`; GateGuard DB **12 → 13**. Deploy runbook: `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md`.
>
> **Controller override applied:** Task 2 backfill targets `role='admin'` (verified enum literal), NOT the brief's placeholder email list (avoids needing prod-only email data).
>
> **Review caught 1 Critical** (Task 10): the HTTP accept-guard (`http/invitations/handler.go:124`) rejected unverified invitees BEFORE the new verify-on-accept logic ran, making the feature inert — fixed (guard removed + handler-level regression test). The full-suite gate caught a 2nd issue: two `fakeSigner` test doubles missing `MarkEmailVerified` after the interface change — fixed (`bafd694`).
>
> **NOT in this plan — discovered during live browser testing + fixed separately (`a366c61`):** a global **React #418 hydration error** on feed pages. Root cause = nested `<a>` (`EventCard` card-link wrapping `VerifiedBadge`'s organizer-link when a verified organizer is present) — NOT the theme/feed things first guessed. Frontend-only redeploy (`lia-frontend:qa20-r4`). See runbook + `[[lia-demo-deployment]]`.
>
> **⚠️ Remaining follow-up:** `YANDEX_PLACES_KEY` still unprovisioned in prod `.env.prod` → venue-name search (Tasks 8/9) inert until added + `up -d --no-build app`.
>
> ---

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Устранить все проблемы QA-прогона 20 июля (`docs/qa/qa-20-jul-run/analysis.md`) плюс баг «баннер неподтверждённой почты висит у верифицированного/админа».

**Architecture:** Точечные фиксы в трёх сервисах модульного монолита + Next.js фронта: `frontend/` (баннер, SignupCTA, календарные ссылки, поиск площадок), `backend/` (owner-withdraw без верификации, прокси Yandex Places, invite-accept верифицирует почту), `gateguard/` (backfill-миграция, новый RPC для верификации по инвайту). Без Google-OAuth (deep-link) и без обхода для админа (по решению заказчика — только backfill).

**Tech Stack:** Go 1.2x (backend + gateguard, go-pg, gRPC/protobuf), Next.js 16 / React 19 / TypeScript / vitest (frontend), Yandex Geocoder + Places (Search) HTTP API, golang-migrate SQL-миграции.

## Global Constraints

- **Русский язык во всех пользовательских строках.** Технические ошибки не показываем пользователю.
- **`NEXT_PUBLIC_*` требует `ARG` в Dockerfile**, иначе билд-арг молча инлайнится пустым. Новых `NEXT_PUBLIC_*` в этом плане нет — все секреты остаются на бэкенде.
- **GateGuard — источник истины для `email_verified`.** Lia-БД лишь кэш; `email_verified` читается живьём через `CheckAuth` → `UserByUUID` (DB read). Никаких изменений в баковый флаг мимо GateGuard.
- **Секреты только из env на бэкенде.** Новый ключ `YANDEX_PLACES_KEY` браузеру не отдаём.
- **Prod-деплой:** билд на Mac → `save | ssh | load`; `.sql` миграции scp на бокс в `/opt/lia/backend/db/migrations` (Lia) и соответствующий путь gateguard ДО `migrate`; фронт нужен `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru` + `NEXT_PUBLIC_YANDEX_MAPS_KEY`; после деплоя — прунить Docker.
- **Тест-инфра:** backend/gateguard — `go test`; frontend — `vitest` (только чистая логика, без jsdom). UI-правки проверяем `tsc --noEmit` + `eslint`, чистую логику выносим в тестируемые хелперы.

---

## Статус по находкам

| # | Находка | Задача |
|---|---------|--------|
| 6 | Баннер висит у верифицированного/админа (frontend-гонка) | **Task 1 (частично сделано)** + Task 2 |
| 6 | «даже админ» — pre-existing аккаунты неверифицированы | **Task 2** (backfill) |
| 5b | Неподтв. почта мешает «снятым» событиям | **Task 3** (CTA по статусу) + **Task 4** (owner-withdraw) |
| 4a | Нет интеграции с Google Календарём | **Task 5** (deep-link) |
| 4b | Календарь грузится долго | **Task 6** (frontend) + **Task 7** (backend) |
| 2 | Площадки по названию не находятся | **Task 8** (backend Places) + **Task 9** (frontend) |
| 5a | Верификация требуется у пришедшего по ссылке из письма | **Task 10** (invite-accept верифицирует) |
| 1 | 2-е поле регистрации читается как «повтор почты» | **Task 11** (UX-разделитель) |
| — | Деплой + проверка + прунинг | **Task 12** |

> **Уже исправлено ревью #2 (задеплоено 19 июля) — только проверить на текущем prod, НЕ переделывать:** `?next=` после логина (`e76808f`), идентификация строк `/me/invitations` (`ba2538e`), московский bias геокодера (`93062f8`), badge-collision (`4ce43c7`), past-events-first feed (`42876c0`), theme-consistency (`3d8deb7`/`51b1964`).

---

### Task 1: Баннер — гейт по `roleResolved` (ЧАСТИЧНО СДЕЛАНО)

Правка в `frontend/components/VerifyEmailBanner.tsx` уже применена (гейт `!roleResolved` добавлен, `tsc`+`eslint` зелёные). Осталось зафиксировать поведение чистым тестом, вынеся предикат видимости.

**Files:**
- Modify: `frontend/components/VerifyEmailBanner.tsx`
- Create: `frontend/components/verify-banner-visibility.ts`
- Test: `frontend/components/__tests__/verify-banner-visibility.test.ts`

**Interfaces:**
- Produces: `shouldShowVerifyBanner({ ready, isAuthed, roleResolved, emailVerified }: { ready: boolean; isAuthed: boolean; roleResolved: boolean; emailVerified: boolean }): boolean`

- [ ] **Step 1: Write the failing test**

```ts
// frontend/components/__tests__/verify-banner-visibility.test.ts
import { describe, expect, it } from "vitest";
import { shouldShowVerifyBanner } from "@/components/verify-banner-visibility";

describe("shouldShowVerifyBanner", () => {
  const base = { ready: true, isAuthed: true, roleResolved: true, emailVerified: false };
  it("shows for an authed, resolved, unverified user", () => {
    expect(shouldShowVerifyBanner(base)).toBe(true);
  });
  it("hides while /auth/me is still in flight (not resolved) — the reported bug", () => {
    expect(shouldShowVerifyBanner({ ...base, roleResolved: false })).toBe(false);
  });
  it("hides for a verified user", () => {
    expect(shouldShowVerifyBanner({ ...base, emailVerified: true })).toBe(false);
  });
  it("hides before hydration and when signed out", () => {
    expect(shouldShowVerifyBanner({ ...base, ready: false })).toBe(false);
    expect(shouldShowVerifyBanner({ ...base, isAuthed: false })).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run components/__tests__/verify-banner-visibility.test.ts`
Expected: FAIL — module `verify-banner-visibility` not found.

- [ ] **Step 3: Extract the predicate**

```ts
// frontend/components/verify-banner-visibility.ts
/**
 * The banner must wait for /auth/me to SETTLE (roleResolved), not merely for
 * hydration (ready): `ready` flips true at hydration while `emailVerified` is
 * still its default `false`, which made the banner show for verified users on
 * every hard load (and stick if getMe errored).
 */
export function shouldShowVerifyBanner(s: {
  ready: boolean;
  isAuthed: boolean;
  roleResolved: boolean;
  emailVerified: boolean;
}): boolean {
  return s.ready && s.isAuthed && s.roleResolved && !s.emailVerified;
}
```

- [ ] **Step 4: Use the predicate in the component**

```tsx
// frontend/components/VerifyEmailBanner.tsx — replace the guard
import { shouldShowVerifyBanner } from "@/components/verify-banner-visibility";
// ...
export function VerifyEmailBanner() {
  const { isAuthed, ready, roleResolved, emailVerified } = useAuth();
  if (!shouldShowVerifyBanner({ ready, isAuthed, roleResolved, emailVerified })) return null;
  // ...unchanged markup
}
```

- [ ] **Step 5: Run test + typecheck + lint**

Run: `cd frontend && npx vitest run components/__tests__/verify-banner-visibility.test.ts && npx --no-install tsc --noEmit && npx --no-install eslint components/VerifyEmailBanner.tsx components/verify-banner-visibility.ts`
Expected: PASS, tsc clean, eslint exit 0.

- [ ] **Step 6: Commit**

```bash
git add frontend/components/VerifyEmailBanner.tsx frontend/components/verify-banner-visibility.ts frontend/components/__tests__/verify-banner-visibility.test.ts
git commit -m "fix(frontend): gate unverified banner on getMe settling, not hydration"
```

---

### Task 2: Backfill `email_verified=true` для доверенных/админ-аккаунтов (gateguard)

Аккаунты, созданные до появления верификации (включая админ `poulissimo`), имеют `email_verified=false` — баннер для них корректен, но нежелателен. Разовая data-миграция. Обход для админа в коде НЕ делаем (решение заказчика) — только backfill.

**Files:**
- Create: `gateguard/db/000013_backfill_trusted_email_verified.up.sql`
- Create: `gateguard/db/000013_backfill_trusted_email_verified.down.sql`

**Interfaces:** none (SQL only).

- [ ] **Step 1: Determine the trusted email set**

Соберите точный список доверенных email (админ + seed-организаторы). Известный админ: `poulissimo` (уточните полный email в проде). Проверьте на боксе:
```bash
# на vds-ru215, в gateguard Postgres:
# SELECT email, role, email_verified FROM users WHERE email_verified = false;
```

- [ ] **Step 2: Write the up-migration (email-list form — точная, рекомендуется)**

```sql
-- gateguard/db/000013_backfill_trusted_email_verified.up.sql
-- Accounts created before email-verification shipped carry email_verified=false.
-- Grandfather the trusted/seed set so they aren't nagged by the unverified banner.
-- NOTE: replace the address list with the real trusted emails before running.
UPDATE users
SET email_verified = true
WHERE email IN (
    'poulissimo@example.com'   -- admin; REPLACE with the real admin email
    -- , 'seed-org@example.com'
);
```

- [ ] **Step 3: Write the down-migration**

```sql
-- gateguard/db/000013_backfill_trusted_email_verified.down.sql
-- Irreversible in spirit (we don't know prior per-user values); no-op keeps
-- migrate consistent without falsely un-verifying accounts that later verified.
SELECT 1;
```

- [ ] **Step 4: Apply locally and verify**

Run (gateguard local DB): `cd gateguard && migrate -path db -database "$GATEGUARD_DB_URL" up`
Then confirm: the listed users show `email_verified = true`.
Expected: migration `13/u` applied, rows updated.

- [ ] **Step 5: Commit**

```bash
git add gateguard/db/000013_backfill_trusted_email_verified.up.sql gateguard/db/000013_backfill_trusted_email_verified.down.sql
git commit -m "fix(gateguard): backfill email_verified for trusted/admin accounts"
```

---

### Task 3: SignupCTA не показывает активную запись на снятых/отменённых событиях (5b)

`SignupCTA` ветвится только по `signupMode`, игнорируя `event.status`. Для `cancelled`/`rejected` (и любого не-`published`) надо показывать статус, а не активную кнопку, которая уводит в верификацию.

**Files:**
- Create: `frontend/lib/signup-availability.ts`
- Modify: `frontend/components/SignupCTA.tsx`
- Test: `frontend/lib/__tests__/signup-availability.test.ts`

**Interfaces:**
- Consumes: `EventStatus` from `@/lib/types` (`"draft" | "pending_review" | "published" | "rejected" | "cancelled"`).
- Produces: `signupClosedLabel(status: EventStatus): string | null` — RU-строка причины закрытия записи, либо `null` если запись доступна.

- [ ] **Step 1: Write the failing test**

```ts
// frontend/lib/__tests__/signup-availability.test.ts
import { describe, expect, it } from "vitest";
import { signupClosedLabel } from "@/lib/signup-availability";

describe("signupClosedLabel", () => {
  it("open for published", () => {
    expect(signupClosedLabel("published")).toBeNull();
  });
  it("cancelled → отменено", () => {
    expect(signupClosedLabel("cancelled")).toBe("Событие отменено");
  });
  it("rejected → снято модератором", () => {
    expect(signupClosedLabel("rejected")).toBe("Событие снято модератором");
  });
  it("draft / pending_review are not signup-able", () => {
    expect(signupClosedLabel("draft")).toBe("Событие ещё не опубликовано");
    expect(signupClosedLabel("pending_review")).toBe("Событие ещё не опубликовано");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run lib/__tests__/signup-availability.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the helper**

```ts
// frontend/lib/signup-availability.ts
import type { EventStatus } from "@/lib/types";

/** RU reason the signup CTA is unavailable, or null when signup is open. */
export function signupClosedLabel(status: EventStatus): string | null {
  switch (status) {
    case "published":
      return null;
    case "cancelled":
      return "Событие отменено";
    case "rejected":
      return "Событие снято модератором";
    case "draft":
    case "pending_review":
      return "Событие ещё не опубликовано";
  }
}
```

- [ ] **Step 4: Short-circuit in SignupCTA before the signupMode branches**

Insert immediately after the `if (!ready)` SSR-placeholder block (around `SignupCTA.tsx:227`), before `const mode = event.signupMode;`:

```tsx
// Non-published events are not signup-able. Render the status instead of an
// active CTA so an unverified viewer isn't pushed into verification on a
// cancelled/withdrawn event (QA 5b).
const closed = signupClosedLabel(event.status);
if (closed) {
  return (
    <div className="flex flex-col items-end gap-2">
      <span className="text-[15px] font-semibold text-label-secondary">{closed}</span>
      {footer}
    </div>
  );
}
```

Add the import at the top of `SignupCTA.tsx`:
```tsx
import { signupClosedLabel } from "@/lib/signup-availability";
```

- [ ] **Step 5: Run test + typecheck + lint**

Run: `cd frontend && npx vitest run lib/__tests__/signup-availability.test.ts && npx --no-install tsc --noEmit && npx --no-install eslint components/SignupCTA.tsx lib/signup-availability.ts`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add frontend/lib/signup-availability.ts frontend/lib/__tests__/signup-availability.test.ts frontend/components/SignupCTA.tsx
git commit -m "fix(frontend): show status instead of active CTA on non-published events"
```

---

### Task 4: Владелец может отозвать своё событие без верификации почты (5b, backend)

`PATCH /events/{id}` отклоняет ЛЮБОЙ апдейт неподтверждённого пользователя до проверки владельца (`events_update.go:33`). Разрешаем перевод в `cancelled` (отзыв) без верификации — владелец должен уметь снять своё событие.

**Files:**
- Modify: `backend/internal/http/handlers/events_update.go`
- Test: `backend/internal/http/handlers/events_update_test.go` (create if absent)

**Interfaces:**
- Consumes: `formatter.EventPatchToUpdateParams(params.Body) eventsdomain.UpdateParams` with field `Status *string`.

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/http/handlers/events_update_test.go
package handlers

import "testing"

func TestIsWithdrawOnly(t *testing.T) {
    cancelled := "cancelled"
    published := "published"
    cases := []struct {
        name string
        in   *string
        want bool
    }{
        {"cancel is withdraw", &cancelled, true},
        {"publish is not withdraw", &published, false},
        {"nil status is not withdraw", nil, false},
    }
    for _, c := range cases {
        if got := isWithdraw(c.in); got != c.want {
            t.Fatalf("%s: isWithdraw=%v want %v", c.name, got, c.want)
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/handlers/ -run TestIsWithdrawOnly`
Expected: FAIL — `isWithdraw` undefined.

- [ ] **Step 3: Implement `isWithdraw` and rewire the gate**

In `events_update.go`, add the helper and move the verified-gate below the patch computation:

```go
// isWithdraw reports whether the patch sets status to "cancelled" (an owner
// withdrawing their own event). Withdrawing must not require email
// verification — an unverified organizer still needs to be able to pull a
// listing (QA 5b).
func isWithdraw(status *string) bool {
    return status != nil && *status == "cancelled"
}
```

Rewrite the head of `Handle` so the gate consults the patch:

```go
func (h *UpdateEvent) Handle(params eventsops.UpdateEventParams, principal *apimodels.User) middleware.Responder {
    if principal == nil {
        return eventsops.NewUpdateEventUnauthorized().
            WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
    }
    ownerID, err := uuid.FromString(principal.UUID.String())
    if err != nil {
        return eventsops.NewUpdateEventUnauthorized().
            WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
    }
    id, err := uuid.FromString(params.ID.String())
    if err != nil {
        return eventsops.NewUpdateEventBadRequest().
            WithPayload(DefaultError(http.StatusBadRequest, err, nil))
    }

    p := formatter.EventPatchToUpdateParams(params.Body)

    // Verified-gate everything EXCEPT a pure withdraw (status→cancelled); the
    // service still enforces ownership + settable-status, so this only relaxes
    // the email-verification precondition for pulling one's own listing.
    if !isWithdraw(p.Status) && !IsVerified(principal) {
        return UnverifiedResponder()
    }

    updated, err := h.events.Update(params.HTTPRequest.Context(), id, ownerID, p)
    // ...unchanged error switch below
```

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/http/handlers/ -run TestIsWithdrawOnly && go build ./...`
Expected: PASS + build ok.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/handlers/events_update.go backend/internal/http/handlers/events_update_test.go
git commit -m "fix(backend): let owners withdraw events without email verification"
```

---

### Task 5: «В Google Календарь» / Apple / Outlook deep-links (4a)

Сейчас только скачивание `.ics` (`SignupCTA` footer). Добавляем deep-link на Google Calendar (без OAuth) рядом; `.ics` остаётся для Apple/Outlook. Чистый билдер URL — тестируемый.

**Files:**
- Create: `frontend/lib/calendar-links.ts`
- Modify: `frontend/components/SignupCTA.tsx`
- Test: `frontend/lib/__tests__/calendar-links.test.ts`

**Interfaces:**
- Consumes: `LiaEvent` (`title`, `description?`, `startsAt` ISO, `endsAt?` ISO, `venue?.address`/`venue?.name`).
- Produces: `googleCalendarUrl(event: LiaEvent): string`.

- [ ] **Step 1: Write the failing test**

```ts
// frontend/lib/__tests__/calendar-links.test.ts
import { describe, expect, it } from "vitest";
import { googleCalendarUrl } from "@/lib/calendar-links";
import type { LiaEvent } from "@/lib/types";

const ev = {
  id: "1", title: "Bla Bla Meet", description: "Митап",
  categories: [], format: "offline", status: "published",
  startsAt: "2026-08-01T18:00:00Z", endsAt: "2026-08-01T20:00:00Z",
  priceType: "free",
  venue: { id: "v1", name: "Дом Радио", address: "СПб, наб. Мойки, 20" },
} as unknown as LiaEvent;

describe("googleCalendarUrl", () => {
  it("builds a render TEMPLATE link with compact UTC dates", () => {
    const u = new URL(googleCalendarUrl(ev));
    expect(u.origin + u.pathname).toBe("https://calendar.google.com/calendar/render");
    expect(u.searchParams.get("action")).toBe("TEMPLATE");
    expect(u.searchParams.get("text")).toBe("Bla Bla Meet");
    expect(u.searchParams.get("dates")).toBe("20260801T180000Z/20260801T200000Z");
    expect(u.searchParams.get("location")).toContain("Дом Радио");
  });
  it("defaults end to +2h when endsAt is missing", () => {
    const u = new URL(googleCalendarUrl({ ...ev, endsAt: undefined }));
    expect(u.searchParams.get("dates")).toBe("20260801T180000Z/20260801T200000Z");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run lib/__tests__/calendar-links.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the builder**

```ts
// frontend/lib/calendar-links.ts
import type { LiaEvent } from "@/lib/types";

/** Google Calendar wants "YYYYMMDDTHHMMSSZ" in UTC. */
function gcalStamp(iso: string): string {
  return new Date(iso).toISOString().replace(/[-:]/g, "").replace(/\.\d{3}/, "");
}

/**
 * Deep-link that opens a pre-filled event in the user's Google Calendar.
 * No OAuth, no backend — closes the "не интегрируется с Google" gap (QA 4a).
 * .ics (existing "В календарь") still covers Apple/Outlook.
 */
export function googleCalendarUrl(event: LiaEvent): string {
  const start = new Date(event.startsAt);
  const end = event.endsAt
    ? new Date(event.endsAt)
    : new Date(start.getTime() + 2 * 60 * 60 * 1000);
  const location = [event.venue?.name, event.venue?.address].filter(Boolean).join(", ");
  const params = new URLSearchParams({
    action: "TEMPLATE",
    text: event.title,
    dates: `${gcalStamp(start.toISOString())}/${gcalStamp(end.toISOString())}`,
  });
  if (event.description) params.set("details", event.description);
  if (location) params.set("location", location);
  return `https://calendar.google.com/calendar/render?${params.toString()}`;
}
```

- [ ] **Step 4: Add the Google link to the SignupCTA footer**

Modify the `footer` block in `SignupCTA.tsx` (around lines 203–214) to sit alongside the existing `.ics` link:

```tsx
import { googleCalendarUrl } from "@/lib/calendar-links";
// ...
const footer = (
  <div className="flex items-center gap-3">
    <a href={calendarUrl} download className="text-[13px] font-medium text-accent hover:opacity-70">
      В календарь
    </a>
    <a
      href={googleCalendarUrl(event)}
      target="_blank"
      rel="noopener noreferrer"
      className="text-[13px] font-medium text-accent hover:opacity-70"
    >
      В Google
    </a>
    <SeatsCounter event={event} />
  </div>
);
```

- [ ] **Step 5: Run test + typecheck + lint**

Run: `cd frontend && npx vitest run lib/__tests__/calendar-links.test.ts && npx --no-install tsc --noEmit && npx --no-install eslint components/SignupCTA.tsx lib/calendar-links.ts`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add frontend/lib/calendar-links.ts frontend/lib/__tests__/calendar-links.test.ts frontend/components/SignupCTA.tsx
git commit -m "feat(frontend): add Google Calendar deep-link next to the .ics link"
```

---

### Task 6: Календарь — устранить рефетч на каждую навигацию (4b, frontend)

`/me/calendar` рефетчит на каждую смену месяца (React Query key меняется, нет `staleTime`/кэша). Добавляем `staleTime` + `gcTime` и предзагрузку соседних периодов.

**Files:**
- Modify: `frontend/app/me/calendar/page.tsx` (React Query options around lines 119–129)

**Interfaces:** none new (config only).

- [ ] **Step 1: Add staleTime / gcTime to the calendar query**

В `useQuery` для ключа `["calendar", view, rangeStart, rangeEnd]` добавьте:

```tsx
const { data, isLoading } = useQuery({
  queryKey: ["calendar", view, rangeStart, rangeEnd],
  queryFn: () => fetchCalendar(rangeStart, rangeEnd),
  staleTime: 5 * 60 * 1000,   // 5 min: month data rarely changes within a session
  gcTime: 30 * 60 * 1000,     // keep visited months cached for back/forward nav
  placeholderData: (prev) => prev, // keep showing the old month while the next loads
});
```

- [ ] **Step 2: Verify no flicker + typecheck**

Run: `cd frontend && npx --no-install tsc --noEmit && npx --no-install eslint app/me/calendar/page.tsx`
Expected: clean. Manual: navigating month→month→back no longer shows a blank refetch (served from cache).

- [ ] **Step 3: Commit**

```bash
git add frontend/app/me/calendar/page.tsx
git commit -m "perf(frontend): cache calendar months to stop per-navigation refetch"
```

---

### Task 7: Календарь — схлопнуть 3 запроса в один enriched-ход (4b, backend)

`follows/handler.go:153-221` делает 2 range-скана + `GetEnriched` (третий полный запрос) на каждый вызов. Сводим к одному обогащённому проходу над объединённым множеством id.

**Files:**
- Modify: `backend/internal/http/follows/handler.go:153-221`
- Modify: `backend/internal/events/service.go` (если нужен новый метод; см. `ListForCalendar`/`GetEnriched`)
- Test: `backend/internal/http/follows/handler_test.go` (extend)

**Interfaces:**
- Consumes: `events.Service.GetEnriched(ctx, ids []uuid.UUID) ([]Enriched, error)` (существующий).
- Produces: одна функция сбора id (branch A + branch B) с последующим единственным `GetEnriched`.

> **Примечание:** это оптимизация; поведение (набор событий) НЕ меняется — только число запросов. Если объём кода/риск велик, задачу можно отложить: Task 6 уже снимает основную боль пользователя (клиентские рефетчи). Реализуйте только если Task 6 недостаточно на замерах.

- [ ] **Step 1: Add a benchmark/assertion test that the handler issues one enrichment**

Расширьте существующий `handler_test.go`: замокайте `events.Service` так, чтобы считать вызовы `GetEnriched`, и утвердите, что на один `GET /api/v1/me/calendar` приходится ровно один `GetEnriched`.

- [ ] **Step 2: Run to verify current code fails the assertion**

Run: `cd backend && go test ./internal/http/follows/ -run TestCalendarSingleEnrichment`
Expected: FAIL — текущий код зовёт enrich после двух скан-запросов, счётчик > ожидаемого (или структура вызовов иная).

- [ ] **Step 3: Merge the two range scans into one id-set, enrich once**

Соберите объединённое множество id (followed ∪ active-in-range) до обогащения, дедуплицируйте, затем один `GetEnriched(ctx, ids)`. Пометьте каждый id флагами `attending`/`fromFollowed` из скан-фаз (map[uuid]struct{...}) вместо повторного запроса.

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/http/follows/... ./internal/events/... && go build ./...`
Expected: PASS + build ok.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/follows/handler.go backend/internal/events/service.go backend/internal/http/follows/handler_test.go
git commit -m "perf(backend): collapse calendar fan-out to a single enriched query"
```

---

### Task 8: Backend-прокси Yandex Places (Search) API для поиска площадок по названию (2, backend)

Геокодер ищет только адреса. Добавляем прокси к Yandex Places/Search API (`search-maps.yandex.ru/v1/`, `type=biz`) для поиска организаций по названию («Дом Радио»). Отдельный ключ `YANDEX_PLACES_KEY`.

**Files:**
- Modify: `backend/config/init.go` (env bind)
- Modify: `backend/config/scheme.go:126-129` (config field)
- Modify: `backend/internal/geocode/client.go` (add `placesKey` + `SearchPlaces`)
- Modify: `backend/internal/http/geocode/handler.go` (interface + `/api/v1/places` route)
- Modify: `backend/internal/http/module.go` (inject places key, mount route)
- Test: `backend/internal/geocode/client_test.go` (extend), `backend/internal/http/geocode/handler_test.go` (extend)

**Interfaces:**
- Produces: `(*geocode.Client).SearchPlaces(ctx context.Context, q string) ([]geocode.Result, error)` — Result reuses `{Lat, Lon, Label}`; `Label` = «name · address».
- Produces: `(*geocode.Client).WithPlacesKey(k string) *geocode.Client` (chainable setter; preserves existing `NewClient(key)` signature and its tests).
- Produces: HTTP `GET /api/v1/places?q=...` (auth-gated) → `[]Result`.

- [ ] **Step 1: Write the failing client test (httptest upstream)**

```go
// backend/internal/geocode/client_test.go — add
func TestSearchPlacesParsesBusinesses(t *testing.T) {
    body := `{"features":[
      {"geometry":{"coordinates":[30.316,59.933]},
       "properties":{"name":"Дом Радио","description":"Санкт-Петербург",
       "CompanyMetaData":{"name":"Дом Радио","address":"наб. реки Мойки, 20"}}}
    ]}`
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Query().Get("type") != "biz" { t.Errorf("type=%q", r.URL.Query().Get("type")) }
        _, _ = w.Write([]byte(body))
    }))
    defer srv.Close()
    c := NewClient("").WithPlacesKey("k")
    c.placesEndpoint = srv.URL // test hook
    got, err := c.SearchPlaces(context.Background(), "Дом Радио")
    if err != nil { t.Fatal(err) }
    if len(got) != 1 || got[0].Label != "Дом Радио · наб. реки Мойки, 20" {
        t.Fatalf("got %+v", got)
    }
    if got[0].Lat != 59.933 || got[0].Lon != 30.316 {
        t.Fatalf("coords %+v", got[0])
    }
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/geocode/ -run TestSearchPlacesParsesBusinesses`
Expected: FAIL — `WithPlacesKey`/`SearchPlaces`/`placesEndpoint` undefined.

- [ ] **Step 3: Implement the Places client**

In `client.go` add fields and method:

```go
const defaultPlacesEndpoint = "https://search-maps.yandex.ru/v1/"

// (add to Client struct)
//   placesKey      string
//   placesEndpoint string

// NewClient sets placesEndpoint to the default (see WithPlacesKey to enable).
// In NewClient(...): endpoint/placesEndpoint defaults + http client as today.

// WithPlacesKey enables the Places (Search) API with its own key and returns c.
func (c *Client) WithPlacesKey(k string) *Client { c.placesKey = k; return c }

// placesResponse mirrors the subset of the Yandex Search API v1 GeoJSON we read.
type placesResponse struct {
    Features []struct {
        Geometry struct {
            Coordinates []float64 `json:"coordinates"` // [lon, lat]
        } `json:"geometry"`
        Properties struct {
            CompanyMetaData struct {
                Name    string `json:"name"`
                Address string `json:"address"`
            } `json:"CompanyMetaData"`
        } `json:"properties"`
    } `json:"features"`
}

// SearchPlaces resolves a venue/organization NAME (e.g. "Дом Радио") to points,
// biased to Moscow. Blank query → empty slice, no HTTP call.
func (c *Client) SearchPlaces(ctx context.Context, q string) ([]Result, error) {
    q = strings.TrimSpace(q)
    if q == "" {
        return []Result{}, nil
    }
    if c.placesKey == "" {
        return nil, errors.New("places: api key not configured")
    }
    params := url.Values{
        "apikey":  {c.placesKey},
        "text":    {q},
        "type":    {"biz"},
        "lang":    {"ru_RU"},
        "results": {"5"},
        "ll":      {moscowLL},
        "spn":     {moscowSpn},
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.placesEndpoint+"?"+params.Encode(), nil)
    if err != nil {
        return nil, err
    }
    res, err := c.http.Do(req)
    if err != nil {
        return nil, err
    }
    defer func() { _ = res.Body.Close() }()
    if res.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("places: upstream status %d", res.StatusCode)
    }
    var pr placesResponse
    if err := json.NewDecoder(res.Body).Decode(&pr); err != nil {
        return nil, err
    }
    out := make([]Result, 0, len(pr.Features))
    for _, f := range pr.Features {
        if len(f.Geometry.Coordinates) != 2 {
            continue
        }
        label := f.Properties.CompanyMetaData.Name
        if a := f.Properties.CompanyMetaData.Address; a != "" {
            label = label + " · " + a
        }
        out = append(out, Result{
            Lon:   f.Geometry.Coordinates[0],
            Lat:   f.Geometry.Coordinates[1],
            Label: label,
        })
    }
    return out, nil
}
```
(Set `placesEndpoint: defaultPlacesEndpoint` inside `NewClient`.)

- [ ] **Step 4: Extend the HTTP handler with `/api/v1/places`**

In `handler.go`:
```go
type Geocoder interface {
    Geocode(ctx context.Context, q string) ([]geo.Result, error)
    SearchPlaces(ctx context.Context, q string) ([]geo.Result, error)
}

// in NewHandler:
h.mux.HandleFunc("GET /api/v1/places", h.places)

func (h *handler) places(w http.ResponseWriter, r *http.Request) {
    if h.principal(r) == nil {
        writeErr(w, http.StatusUnauthorized, "unauthorized")
        return
    }
    results, err := h.deps.Client.SearchPlaces(r.Context(), r.URL.Query().Get("q"))
    if err != nil {
        writeErr(w, http.StatusServiceUnavailable, "places_failed")
        return
    }
    writeJSON(w, http.StatusOK, results)
}
```

- [ ] **Step 5: Config + module wiring**

`config/scheme.go` — add to `GeocoderConfig`:
```go
type GeocoderConfig struct {
    Key       string `mapstructure:"key"`
    PlacesKey string `mapstructure:"places_key"`
}
```
`config/init.go` — after the geocoder.key binds (line 127):
```go
viper.SetDefault("geocoder.places_key", "")
viper.BindEnv("geocoder.places_key", "YANDEX_PLACES_KEY") //nolint:errcheck
```
`module.go` — add field + setter + inject:
```go
// field: placesKey string
// SetPlaces injects the Yandex Places (Search) API key. Call before Init.
func (m *Module) SetPlaces(key string) { m.placesKey = key }
// at construction (line ~390):
Client: geo.NewClient(m.geocoderKey).WithPlacesKey(m.placesKey),
// router: mount alongside /api/v1/geocode
if p == "/api/v1/places" {
    geocodeH.ServeHTTP(w, r)
    return
}
```
Find the call site of `SetGeocoder(...)` (where the app wires config → module) and add `mod.SetPlaces(cfg.Geocoder.PlacesKey)` next to it.

- [ ] **Step 6: Run tests + build**

Run: `cd backend && go test ./internal/geocode/... ./internal/http/geocode/... && go build ./...`
Expected: PASS + build ok.

- [ ] **Step 7: Document the env var**

Add to `backend/.env.prod.example`: `YANDEX_PLACES_KEY=` with a comment «Yandex Places/Search API key (venue-name search)».

- [ ] **Step 8: Commit**

```bash
git add backend/config/init.go backend/config/scheme.go backend/internal/geocode/client.go backend/internal/http/geocode/handler.go backend/internal/http/module.go backend/.env.prod.example backend/internal/geocode/client_test.go backend/internal/http/geocode/handler_test.go
git commit -m "feat(backend): proxy Yandex Places API for venue-name search (/api/v1/places)"
```

---

### Task 9: Frontend — поиск площадки по названию + мёрдж с адресами (2, frontend)

`VenueGeoModal` сейчас зовёт только `geocodeAddress`. Добавляем `searchPlaces` и объединяем: сначала организации (по названию), затем адреса.

**Files:**
- Modify: `frontend/lib/geocode.ts` (add `searchPlaces`)
- Modify: `frontend/components/VenueGeoModal.tsx` (call both, merge, relabel input)
- Test: `frontend/lib/__tests__/geocode.test.ts` (extend — mock fetch)

**Interfaces:**
- Consumes: backend `GET /api/v1/places?q=...` → `GeoResult[]`.
- Produces: `searchPlaces(q: string): Promise<GeoResult[]>`.

- [ ] **Step 1: Extend the geocode test to cover searchPlaces**

Add a case mirroring the existing `geocodeAddress` test but for `/api/v1/places`, asserting the URL and that results pass through.

- [ ] **Step 2: Run to verify it fails**

Run: `cd frontend && npx vitest run lib/__tests__/geocode.test.ts`
Expected: FAIL — `searchPlaces` not exported.

- [ ] **Step 3: Implement `searchPlaces`**

```ts
// frontend/lib/geocode.ts — add
/** Venue/organization NAME search via the auth-gated backend Yandex Places proxy. */
export async function searchPlaces(q: string): Promise<GeoResult[]> {
  const query = q.trim();
  if (query === "") return [];
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const res = await fetch(`${API_BASE}/api/v1/places?q=${encodeURIComponent(query)}`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`places failed: ${res.status}`);
  return (await res.json()) as GeoResult[];
}
```

- [ ] **Step 4: Merge both sources in VenueGeoModal**

Replace the single-source effect (lines 38–47) so it queries both and concatenates (places first — the QA case is name search), de-dupledby label:

```tsx
import { geocodeAddress, searchPlaces, type GeoResult } from "@/lib/geocode";
// ...
useEffect(() => {
  if (debounced.trim() === "") return;
  let live = true;
  Promise.allSettled([searchPlaces(debounced), geocodeAddress(debounced)]).then(
    ([places, addrs]) => {
      if (!live) return;
      const p = places.status === "fulfilled" ? places.value : [];
      const a = addrs.status === "fulfilled" ? addrs.value : [];
      const seen = new Set<string>();
      const merged = [...p, ...a].filter((r) => {
        if (seen.has(r.label)) return false;
        seen.add(r.label);
        return true;
      });
      setResults(merged);
    },
  );
  return () => {
    live = false;
  };
}, [debounced]);
```

Relabel the input + footer to reflect name-or-address:
```tsx
placeholder="Название или адрес"
// footer:
Поиск по названию и адресу — © Яндекс. Перетащите метку для точности.
```

- [ ] **Step 5: Run test + typecheck + lint**

Run: `cd frontend && npx vitest run lib/__tests__/geocode.test.ts && npx --no-install tsc --noEmit && npx --no-install eslint lib/geocode.ts components/VenueGeoModal.tsx`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add frontend/lib/geocode.ts frontend/lib/__tests__/geocode.test.ts frontend/components/VenueGeoModal.tsx
git commit -m "feat(frontend): search venues by name (Places) alongside address geocoding"
```

---

### Task 10: Приём приглашения по ссылке из письма подтверждает почту (5a)

Приглашение уходит на конкретный адрес (`invRow.InviteeEmail`); приём его с совпадающего аккаунта — доказательство владения почтой. Значит приём инвайта должен ПОДТВЕРЖДАТЬ почту, а не требовать её заранее. `email_verified` — за GateGuard, поэтому Lia-бэкенд просит GateGuard пометить адрес через новый доверенный RPC.

> **Самая тяжёлая задача:** меняет `.proto` → требует protobuf-codegen в двух местах (`gateguard/protocols/gateguard`, `backend/protocols/gateguard`). Убедитесь, что установлен `protoc`/`buf` тулчейн проекта до старта.

**Files:**
- Modify: `gateguard/protocols/gateguard/service_gateguard.proto` (new RPC)
- Regenerate: `gateguard/protocols/gateguard/*.pb.go` **и** `backend/protocols/gateguard/*.pb.go`
- Modify: `gateguard/internal/server/auth_password.go` (handler)
- Modify: `gateguard/internal/service/email_verification.go` (service method)
- Modify: `gateguard/internal/service/interface.go` (interface)
- Modify: `backend/internal/http/auth/signer.go` (new signer method + interface)
- Modify: `backend/internal/invitations/service.go:205-219` (verify-on-accept)
- Modify: wiring where `invitations.NewService(...)` is constructed (inject verifier)
- Modify: `frontend/app/invite/[token]/page.tsx` (drop the `!emailVerified` guard branch)
- Test: `gateguard/internal/service/tests/email_verification_test.go`, `backend/internal/invitations/service_test.go`

**Interfaces:**
- Produces (proto): `rpc MarkEmailVerified(EmailRequest) returns(Empty);` — trusted internal RPC that sets `email_verified=true` by email.
- Produces (gateguard service): `(*UsersService).MarkEmailVerified(ctx, email string) error`.
- Produces (backend signer): `Signer.MarkEmailVerified(ctx context.Context, email string) error`.
- Consumes (invitations): new dep `EmailVerifier interface { MarkEmailVerified(ctx, email string) error }` on the invitations service.

- [ ] **Step 1: Add the RPC to the proto**

In `service_gateguard.proto`, after `VerifyEmail` (line 30):
```proto
// MarkEmailVerified (trusted, internal) flips email_verified=true for an email
// without a code. Called by Lia when a user accepts an event invitation that
// was emailed to that same address — the accept proves ownership.
rpc MarkEmailVerified(EmailRequest) returns(Empty);
```

- [ ] **Step 2: Regenerate protobuf on both sides**

Run the project's codegen (confirm the exact command from the repo Makefile/buf config), e.g.:
```bash
cd gateguard && make proto   # or: buf generate
# then sync/regenerate the backend copy under backend/protocols/gateguard
```
Expected: `MarkEmailVerified` appears in both `gateguard/protocols/gateguard/service_gateguard_grpc.pb.go` and `backend/protocols/gateguard/*_grpc.pb.go`.

- [ ] **Step 3: Write the failing gateguard service test**

Extend `email_verification_test.go`: a user with `email_verified=false` → `MarkEmailVerified(ctx, email)` → reload → `email_verified == true`; unknown email → error.

- [ ] **Step 4: Run to verify it fails**

Run: `cd gateguard && go test ./internal/service/... -run MarkEmailVerified`
Expected: FAIL — method undefined.

- [ ] **Step 5: Implement gateguard service + handler + interface**

`email_verification.go`:
```go
// MarkEmailVerified flips email_verified=true for an already-existing account.
// Trusted path: the caller (Lia) has proven the user owns the address (invite
// accept). No token/code required.
func (u *UsersService) MarkEmailVerified(ctx context.Context, email string) error {
    user := &models.User{Email: email}
    if err := u.repository.GetUser(ctx, user, repository.Email); err != nil {
        return fmt.Errorf("lookup user %s: %w", email, err)
    }
    user.EmailVerified = true
    if err := u.repository.UpdateUserBy(ctx, user, repository.Email, "email_verified"); err != nil {
        return fmt.Errorf("mark verified %s: %w", email, err)
    }
    return nil
}
```
Add to `interface.go` and implement the handler in `auth_password.go`:
```go
func (h *GateguardHandlers) MarkEmailVerified(ctx context.Context, req *proto.EmailRequest) (*proto.Empty, error) {
    if err := h.srv.MarkEmailVerified(ctx, req.Email); err != nil {
        h.log.ErrorCtx(ctx, err, "Failed to mark email verified")
        return nil, fmt.Errorf("mark email verified: %w", err)
    }
    return &proto.Empty{}, nil
}
```

- [ ] **Step 6: Run gateguard tests**

Run: `cd gateguard && go test ./internal/service/... -run MarkEmailVerified && go build ./...`
Expected: PASS + build.

- [ ] **Step 7: Add the backend signer method**

`signer.go`: extend the `Signer` interface + implement:
```go
// MarkEmailVerified asks GateGuard to flip email_verified for an address the
// caller has already proven ownership of (invite accept).
MarkEmailVerified(ctx context.Context, email string) error
```
```go
func (s *gatekeeperSigner) MarkEmailVerified(ctx context.Context, email string) error {
    if s.cfg.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, s.cfg.Timeout)
        defer cancel()
    }
    if _, err := s.client.MarkEmailVerified(ctx, &gg.EmailRequest{Email: email}); err != nil {
        return fmt.Errorf("gateguard mark verified: %w", err)
    }
    return nil
}
```
(Update the `ggClient` interface/mocks used by signer tests to include `MarkEmailVerified`.)

- [ ] **Step 8: Verify-on-accept in the invitations service**

Add an `EmailVerifier` dep to the invitations service and rewrite `accept()` so the email-match is checked FIRST, and an unverified-but-matching invitee is verified rather than rejected:
```go
type EmailVerifier interface {
    MarkEmailVerified(ctx context.Context, email string) error
}
// add `verifier EmailVerifier` to the service struct + constructor.

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
        if err := s.verifier.MarkEmailVerified(ctx, invRow.InviteeEmail); err != nil {
            return fmt.Errorf("verify invitee on accept: %w", err)
        }
    }
    if err := s.rsvp.SignUp(ctx, invRow.EventID, userID, ""); err != nil {
        return fmt.Errorf("rsvp on accept: %w", err)
    }
    return s.repo.SetStatus(ctx, invRow.ID, "accepted")
}
```
Inject the signer as the verifier where `invitations.NewService(...)` is wired (the signer already reaches GateGuard). Update `service_test.go` with a fake verifier; assert an unverified matching invitee is accepted AND `MarkEmailVerified` was called.

- [ ] **Step 9: Drop the frontend pre-accept verification wall**

In `frontend/app/invite/[token]/page.tsx`, remove the `!emailVerified` branch (lines 49–53) so an authed invitee reaches `onAccept()` directly; the backend now verifies on accept. Keep the not-signed-in branch. Also drop the now-dead `router.push("/auth/verify")` on `EMAIL_NOT_VERIFIED` (it should no longer occur for a matching invitee; leave a generic error fallback).

- [ ] **Step 10: Run all affected tests + builds**

Run:
```bash
cd gateguard && go test ./... && go build ./...
cd ../backend && go test ./internal/invitations/... ./internal/http/auth/... && go build ./...
cd ../frontend && npx --no-install tsc --noEmit && npx --no-install eslint app/invite/\[token\]/page.tsx
```
Expected: all PASS/clean.

- [ ] **Step 11: Commit**

```bash
git add gateguard/protocols backend/protocols gateguard/internal backend/internal/http/auth/signer.go backend/internal/invitations frontend/app/invite
git commit -m "feat: accepting an emailed invitation verifies the invitee's email (5a)"
```

---

### Task 11: Регистрация — визуально отделить «Имя» от блока учётных данных (1)

Поле «Имя (необязательно)» стоит вплотную под Email и читается как «повтор почты». Не добавляем повтор email (его нет by design) — добавляем разделитель/подпись, чтобы порядок читался однозначно.

**Files:**
- Modify: `frontend/components/AuthButton.tsx` (`LoginModal`, register branch ~lines 179–203)

**Interfaces:** none.

- [ ] **Step 1: Reorder + label so credentials read unambiguously**

Поставьте пароль сразу под email (учётные данные вместе), «Имя» — последним, с явной подписью-подсказкой; либо оставьте порядок, но добавьте разделитель и хинт под email. Пример (перестановка): Email → Пароль → «Имя (необязательно)» с подписью «Как вас представить другим участникам»:

```tsx
{isRegister && (
  <label className="mb-3 block">
    <span className="mb-1 block text-[13px] text-label-secondary">
      Имя (необязательно)
    </span>
    <input
      value={name}
      onChange={(e) => setName(e.target.value)}
      placeholder="Как вас представить участникам"
      className={inputClass}
    />
  </label>
)}
```
Убедитесь, что порядок в JSX: Email → Пароль → Имя (перенесите блок «Имя» под пароль).

- [ ] **Step 2: Typecheck + lint + manual**

Run: `cd frontend && npx --no-install tsc --noEmit && npx --no-install eslint components/AuthButton.tsx`
Expected: clean. Manual: второе поле теперь пароль, «Имя» внизу с поясняющей подписью — прочитать его как «повтор почты» невозможно.

- [ ] **Step 3: Commit**

```bash
git add frontend/components/AuthButton.tsx
git commit -m "fix(frontend): reorder signup fields so name isn't mistaken for confirm-email"
```

---

### Task 12: Деплой на prod + проверка + прунинг

**Files:** none (ops). Соберите билды и выкатите на `vds-ru215` (193.32.188.7).

- [ ] **Step 1: Full local verification gate**

Run:
```bash
cd backend && go build ./... && go test ./...
cd ../gateguard && go build ./... && go test ./...
cd ../frontend && npx vitest run && npx --no-install tsc --noEmit && npx --no-install eslint
```
Expected: all PASS/clean. НЕ деплоить, пока хоть один шаг красный.

- [ ] **Step 2: Provision the new secret**

Add `YANDEX_PLACES_KEY=<key>` to `/opt/lia/backend/.env.prod` on the box.

- [ ] **Step 3: Ship migrations**

`scp` `gateguard/db/000013_*.sql` в путь gateguard-миграций на боксе; примените (gateguard `migrate up`) ПОСЛЕ деплоя gateguard-образа. (Lia-миграций в этом плане нет.)

- [ ] **Step 4: Build + load images (build-on-Mac → save|ssh|load)**

Соберите backend, gateguard и frontend образы. **Frontend build-args (обязательно оба):** `NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru` и `NEXT_PUBLIC_YANDEX_MAPS_KEY=<key>`. Backend recreate — все 4 compose-файла + `--no-build`.

- [ ] **Step 5: Live smoke-verify each fix**

- Баннер: залогиньтесь верифицированным аккаунтом → жёсткий reload → баннера НЕТ (Task 1). Админ после backfill → баннера НЕТ (Task 2).
- Снятое событие: откройте `cancelled`-событие → «Событие отменено», активной кнопки записи нет, верификация не всплывает (Task 3).
- Owner-withdraw: неподтверждённым владельцем отзовите своё событие → успех (Task 4).
- Google: на событии есть «В Google» → открывает предзаполненный Google Calendar (Task 5).
- Календарь: навигация месяц→месяц→назад без пустого рефетча (Task 6).
- Площадки: в пикере введите «Дом Радио» → находится по названию (Task 8/9).
- Инвайт: примите приглашение неподтверждённым аккаунтом с совпадающим email → принято, почта стала подтверждённой, баннер исчез (Task 10).
- Регистрация: второе поле — пароль, «Имя» внизу с подписью (Task 11).
- **Регрессия ревью #2:** проверьте, что `?next=`, идентификация инвайтов, badge, feed-порядок, темы — по-прежнему в порядке.

- [ ] **Step 6: Prune Docker on the box**

Run on box: `docker builder prune -f && docker image prune -f` и обрежьте `rollback-*` до последних ~3 (диск 20 ГБ).

- [ ] **Step 7: Update HANDOFF + memory**

Отметьте в `HANDOFF.md` выкат; обновите память (`lia-review2-major-fixes` / `lia-project-state`): поправьте формулировку про баннер — «email_verified читается живьём из БД (GateGuard CheckAuth→UserByUUID); симптом был frontend-гонкой в `VerifyEmailBanner`, чинится гейтом на `roleResolved`; НЕ stale-JWT».

---

## Self-Review

**Spec coverage:** каждая находка QA (1, 2, 4a, 4b, 5a, 5b) + баг баннера (#6) → Tasks 1–11; деплой → Task 12. Carried-over review#2 items помечены verify-only (Task 12 Step 5). ✔

**Placeholder scan:** Task 2 требует подставить реальные admin-email (помечено); Task 7 помечена как опциональная оптимизация с benchmark-first; proto-codegen команда в Task 10 требует подтверждения по Makefile — все явно отмечены как «уточнить/подтвердить», кода-заглушек нет. ✔

**Type consistency:** `Result{Lat,Lon,Label}` переиспользован Geocoder+Places; `GeoResult` на фронте едина для обоих; `Signer.MarkEmailVerified` ↔ gateguard `MarkEmailVerified(EmailRequest)→Empty` ↔ `EmailVerifier.MarkEmailVerified`; `signupClosedLabel`/`shouldShowVerifyBanner`/`googleCalendarUrl` согласованы между задачей и тестом. ✔
</content>
