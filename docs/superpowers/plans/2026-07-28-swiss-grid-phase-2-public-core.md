# Swiss Grid Phase 2 — Public Core (U1 feed, U2 detail, U7 auth, U8 states) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restyle the public core — feed (`/`), event detail (`/events/[id]`), dedicated login/signup pages, and every designed empty/gated/error state — to the Swiss Grid system on top of the Phase 1 foundation, and fold in the surviving hygiene items (category taxonomy from API, «Читательские группы» migration).

**Architecture:** Screens consume the Phase 1 primitives (`AppHeader`, `Chip`, `Button`, `Cell/CellStrip`, `EventModule`, `Field`, `EmptyState`, `Skeleton`) and three new pure helpers (module date, short attendance, RU plural). Data layer unchanged (`lib/api.ts`, TanStack Query, `useAuth`). `EventCard`/`GlassNav`/old `TabBar` remain for not-yet-migrated screens; this phase removes their usage only from the pages it touches. Phase 1+2 deploy together at the end of this plan.

**Tech Stack:** Next.js 16 App Router / React 19 / TS, Tailwind v4 (Swiss Grid tokens from Phase 1), TanStack Query 5, Vitest 4 (node-only), Go backend (one SQL migration), Yandex Maps v2.1.

**Prerequisite:** local `main` at `5c9a3a4` (Phase 1 merged). Execute in an isolated worktree branched from it.

## Global Constraints (inherit Phase 1's; these are the Phase-2-specific ones)

- Handoff spec: `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` — screens U1 (`/events`→our `/`), U2, U7, U8. Visual reference: `Presence Swiss Grid - Full System.dc.html` (search badge strings "U1", "U2", "U7", "U8").
- Zero radius, zero shadows, 1px hairlines, hover inverts, 2px square focus, 120ms linear; categories = numerals via `categoryNumeral()`; all numbers JetBrains Mono (`font-mono`); price strings only via `priceLabel()` (`FREE` literal).
- Every empty/loading/gated/error surface designed: `EmptyState` / `Skeleton` (`#ECEAE4` boxes at final dimensions, numbers show `—`), never spinners, never blank divs.
- U8 404 is **inverted to ink** (`data-surface="ink"`).
- Preserve: React #418 single-`<Link>`/no-nested-`<a>` rule, Europe/Moscow-pinned formatters (all new date/time helpers pin `timeZone: "Europe/Moscow"`), `?next=` open-redirect guard (app-internal single-leading-`/` paths only).
- Preserve existing behaviours: SSR feed with mock fallback, TanStack Query cache keys, near-me geolocation mode, RSVP logic in `SignupCTA` (restyle only), owner-draft fallback on detail.
- **Deliberate deviations from the handoff (pre-decided, do not "fix" backwards):** (a) feed keeps the search input (existing feature; handoff U1 has none) styled as a Swiss field; (b) U2 cover strip renders ONE cover image full-width (we have one cover per event, not two); (c) near-me chip stays in the time-filter group; (d) «Бесплатно» time-group chip is added per handoff and filters `priceType === "free"` client-side.
- Category chips on the feed come from `getCategories()` (`ApiCategory { id, slug, label }`, `lib/api.ts:141-154`) — never hard-coded (this completes the old taxonomy bug fix).
- All UI copy Russian; code/commits English. TEMP Phase-1 shims in `globals.css` are removed only for classes this phase's pages stop using — the full shim removal happens in Phase 7 (other screens still consume them).
- Every commit: `pnpm build && pnpm test && pnpm lint` green (run from `frontend/`).

## File Structure

```
frontend/
  lib/format.ts                         MODIFY — add formatModuleDate, formatStartTime, attendanceShort
  lib/plural.ts                         CREATE — pluralRu()
  lib/event-module.ts                   CREATE — eventToModuleProps()
  lib/__tests__/{format-module,plural,event-module}.test.ts  CREATE
  app/not-found.tsx                     CREATE — U8 404, inverted
  components/ui/AuthGate.tsx            CREATE — U8 auth-required pattern
  components/AuthForm.tsx               CREATE — shared login/register form internals (Swiss)
  app/login/page.tsx                    CREATE — U7 split page
  app/signup/page.tsx                   CREATE — U7 registration variant
  components/AuthButton.tsx             MODIFY — LoginModal wraps AuthForm; nav control restyled
  components/ui/AuthNavControl.tsx      CREATE — AppHeader actions slot (Войти / email+Выйти)
  app/page.tsx                          MODIFY — SSR events+categories, AppHeader
  components/DiscoveryFeed.tsx          MODIFY (rewrite) — U1 layout
  components/EventDetailView.tsx        MODIFY (rewrite) — U2 layout
  components/SignupCTA.tsx              MODIFY — class-level restyle only
  components/VenueMap.tsx               MODIFY — desaturated container + square treatment
  app/me/calendar/page.tsx, app/events/mine/page.tsx, app/me/organizer/page.tsx,
  app/organizer/applications/page.tsx   MODIFY — logged-out prompt → AuthGate (block swap only)
backend/db/migrations/
  000021_reading_group_category.up.sql  CREATE
  000021_reading_group_category.down.sql CREATE
frontend/lib/covers.ts                  MODIFY — add reading-group entry
frontend/components/ui/CategoryGlyph.tsx MODIFY — add reading-group glyph mapping
```

---

### Task 1: Pure helpers — module date/time, short attendance, RU plural (TDD)

**Files:**
- Modify: `frontend/lib/format.ts` (append; do not touch existing exports)
- Create: `frontend/lib/plural.ts`, `frontend/lib/event-module.ts`
- Test: `frontend/lib/__tests__/format-module.test.ts`, `frontend/lib/__tests__/plural.test.ts`, `frontend/lib/__tests__/event-module.test.ts`

**Interfaces:**
- Consumes: `categoryNumeral(slug, ordered)` (`lib/category-numerals.ts`), `priceLabel(priceRub, kind)` (`lib/price-label.ts`), `LiaEvent` (`lib/types.ts:59`), `ApiCategory` (`lib/api.ts:141`).
- Produces: `formatModuleDate(startsAt: string, endsAt?: string): string` — `"12.07 · 16:00"` single-day; `"15.08 – 17.08"` multi-day (Moscow civil days; reuses the zero-time "no end" rule: `endsAt` year ≤ 1 means none).
- Produces: `formatStartTime(iso: string): string` — `"16:00"` Moscow.
- Produces: `attendanceShort(event: Pick<LiaEvent,"attendeeCount"|"capacity">): string` — `"12 / 40"` | `"64"` | `"—"`.
- Produces: `pluralRu(n: number, forms: [string, string, string]): string` — `pluralRu(42, ["событие","события","событий"])` → `"события"` (1→[0], 2–4→[1], 0/5–20→[2], respecting 11–14).
- Produces: `eventToModuleProps(event: LiaEvent, categories: ReadonlyArray<{slug: string}>): { numeral: string; category: string; title: string; venue: string; date: string; price: string; href: string }`.

- [ ] **Step 1: Write the failing tests**

`frontend/lib/__tests__/plural.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { pluralRu } from "../plural";

const FORMS: [string, string, string] = ["событие", "события", "событий"];

describe("pluralRu", () => {
  it("singular", () => {
    expect(pluralRu(1, FORMS)).toBe("событие");
    expect(pluralRu(21, FORMS)).toBe("событие");
  });
  it("few", () => {
    expect(pluralRu(2, FORMS)).toBe("события");
    expect(pluralRu(42, FORMS)).toBe("события");
  });
  it("many + teens", () => {
    expect(pluralRu(0, FORMS)).toBe("событий");
    expect(pluralRu(5, FORMS)).toBe("событий");
    expect(pluralRu(11, FORMS)).toBe("событий");
    expect(pluralRu(14, FORMS)).toBe("событий");
    expect(pluralRu(111, FORMS)).toBe("событий");
  });
});
```

`frontend/lib/__tests__/format-module.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { attendanceShort, formatModuleDate, formatStartTime } from "../format";

describe("formatModuleDate", () => {
  it("single day → DD.MM · HH:mm (Moscow)", () => {
    expect(formatModuleDate("2026-07-12T13:00:00Z")).toBe("12.07 · 16:00");
  });
  it("zero end is ignored", () => {
    expect(formatModuleDate("2026-07-12T13:00:00Z", "0001-01-01T00:00:00Z")).toBe(
      "12.07 · 16:00",
    );
  });
  it("same-civil-day end keeps single form", () => {
    expect(
      formatModuleDate("2026-07-12T13:00:00Z", "2026-07-12T18:00:00Z"),
    ).toBe("12.07 · 16:00");
  });
  it("multi-day → DD.MM – DD.MM", () => {
    expect(
      formatModuleDate("2026-08-15T09:00:00Z", "2026-08-17T18:00:00Z"),
    ).toBe("15.08 – 17.08");
  });
});

describe("formatStartTime", () => {
  it("Moscow wall-clock time", () => {
    expect(formatStartTime("2026-07-12T13:00:00Z")).toBe("16:00");
  });
});

describe("attendanceShort", () => {
  it("count / capacity", () => {
    expect(attendanceShort({ attendeeCount: 12, capacity: 40 })).toBe("12 / 40");
  });
  it("count only", () => {
    expect(attendanceShort({ attendeeCount: 64 })).toBe("64");
  });
  it("nothing known → em dash", () => {
    expect(attendanceShort({})).toBe("—");
  });
});
```

`frontend/lib/__tests__/event-module.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { eventToModuleProps } from "../event-module";
import type { LiaEvent } from "../types";

const CATS = [{ slug: "festival" }, { slug: "mediation" }, { slug: "lecture" }];

const EVENT = {
  id: "evt-1",
  title: "Медиация по выставке «Свет»",
  categories: [{ id: "c2", slug: "mediation", label: "Медиации" }],
  format: "offline",
  status: "published",
  startsAt: "2026-07-12T13:00:00Z",
  priceType: "free",
} as LiaEvent;

describe("eventToModuleProps", () => {
  it("maps every module field", () => {
    expect(eventToModuleProps(EVENT, CATS)).toEqual({
      numeral: "02",
      category: "Медиации",
      title: "Медиация по выставке «Свет»",
      venue: "Онлайн",
      date: "12.07 · 16:00",
      price: "FREE",
      href: "/events/evt-1",
    });
  });
  it("venue name wins; offline without venue → —", () => {
    expect(
      eventToModuleProps(
        { ...EVENT, venue: { id: "v", name: "Винзавод" } } as LiaEvent,
        CATS,
      ).venue,
    ).toBe("Винзавод");
    expect(
      eventToModuleProps({ ...EVENT, format: "offline" } as LiaEvent, CATS).venue,
    ).toBe("Онлайн"); // format online path
  });
  it("paid + from prices go through priceLabel", () => {
    expect(
      eventToModuleProps(
        { ...EVENT, priceType: "from", priceMin: 1500 } as LiaEvent,
        CATS,
      ).price,
    ).toBe("от 1 500 ₽"); // NB: NBSP group separator from Intl
  });
  it("no categories → numeral —, category —", () => {
    const p = eventToModuleProps({ ...EVENT, categories: [] } as LiaEvent, CATS);
    expect(p.numeral).toBe("—");
    expect(p.category).toBe("—");
  });
});
```

(Adjust the second test's expectations while implementing so the venue rule is: `event.venue?.name ?? (event.format === "online" ? "Онлайн" : "—")` — same rule `EventDetailView` uses today; the test above must assert exactly that rule with three cases: named venue, online-no-venue → "Онлайн", offline-no-venue → "—". The «от 1 500 ₽» literal contains U+00A0 between 1 and 5 — copy it from `price-label.test.ts`.)

- [ ] **Step 2: Run to verify failure**

Run: `pnpm vitest run lib/__tests__/plural.test.ts lib/__tests__/format-module.test.ts lib/__tests__/event-module.test.ts`
Expected: FAIL — modules/exports not found.

- [ ] **Step 3: Implement**

Append to `frontend/lib/format.ts`:

```ts
// Compact numeric forms for the Swiss Grid modules — all numerals render in
// JetBrains Mono at the call site. Moscow-pinned like everything above.
const shortDateFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "2-digit",
  month: "2-digit",
  timeZone: "Europe/Moscow",
});
const shortTimeFmt = new Intl.DateTimeFormat("ru-RU", {
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "Europe/Moscow",
});

/** "16:00" — Moscow wall clock. */
export function formatStartTime(iso: string): string {
  return shortTimeFmt.format(new Date(iso));
}

/** "12.07 · 16:00" single-day; "15.08 – 17.08" across civil days. */
export function formatModuleDate(startsAt: string, endsAt?: string): string {
  const single = `${shortDateFmt.format(new Date(startsAt))} · ${formatStartTime(startsAt)}`;
  if (!hasRealEnd(endsAt)) return single;
  if (moscowDay(startsAt) === moscowDay(endsAt as string)) return single;
  return `${shortDateFmt.format(new Date(startsAt))} – ${shortDateFmt.format(new Date(endsAt as string))}`;
}

/** "12 / 40" | "64" | "—" — mono seат counter for cells and module footers. */
export function attendanceShort(
  event: Pick<LiaEvent, "attendeeCount" | "capacity">,
): string {
  if (event.attendeeCount == null) return "—";
  return event.capacity != null
    ? `${event.attendeeCount} / ${event.capacity}`
    : String(event.attendeeCount);
}
```

(`hasRealEnd` and `moscowDay` already exist in this file at lines 31–42 — they are module-private, no import needed.)

`frontend/lib/plural.ts`:

```ts
/** Russian plural: pluralRu(42, ["событие","события","событий"]) → "события". */
export function pluralRu(n: number, forms: [string, string, string]): string {
  const abs = Math.abs(n) % 100;
  if (abs >= 11 && abs <= 14) return forms[2];
  const last = abs % 10;
  if (last === 1) return forms[0];
  if (last >= 2 && last <= 4) return forms[1];
  return forms[2];
}
```

`frontend/lib/event-module.ts`:

```ts
import { categoryNumeral } from "./category-numerals";
import { formatModuleDate } from "./format";
import { priceLabel } from "./price-label";
import type { LiaEvent } from "./types";

export interface EventModuleData {
  numeral: string;
  category: string;
  title: string;
  venue: string;
  date: string;
  price: string;
  href: string;
}

/** Adapts a backend event to the EventModule props. Numerals are positional
 * in the backend's ordered category list (Swiss rule: numerals, not colours). */
export function eventToModuleProps(
  event: LiaEvent,
  categories: ReadonlyArray<{ slug: string }>,
): EventModuleData {
  const cat = event.categories[0];
  return {
    numeral: cat ? categoryNumeral(cat.slug, categories) : "—",
    category: cat?.label ?? "—",
    title: event.title,
    venue: event.venue?.name ?? (event.format === "online" ? "Онлайн" : "—"),
    date: formatModuleDate(event.startsAt, event.endsAt),
    price: priceLabel(event.priceMin, event.priceType),
    href: `/events/${event.id}`,
  };
}
```

Note: `priceLabel`'s `PriceKind` is `"free" | "from" | "fixed"` — identical to the backend `PriceType`, pass `event.priceType` straight through.

- [ ] **Step 4: Run tests** — `pnpm test` → all pass (49 + new).

- [ ] **Step 5: Commit**

```bash
git add lib/format.ts lib/plural.ts lib/event-module.ts lib/__tests__/
git commit -m "feat(frontend): module date/attendance/plural helpers + event->module adapter"
```

---

### Task 2: U8 — inverted 404 + AuthGate

**Files:**
- Create: `frontend/app/not-found.tsx`, `frontend/components/ui/AuthGate.tsx`

**Interfaces:**
- Consumes: `Button` (`variant="inverted" | "ghost"`), `EmptyState` is NOT used here (404 and auth-gate have bespoke U8 layouts).
- Produces: `<AuthGate title? reassurance?>` — client component; renders the U8 auth-required block with primary `ВОЙТИ` linking to `/login?next=<current path>` and ghost `К ЛЕНТЕ` → `/`. Later tasks and phases drop it into any gated route.

- [ ] **Step 1: Implement `frontend/app/not-found.tsx`** (server component)

```tsx
import Link from "next/link";

/** U8-3: 404, inverted to ink. Mono numeral, one sentence, one action. */
export default function NotFound() {
  return (
    <div
      data-surface="ink"
      className="flex min-h-screen flex-col items-start justify-center gap-[10px] bg-surface px-[20px] text-on-surface"
    >
      <span className="font-mono text-[44px] font-bold leading-none tracking-[-0.04em]">
        404
      </span>
      <h1 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">
        Страница не найдена
      </h1>
      <p className="max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim-dark-2">
        Возможно, событие сняли с публикации или ссылка устарела.
      </p>
      <Link
        href="/"
        className="swiss-focus mt-[6px] bg-paper px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink hover:opacity-90"
      >
        Вернуться к ленте
      </Link>
    </div>
  );
}
```

- [ ] **Step 2: Implement `frontend/components/ui/AuthGate.tsx`**

```tsx
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button } from "@/components/ui/Button";

/** U8-2: auth-required surface. Names the situation, one sentence, two actions. */
export function AuthGate({
  title = "Войдите, чтобы продолжить",
  reassurance = "Лента и карта доступны без входа.",
}: {
  title?: string;
  reassurance?: string;
}) {
  const pathname = usePathname();
  const loginHref = `/login?next=${encodeURIComponent(pathname)}`;
  return (
    <div className="flex flex-col items-start gap-[10px] px-[20px] py-[40px]">
      <span className="cap">Доступ</span>
      <h2 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">{title}</h2>
      <p className="max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim">{reassurance}</p>
      <div className="mt-[6px] flex gap-[8px]">
        <Link href={loginHref} className="swiss-focus bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black">
          Войти
        </Link>
        <Link href="/" className="swiss-focus border border-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink hover-invert">
          К ленте
        </Link>
      </div>
    </div>
  );
}
```

(Links, not `<Button>`, because both actions navigate. `Button` import removed if unused — keep the file free of dead imports.)

- [ ] **Step 3: Verify** — `pnpm build && pnpm lint && npx tsc --noEmit`; in the dev browser `http://localhost:3000/no-such-page` renders the inverted 404.

- [ ] **Step 4: Commit**

```bash
git add app/not-found.tsx components/ui/AuthGate.tsx
git commit -m "feat(frontend): U8 inverted 404 + AuthGate pattern"
```

---

### Task 3: U7 — shared AuthForm + /login + /signup pages

**Files:**
- Create: `frontend/components/AuthForm.tsx`, `frontend/app/login/page.tsx`, `frontend/app/signup/page.tsx`

**Interfaces:**
- Consumes: `useAuth()` → `{ register(email, name, password), loginPassword(email, password) }` (`lib/auth-context.tsx`; exact call shapes as used in `components/AuthButton.tsx:113-118`); `Input` from `components/ui/Field`; `Button`.
- Produces: `<AuthForm mode="login"|"register" onSuccess?: () => void>` — client component with the full form logic: fields, busy/error state, registered-confirmation state (registration shows «Проверьте почту» + link `/auth/verify?next=…`), `?next=` resume via the same `currentNext()` guard (copy it — single leading `/`, reject `//`). Both the pages and Task 4's restyled `LoginModal` consume it.

- [ ] **Step 1: Implement `frontend/components/AuthForm.tsx`**

```tsx
"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Field";
import { useAuth } from "@/lib/auth-context";

// App-internal ?next= only (single leading "/"): prevents open redirects.
function currentNext(): string | null {
  if (typeof window === "undefined") return null;
  const next = new URLSearchParams(window.location.search).get("next");
  return next && next.startsWith("/") && !next.startsWith("//") ? next : null;
}

export function AuthForm({
  mode,
  onSuccess,
}: {
  mode: "login" | "register";
  onSuccess?: () => void;
}) {
  const { register, loginPassword } = useAuth();
  const router = useRouter();
  const next = currentNext();
  const isRegister = mode === "register";

  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [registeredEmail, setRegisteredEmail] = useState<string | null>(null);

  const mismatch = isRegister && confirm.length > 0 && confirm !== password;
  const canSubmit =
    email.trim().length > 0 &&
    password.length >= (isRegister ? 8 : 1) &&
    (!isRegister || confirm === password) &&
    !busy;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setBusy(true);
    setError(null);
    try {
      if (isRegister) {
        const addr = email.trim();
        await register(addr, name.trim(), password);
        setRegisteredEmail(addr);
      } else {
        await loginPassword(email.trim(), password);
        onSuccess?.();
        router.push(next ?? "/");
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Не удалось войти. Попробуйте ещё раз.",
      );
    } finally {
      setBusy(false);
    }
  };

  if (registeredEmail) {
    return (
      <div className="flex flex-col gap-[10px]">
        <h2 className="text-[22px] font-black tracking-[-0.02em]">Проверьте почту</h2>
        <p className="max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim">
          Мы отправили 6-значный код на {registeredEmail}. Он действует 24 часа.
        </p>
        <Link
          href={next ? `/auth/verify?next=${encodeURIComponent(next)}` : "/auth/verify"}
          className="swiss-focus mt-[6px] self-start bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black"
        >
          Ввести код
        </Link>
      </div>
    );
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-[11px]">
      {isRegister && (
        <Input label="Имя" value={name} onChange={(e) => setName(e.target.value)} placeholder="Как вас представить участникам" autoComplete="name" />
      )}
      <Input label="Почта" type="email" required autoFocus value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" autoComplete="email" />
      <Input label={isRegister ? "Пароль (минимум 8 символов)" : "Пароль"} type="password" required minLength={isRegister ? 8 : undefined} value={password} onChange={(e) => setPassword(e.target.value)} placeholder="••••••••" autoComplete={isRegister ? "new-password" : "current-password"} />
      {isRegister && (
        <Input label="Пароль ещё раз" type="password" required value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder="••••••••" autoComplete="new-password" error={mismatch ? "Пароли не совпадают" : undefined} />
      )}
      {error && <p className="text-[11px] text-signal">{error}</p>}
      <Button type="submit" disabled={!canSubmit} className="mt-[6px]">
        {busy ? (isRegister ? "Создаём…" : "Вход…") : isRegister ? "Создать аккаунт" : "Войти"}
      </Button>
    </form>
  );
}
```

- [ ] **Step 2: Implement `frontend/app/login/page.tsx`**

```tsx
import Link from "next/link";
import { AuthForm } from "@/components/AuthForm";

export const metadata = { title: "Вход — PRESENCE" };

/** U7: split screen — ink brand panel left, paper form right; stacked on mobile. */
export default function LoginPage() {
  return (
    <main className="grid min-h-screen grid-cols-2 max-md:grid-cols-1">
      <div
        data-surface="ink"
        className="flex flex-col justify-between bg-surface p-[20px] text-on-surface max-md:min-h-[220px]"
      >
        <span className="text-[13px] font-black tracking-[-0.01em]">PRESENCE</span>
        <div className="flex flex-col gap-[10px]">
          <h1 className="max-w-[16ch] text-[34px] font-black leading-[0.94] tracking-[-0.03em] max-md:text-[22px]">
            Медиации, лекции и разговоры об искусстве
          </h1>
          <p className="cap">События Москвы</p>
        </div>
      </div>
      <div className="flex flex-col justify-center gap-[14px] p-[20px] md:px-[48px]">
        <div>
          <p className="cap">Вход</p>
          <h2 className="mt-[6px] text-[22px] font-black tracking-[-0.02em]">
            С возвращением
          </h2>
        </div>
        <AuthForm mode="login" />
        <p className="mt-auto pt-[14px] text-[11.5px] text-text-dim">
          Нет аккаунта?{" "}
          <Link href="/signup" className="swiss-focus underline underline-offset-2">
            Регистрация
          </Link>
        </p>
      </div>
    </main>
  );
}
```

- [ ] **Step 3: Implement `frontend/app/signup/page.tsx`** — same frame, registration copy:

```tsx
import Link from "next/link";
import { AuthForm } from "@/components/AuthForm";

export const metadata = { title: "Регистрация — PRESENCE" };

export default function SignupPage() {
  return (
    <main className="grid min-h-screen grid-cols-2 max-md:grid-cols-1">
      <div
        data-surface="ink"
        className="flex flex-col justify-between bg-surface p-[20px] text-on-surface max-md:min-h-[220px]"
      >
        <span className="text-[13px] font-black tracking-[-0.01em]">PRESENCE</span>
        <div className="flex flex-col gap-[10px]">
          <h1 className="max-w-[16ch] text-[34px] font-black leading-[0.94] tracking-[-0.03em] max-md:text-[22px]">
            Участвуйте, а не только смотрите
          </h1>
          <p className="cap">Аккаунт за минуту</p>
        </div>
      </div>
      <div className="flex flex-col justify-center gap-[14px] p-[20px] md:px-[48px]">
        <div>
          <p className="cap">Регистрация</p>
          <h2 className="mt-[6px] text-[22px] font-black tracking-[-0.02em]">
            Создайте аккаунт
          </h2>
        </div>
        <AuthForm mode="register" />
        <p className="mt-auto pt-[14px] text-[11.5px] text-text-dim">
          Уже есть аккаунт?{" "}
          <Link href="/login" className="swiss-focus underline underline-offset-2">
            Войти
          </Link>
        </p>
      </div>
    </main>
  );
}
```

Copy note: the left-panel headline for /login is the handoff's literal string; the /signup headline is new copy (handoff reuses the frame without specifying signup copy) — content team may replace.

- [ ] **Step 4: Verify** — `pnpm build && pnpm test && pnpm lint`; dev browser: `/login` and `/signup` at desktop + 390px (panel stacks above form), tab bar absent (already gated), form error states render in signal red, successful login redirects to `/` (or `?next=`).

- [ ] **Step 5: Commit**

```bash
git add components/AuthForm.tsx app/login app/signup
git commit -m "feat(frontend): U7 login/signup pages with shared AuthForm"
```

---

### Task 4: Restyled LoginModal + AuthNavControl

**Files:**
- Modify: `frontend/components/AuthButton.tsx` (LoginModal becomes a Swiss modal shell around `AuthForm`; `AuthButton` stays exported and working for old GlassNav pages)
- Create: `frontend/components/ui/AuthNavControl.tsx`

**Interfaces:**
- Consumes: `AuthForm` (Task 3), `useAuth()` → `{ email, isAuthed, ready, logout, role }`.
- Produces: `LoginModal({ onClose })` — same export name/signature (SignupCTA, CreateEventForm, ReportButton import it; do not break them). Produces `<AuthNavControl />` — nav-styled auth slot for `AppHeader` `actions` (9px uppercase tracked links, matching nav items).

- [ ] **Step 1: Rewrite `LoginModal` in `frontend/components/AuthButton.tsx`** — replace the entire modal body (lines ~89–252) with:

```tsx
export function LoginModal({ onClose }: { onClose: () => void }) {
  const [mode, setMode] = useState<"login" | "register">("login");
  const isRegister = mode === "register";
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm border border-ink bg-paper p-[20px]"
        onClick={(e) => e.stopPropagation()}
      >
        <p className="cap">{isRegister ? "Регистрация" : "Вход"}</p>
        <h2 className="mb-[14px] mt-[6px] text-[22px] font-black tracking-[-0.02em]">
          {isRegister ? "Создайте аккаунт" : "С возвращением"}
        </h2>
        <AuthForm mode={mode} onSuccess={onClose} />
        <div className="mt-[14px] flex items-center justify-between">
          <button
            type="button"
            className="swiss-focus text-[11.5px] underline underline-offset-2"
            onClick={() => setMode(isRegister ? "login" : "register")}
          >
            {isRegister ? "Уже есть аккаунт? Войти" : "Нет аккаунта? Регистрация"}
          </button>
          <button type="button" className="cap swiss-focus" onClick={onClose}>
            Закрыть
          </button>
        </div>
      </div>
    </div>
  );
}
```

Keep `AuthButton` itself compiling (it still calls `LoginModal`); remove the now-unused `currentNext`, `useRouter`, and input-class code from this file (AuthForm owns them). Add `import { AuthForm } from "@/components/AuthForm";`.

- [ ] **Step 2: Implement `frontend/components/ui/AuthNavControl.tsx`**

```tsx
"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth-context";

const NAV_ITEM = "swiss-focus font-alt text-[9px] uppercase tracking-[0.14em]";

/** AppHeader actions slot: «Войти» when signed out; email + «Выйти» (+Админ) when in. */
export function AuthNavControl() {
  const { email, isAuthed, ready, logout, role } = useAuth();
  if (!ready) return <span className={NAV_ITEM}>…</span>;
  if (!isAuthed) {
    return (
      <Link href="/login" className={NAV_ITEM}>
        Войти
      </Link>
    );
  }
  return (
    <span className="flex items-baseline gap-[14px]">
      {role === "admin" && (
        <Link href="/admin" className={NAV_ITEM}>
          Админ
        </Link>
      )}
      <span className="max-w-[10rem] truncate font-alt text-[9px] tracking-[0.14em] text-muted-2" title={email ?? undefined}>
        {email}
      </span>
      <button type="button" onClick={logout} className={NAV_ITEM}>
        Выйти
      </button>
    </span>
  );
}
```

- [ ] **Step 3: Verify** — `pnpm build && pnpm test && pnpm lint`; browser: open an RSVP flow signed-out (SignupCTA «Записаться» on any event) — the Swiss modal appears and logs in.

- [ ] **Step 4: Commit**

```bash
git add components/AuthButton.tsx components/ui/AuthNavControl.tsx
git commit -m "feat(frontend): Swiss LoginModal over AuthForm + AuthNavControl for AppHeader"
```

---

### Task 5: U1 — feed page rewrite

**Files:**
- Modify: `frontend/app/page.tsx`, `frontend/components/DiscoveryFeed.tsx` (full rewrite of the render; keep all data logic)

**Interfaces:**
- Consumes: `AppHeader`+`USER_NAV`, `AuthNavControl`, `Chip`, `EventModule`, `EmptyState`, `Skeleton`, `eventToModuleProps`, `pluralRu`, `getCategories`/`ApiCategory`, existing query/near-me logic in `DiscoveryFeed` (`components/DiscoveryFeed.tsx:30-143` — keep verbatim), `FILTERS` time entries (`lib/mock-events.ts:52` — only `all/today/weekend` survive).
- Produces: `DiscoveryFeed({ initialEvents, categories }: { initialEvents: LiaEvent[]; categories: ApiCategory[] })` — new prop.

- [ ] **Step 1: Rewrite `frontend/app/page.tsx`**

```tsx
import { DiscoveryFeed } from "@/components/DiscoveryFeed";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { fetchPublishedEvents, getCategories } from "@/lib/api";
import { MOCK_EVENTS } from "@/lib/mock-events";

// U1 · Лента событий. SSR both the events and the ordered category taxonomy
// (numerals are positional in this list); mock fallback keeps local dev alive.
export default async function DiscoveryPage() {
  const [initialEvents, categories] = await Promise.all([
    fetchPublishedEvents().catch(() => MOCK_EVENTS),
    getCategories().catch(() => []),
  ]);

  return (
    <>
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="МСК" />
      <DiscoveryFeed initialEvents={initialEvents} categories={categories} />
    </>
  );
}
```

- [ ] **Step 2: Rewrite the render of `frontend/components/DiscoveryFeed.tsx`.** Keep ALL existing state/query/geolocation logic (lines 30–143) unchanged, plus these changes:

1. New props: `{ initialEvents, categories }: { initialEvents: LiaEvent[]; categories: ApiCategory[] }` (import `ApiCategory` from `@/lib/api`).
2. Time filters: replace the `FILTERS` import usage with a local constant of the three surviving time chips plus «Бесплатно»:

```tsx
const TIME_FILTERS = [
  { slug: "all", label: "Все" },
  { slug: "today", label: "Сегодня", dateRange: todayRange },
  { slug: "weekend", label: "Выходные", dateRange: weekendRange },
  { slug: "free", label: "Бесплатно" },
] as const;
```

Export `todayRange`/`weekendRange` from `lib/mock-events.ts` (they are module-private there today — add `export` to both; `FILTERS` itself stays for any other consumer until Phase 7 deletes it). `active === "free"` filters `e.priceType === "free"` client-side (no dateRange, no category). Category selection becomes a separate state: `const [activeCat, setActiveCat] = useState<string | null>(null)` filtering `e.categories.some(c => c.slug === activeCat)`; time chip and category chip compose (AND).
3. The dateRange memo keys off `TIME_FILTERS` now; the category branch moves out of it (it lives in `activeCat`).
4. New render (replace everything from `return (` down):

```tsx
  const count = displayEvents.length;

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
      {/* Title block */}
      <div className="border-b border-ink px-[20px] py-[18px]">
        <p className="cap">
          Москва · <span className="font-mono">{count}</span>{" "}
          {pluralRu(count, ["событие", "события", "событий"])}
        </p>
        <h1 className="mt-[8px] max-w-[14ch] text-[38px] font-black leading-[0.94] tracking-[-0.03em] max-sm:text-[22px]">
          Лента событий
        </h1>
      </div>

      {/* Filter bar: time+near-me left, categories (API-ordered) right */}
      <div className="flex items-center justify-between gap-[10px] overflow-x-auto border-b border-ink px-[20px] py-[9px] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <div className="flex shrink-0 items-center gap-[6px]">
          {TIME_FILTERS.map((f) => (
            <Chip key={f.slug} variant={active === f.slug ? "active" : "default"} onClick={() => setActive(f.slug)}>
              {f.label}
            </Chip>
          ))}
          <Chip
            variant={isNearbyMode ? "active" : "default"}
            disabled={geoLoading}
            onClick={isNearbyMode ? resetNearby : enableNearby}
            aria-label={isNearbyMode ? "Сбросить фильтр по расстоянию" : "Показать события рядом со мной"}
          >
            {geoLoading ? "Определяем…" : "Рядом со мной"}
          </Chip>
        </div>
        <div className="flex shrink-0 items-center gap-[6px]">
          {categories.map((c) => (
            <Chip key={c.id} variant={activeCat === c.slug ? "active" : "default"} onClick={() => setActiveCat(activeCat === c.slug ? null : c.slug)}>
              {c.label}
            </Chip>
          ))}
        </div>
      </div>

      {/* Search (retained feature, Swiss field) */}
      <div className="border-b border-rule-inner px-[20px] py-[9px]">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Поиск по названию, месту, ведущему"
          aria-label="Поиск событий"
          className="swiss-focus w-full bg-transparent text-[12.5px] text-on-surface placeholder:text-field-text"
        />
      </div>

      {geoError && <p className="border-b border-rule-inner px-[20px] py-[9px] text-[11.5px] text-signal">{geoError}</p>}

      {/* Catalogue grid */}
      {isError && allEvents.length === 0 ? (
        <EmptyState
          numeral="!"
          title="Не удалось загрузить события"
          text="Проверьте соединение и попробуйте обновить страницу."
        />
      ) : count > 0 ? (
        <div className="grid grid-cols-3 border-b border-ink max-sm:grid-cols-1 [&>a]:border-b [&>a]:border-r [&>a]:border-rule-inner max-sm:[&>a]:border-r-0">
          {displayEvents.map((e) => {
            const m = eventToModuleProps(e, categories);
            return (
              <EventModule
                key={e.id}
                {...m}
                venue={
                  isNearbyMode && e.distanceM != null
                    ? `${m.venue} · ${(e.distanceM / 1000).toFixed(1)} км`
                    : m.venue
                }
              />
            );
          })}
        </div>
      ) : (
        <EmptyState
          title={isNearbyMode ? "Событий рядом нет" : "Ничего не нашлось"}
          text={isNearbyMode ? "Попробуйте расширить поиск или сбросить фильтр." : "Попробуйте другой фильтр или сбросьте поиск."}
          actions={
            <Button variant="ghost" onClick={() => { setActive("all"); setActiveCat(null); setQuery(""); resetNearby(); }}>
              Сбросить фильтры
            </Button>
          }
        />
      )}
    </main>
  );
```

Delete the `PinGlyph`/`ClearGlyph`/`SpinnerGlyph` components and the `SearchField`/`FilterChip`/`EventCard` imports from this file (system has no icon library). Add a loading state: while `useQuery` `isPending && allEvents.length === 0`, render 6 `<Skeleton className="h-[140px]" />` in the same grid instead of modules.
5. Keep the mock fallback consistent: when `categories` is empty (offline), `eventToModuleProps` yields `numeral "—"` — acceptable offline degradation.

- [ ] **Step 3: Verify** — `pnpm build && pnpm test && pnpm lint`; browser desktop + 390px: 3-col hairline grid / stacked rows, chips invert on hover and select, «Бесплатно» filters, near-me still works, count caption updates, tab bar clears the content (no overlap — the `max-sm:pb-[88px]` handles the fixed bar).

- [ ] **Step 4: Commit**

```bash
git add app/page.tsx components/DiscoveryFeed.tsx lib/mock-events.ts
git commit -m "feat(frontend): U1 swiss feed — title block, chip filter bar, EventModule grid"
```

---

### Task 6: U2 — event detail rewrite + map treatment

**Files:**
- Modify: `frontend/components/EventDetailView.tsx` (full rewrite), `frontend/components/VenueMap.tsx`

**Interfaces:**
- Consumes: `Cell`/`CellStrip`, `Chip`, `Button`, `formatEventRange`/`formatStartTime`/`attendanceShort`/`priceLabel`, `VerifiedBadge`, `SignupCTA` (restyled in Task 7 — consume as-is here), `FeedbackForm`, `ReportButton`, `EventCover`, `VenueMap`.
- Produces: same export `EventDetailView({ event })` — server/client-agnostic as today.

- [ ] **Step 1: Update `frontend/components/VenueMap.tsx`** — wrap in the desaturated square treatment:

```tsx
"use client";

import dynamic from "next/dynamic";

const YandexMap = dynamic(() => import("@/components/map/YandexMap"), { ssr: false });

/** Swiss map treatment: grayscale печатный план, square, hairline border. */
export function VenueMap({ lat, lon }: { lat: number; lon: number }) {
  return (
    <div className="border border-ink [filter:grayscale(1)_contrast(1.05)]">
      <YandexMap center={[lat, lon]} zoom={15} marker={[lat, lon]} className="h-[190px] w-full" />
    </div>
  );
}
```

(Check `components/VenueMap.tsx` first — it is currently 11 lines wrapping `YandexMap`; keep its existing prop pass-through if it differs, only adding the wrapper div + `className` override killing the old `rounded-control` default.)

- [ ] **Step 2: Rewrite `frontend/components/EventDetailView.tsx`**

```tsx
import { FeedbackForm } from "@/components/FeedbackForm";
import { ReportButton } from "@/components/ReportButton";
import { SignupCTA } from "@/components/SignupCTA";
import { VerifiedBadge } from "@/components/VerifiedBadge";
import { EventCover } from "@/components/ui/EventCover";
import { Cell, CellStrip } from "@/components/ui/Cell";
import { VenueMap } from "@/components/VenueMap";
import {
  attendanceShort,
  formatEventRange,
  formatStartTime,
} from "@/lib/format";
import { priceLabel } from "@/lib/price-label";
import type { LiaEvent } from "@/lib/types";
import Link from "next/link";

/** U2 · Страница события. Paper, hairline blocks, price+CTA rail on desktop,
 * sticky price+CTA footer on mobile. */
export function EventDetailView({ event }: { event: LiaEvent }) {
  const ended = new Date(event.endsAt ?? event.startsAt) < new Date();
  const price = priceLabel(event.priceMin, event.priceType);
  const cat = event.categories[0];
  const routeHref =
    event.venue?.lat != null && event.venue?.lon != null
      ? `https://yandex.ru/maps/?rtext=~${event.venue.lat},${event.venue.lon}`
      : null;

  return (
    <div className="mx-auto min-h-screen max-w-[1360px] pb-[96px] md:pb-[40px]">
      {/* Breadcrumb header */}
      <header className="flex items-baseline justify-between border-b border-ink px-[20px] py-[13px]">
        <Link href="/" className="swiss-focus font-alt text-[9px] uppercase tracking-[0.14em]">
          ← События
        </Link>
        <span className="cap">{cat?.label ?? "Событие"}</span>
      </header>

      {/* Cover strip (single cover — we have one image per event) */}
      <div className="border-b border-ink">
        <EventCover
          event={event}
          aspect="aspect-[3/1] max-md:aspect-[3/2]"
          rounded=""
          sizes="(max-width: 768px) 100vw, 1360px"
          priority
        />
      </div>

      {/* Title block: text left, price+CTA rail right */}
      <div className="grid grid-cols-[1fr_200px] border-b border-ink max-md:grid-cols-1">
        <div className="flex flex-col gap-[10px] px-[20px] py-[16px]">
          <p className="cap">
            {[cat?.label, event.format === "online" ? "Онлайн" : "Очно"]
              .filter(Boolean)
              .join(" · ")}
          </p>
          <h1 className="max-w-[22ch] text-[30px] font-black leading-[0.94] tracking-[-0.03em] max-md:text-[21px]">
            {event.title}
          </h1>
          {event.description && (
            <p className="max-w-[52ch] text-[12px] leading-[1.45] text-text-dim">
              {event.description}
            </p>
          )}
        </div>
        <div className="flex flex-col border-l border-rule-inner px-[14px] py-[10px] max-md:hidden">
          <span className="cap">Цена</span>
          <span className="mt-[4px] font-mono text-[26px] font-bold leading-none">{price}</span>
          {!ended && (
            <div className="mt-auto pt-[10px]">
              <SignupCTA event={event} />
            </div>
          )}
        </div>
      </div>

      {/* Fact strip */}
      <CellStrip cols={4} className="max-md:grid-cols-2">
        <Cell caption="Когда" value={formatEventRange(event)} />
        <Cell caption="Начало" value={formatStartTime(event.startsAt)} mono />
        <Cell caption="Места" value={attendanceShort(event)} mono />
        <Cell
          caption="Организатор"
          value={
            event.organizer ? (
              event.organizer.profile_id ? (
                <Link
                  href={`/organizers/${event.organizer.profile_id}`}
                  className="swiss-focus underline-offset-2 hover:underline"
                >
                  {event.organizer.name || "Организатор"}
                  {event.organizer.verified ? <> <VerifiedBadge /></> : null}
                </Link>
              ) : (
                <>
                  {event.organizer.name || "Организатор"}
                  {event.organizer.verified ? <> <VerifiedBadge /></> : null}
                </>
              )
            ) : (
              "—"
            )
          }
        />
      </CellStrip>

      {/* Venue block: map left, address rail right */}
      {event.venue && (
        <div className="grid grid-cols-[1fr_200px] border-b border-ink max-md:grid-cols-1">
          <div className="p-[14px]">
            {event.venue.lat != null && event.venue.lon != null ? (
              <VenueMap lat={event.venue.lat} lon={event.venue.lon} />
            ) : (
              <p className="text-[12px] text-text-dim">{event.venue.name}</p>
            )}
          </div>
          <div className="flex flex-col border-l border-rule-inner px-[14px] py-[10px]">
            <span className="cap">Адрес</span>
            <span className="mt-[4px] text-[12px] font-bold leading-[1.25]">
              {event.venue.name}
            </span>
            {event.venue.address && (
              <span className="mt-[2px] text-[11px] text-field-text">{event.venue.address}</span>
            )}
            {event.venue.metro && (
              <span className="mt-[2px] text-[11px] text-field-text">м. {event.venue.metro}</span>
            )}
            {routeHref && (
              <a
                href={routeHref}
                target="_blank"
                rel="noopener noreferrer"
                className="swiss-focus hover-invert mt-auto border border-ink px-[11px] py-[7px] text-center text-[9px] font-bold uppercase tracking-[0.07em]"
              >
                Маршрут
              </a>
            )}
          </div>
        </div>
      )}

      {/* Description / feedback / report */}
      {ended && (
        <section className="border-b border-rule-inner px-[20px] py-[16px]">
          <p className="cap mb-[10px]">Отзыв</p>
          <FeedbackForm eventId={event.id} />
        </section>
      )}
      <div className="px-[20px] py-[16px]">
        <ReportButton eventId={event.id} />
      </div>

      {/* Mobile sticky footer: price + CTA */}
      {!ended && (
        <div className="fixed inset-x-0 bottom-0 z-20 border-t border-ink bg-paper md:hidden">
          <div className="flex items-center justify-between gap-[10px] px-[14px] py-[9px] pb-[calc(9px+env(safe-area-inset-bottom))]">
            <span className="font-mono text-[17px] font-bold">{price}</span>
            <SignupCTA event={event} />
          </div>
        </div>
      )}
    </div>
  );
}
```

Notes for the implementer: `Cell`'s `value` prop is `ReactNode` — the organizer link nests inside a `Cell`, which is NOT wrapped in any `<Link>` (no nested-anchor risk). The mobile sticky footer overlaps `BottomTabBar` — but `/events/[id]` shows the tab bar; stack them: change the footer to `bottom-[57px] max-sm:bottom-[57px] sm:bottom-0`... **Simpler correct rule:** TabBarGate already hides nothing here, so set the footer `z-20` above and add `pb-[120px]` page bottom padding on mobile (`pb-[96px]` in the root div covers tab bar 57px + footer ~63px is tight — use `pb-[140px] md:pb-[40px]` on the root div instead). The exact numbers matter less than: on a 390px viewport, the last content block must scroll fully above BOTH fixed bars, and the sticky footer must sit visually ABOVE the tab bar (footer `bottom-[49px]` = tab bar height 48px + 1px rule, `md:bottom-0`). Implement it as: footer classes `fixed inset-x-0 bottom-[49px] z-20 border-t border-ink bg-paper md:hidden` and root padding `pb-[150px] md:pb-[40px]`, then verify visually at 390px.

- [ ] **Step 3: Verify** — `pnpm build && pnpm test && pnpm lint`; browser: a published event at desktop (price rail right, 4-cell fact strip, grayscale map, МАРШРУТ opens Yandex route) and 390px (sticky footer above tab bar, 2-col fact strip); a draft event as owner still renders (OwnerDraftFallback path unchanged); unknown id → inverted 404 from Task 2.

- [ ] **Step 4: Commit**

```bash
git add components/EventDetailView.tsx components/VenueMap.tsx
git commit -m "feat(frontend): U2 swiss event detail — cells, price rail, desaturated map"
```

---

### Task 7: SignupCTA restyle (logic untouched)

**Files:**
- Modify: `frontend/components/SignupCTA.tsx` (438 lines — visual classes only)

**Interfaces:**
- Consumes: `Button` (Phase 1 variants), `Chip`, existing RSVP logic/state/mutations (DO NOT change any hook, handler, condition, or copy string).
- Produces: same export, same behaviour; Swiss visuals.

- [ ] **Step 1: Read the file, then apply this exact class mapping everywhere it renders UI** (visual-only diff; every changed line must be a `className` or wrapping-element style change):

| Old pattern | Replacement |
|---|---|
| `rounded-*` (any) | remove (zero radius) |
| `bg-accent text-white` primary actions | use `<Button>` (default primary) or classes `bg-ink text-white hover:bg-black` |
| waitlist / secondary «В ЛИСТ ОЖИДАНИЯ»-style actions | `<Button variant="ghost">` or classes `border border-ink text-ink hover-invert` |
| `bg-fill` info pills / boxes | `border border-rule-inner bg-transparent` |
| `text-label-secondary` helper text | `text-text-dim text-[11.5px]` |
| `text-[15px]/text-[17px]` control text | `text-[11px] font-bold uppercase tracking-[0.07em]` on buttons; `text-[12.5px]` on body text |
| status lines («Вы записаны», counters) | caption style `cap` for labels; counts wrapped in `font-mono` |
| `text-red-*` errors | `text-signal text-[11px]` |
| `shadow-*`, `glass` | remove |
| spinners (if any `animate-spin`) | text «…» (system has no spinners) |

Numbers rule: any seat counter («Осталось N мест», «N / M») wraps the numeals in `<span className="font-mono">`.

- [ ] **Step 2: Verify** — `pnpm build && pnpm test && pnpm lint` (the create-event schema tests and signup-availability tests must stay green — they cover logic you did not touch); browser: open/application/external/full/waitlist states as far as reachable with local data (at minimum: signed-out → LoginModal opens; open RSVP → Записаться → Вы записаны; cancel works).

- [ ] **Step 3: Commit**

```bash
git add components/SignupCTA.tsx
git commit -m "feat(frontend): SignupCTA restyled to swiss grid (logic untouched)"
```

---

### Task 8: U8 rollout — AuthGate on gated routes + no blank loading divs

**Files:**
- Modify: `frontend/app/me/calendar/page.tsx`, `frontend/app/events/mine/page.tsx`, `frontend/app/me/organizer/page.tsx`, `frontend/app/organizer/applications/page.tsx`

**Interfaces:**
- Consumes: `AuthGate` (Task 2), `Skeleton`.

- [ ] **Step 1:** In each file, find the logged-out branch (grep `Войдите` and the `!isAuthed` / `ready` checks) and replace the prompt block with `<AuthGate />` (customize `title` where the page names itself, e.g. `<AuthGate title="Войдите, чтобы видеть свой календарь" />`). Find the pre-hydration blank (`!ready` branches rendering `<div className="min-h-screen bg-bg-grouped" />` or similar) and replace with `<div className="px-[20px] py-[26px]"><Skeleton className="h-[120px] w-full" /></div>`. DO NOT restyle anything else on these pages (they belong to Phases 3/5).

- [ ] **Step 2: Verify** — `pnpm build && pnpm test && pnpm lint`; browser signed-out: all four routes show the U8 gate (not blanks), «Войти» goes to `/login?next=<route>` and returns after login.

- [ ] **Step 3: Commit**

```bash
git add app/me/calendar/page.tsx app/events/mine/page.tsx app/me/organizer/page.tsx app/organizer/applications/page.tsx
git commit -m "feat(frontend): U8 AuthGate + skeletons on gated routes"
```

---

### Task 9: «Читательские группы» category (migration 000021) + frontend maps

**Files:**
- Create: `backend/db/migrations/000021_reading_group_category.up.sql`, `backend/db/migrations/000021_reading_group_category.down.sql`
- Modify: `frontend/lib/covers.ts`, `frontend/components/ui/CategoryGlyph.tsx`

- [ ] **Step 1:** Read `backend/db/migrations/000006_categories_table.up.sql` to copy the exact column set/conflict style, then write (adjust columns to match 000006 exactly — if it has `label` not `name`, use `label`; if ids are uuid-defaulted, omit id):

```sql
-- 000021_reading_group_category.up.sql
INSERT INTO categories (slug, label)
VALUES ('reading-group', 'Читательские группы')
ON CONFLICT (slug) DO NOTHING;
```

```sql
-- 000021_reading_group_category.down.sql
DELETE FROM categories WHERE slug = 'reading-group';
```

- [ ] **Step 2:** Add `reading-group` entries to the category→gradient map in `frontend/lib/covers.ts` (copy an existing entry's shape; pick the same values as `lecture` — gradients die in Phase 7 anyway) and to the glyph map in `frontend/components/ui/CategoryGlyph.tsx` (reuse the `lecture` glyph path for now; these components serve the not-yet-migrated screens only).

- [ ] **Step 3:** Verify migration syntax locally if the local DB stack is up (`cd backend && make migrate-up` or the project's equivalent — check `backend/Makefile`; if the local stack is down, note it in the report — the migration runs on prod during the deploy task). `pnpm build && pnpm test` in `frontend/`.

- [ ] **Step 4: Commit**

```bash
git add backend/db/migrations/000021_reading_group_category.* frontend/lib/covers.ts frontend/components/ui/CategoryGlyph.tsx
git commit -m "feat: reading-group category (migration 000021) + frontend maps"
```

---

### Task 10: Dead-code sweep for migrated pages

**Files:**
- Modify/Delete: `frontend/components/ui/ThemeSwitch.tsx` (delete), plus whatever the greps below surface.

- [ ] **Step 1:** `grep -rln "ThemeSwitch" app components lib` → must be zero usages; `git rm components/ui/ThemeSwitch.tsx`.
- [ ] **Step 2:** `grep -rln "GlassNav\|BrandLogo\|SearchField\|FilterChip\|EventCard" app components` — expected remaining consumers are ONLY not-yet-migrated screens (`app/events/mine`, `app/organizer/*`, `app/me/*`, `app/organizers/[id]`, `app/map`, `app/search`, `app/admin/*`, `app/events/new|edit`, `app/auth/verify`, `app/invite`, `app/me/invitations`). If `app/page.tsx`, `DiscoveryFeed`, `EventDetailView`, `AuthForm`, login/signup still import any of them — remove. Do not delete the components themselves.
- [ ] **Step 3:** In `frontend/app/globals.css` TEMP shim block: no removals this phase (other screens still consume every alias) — only verify the comment still says Phase 2... update the comment to `remove in Phase 7 (final sweep)` on each TEMP block (the master plan moved final cleanup there).
- [ ] **Step 4:** `pnpm build && pnpm test && pnpm lint`; commit:

```bash
git add -A
git commit -m "chore(frontend): drop ThemeSwitch, retarget TEMP shim removal to phase 7"
```

---

### Task 11: Full-app browser verification (controller) + merge prep

No new files. The controller (not a subagent) verifies in the live dev browser at desktop and 390px against the `.dc.html` reference:

- [ ] `/` — U1: hairline 3-col grid, chip bar (time left / API categories right), «Бесплатно» works, near-me works, search works, count caption + plural correct, hover inverts, skeleton on cold load, tab bar clearance.
- [ ] `/events/<published-id>` — U2 desktop rail + mobile sticky footer above tab bar; grayscale square map; МАРШРУТ; fact cells all mono where numeric; organizer link works; ReportButton modal still opens; feedback block on a past event.
- [ ] `/login`, `/signup` — U7 split/stacked; register → «Проверьте почту» → `/auth/verify` flow reachable; login honours `?next=`.
- [ ] `/no-such-page` + `/events/nonexistent-id` — inverted 404.
- [ ] Gated routes signed-out — AuthGate everywhere, no blanks.
- [ ] Old screens (map, mine, organizer, admin) — still functional (visually stale is expected).
- [ ] `pnpm build && pnpm test && pnpm lint` at HEAD.

Then: final whole-branch review (subagent-driven-development flow) → merge to `main`.

---

### Task 12: Deploy Phases 1+2 to prod (controller-run, follows the runbook)

**⚠️ Contains a destructive prod-DB step (QA data purge) — requires explicit user confirmation before running; everything else is standard deploy.**

- [ ] **Step 1: Pre-flight.** Per `docs/superpowers/runbooks/2026-07-23-qa-20-jul-deploy.md` and [[lia-demo-deployment]] conventions: frontend build needs BOTH build-args (`NEXT_PUBLIC_API_URL=https://api.presence.tarski.ru`, `NEXT_PUBLIC_YANDEX_MAPS_KEY=…`); check the frontend `Dockerfile` declares `ARG` for both. **New risk this deploy: `next/font/google` downloads Golos Text/Manrope/JetBrains Mono at build time — the Docker build stage needs outbound network to `fonts.googleapis.com`/`fonts.gstatic.com`. Build on the Mac (as always); if the build fails on font fetch, retry — next/font caches in `.next/cache`.**
- [ ] **Step 2:** Build amd64 images on Mac → `docker save | ssh | docker load` (box can't pull big images), tag `swiss-p2-r1`, keep `rollback-preswiss` tags of the running images.
- [ ] **Step 3:** scp `backend/db/migrations/000021_reading_group_category.*.sql` to `/opt/lia/backend/db/migrations/` on the box, run migrate (Lia DB 020 → 021). `pg_dump` first.
- [ ] **Step 4 (DESTRUCTIVE — ask the user first, show them the exact SQL and ids):** QA test-data purge per old master-plan 0.2: identify live QA events/organizers («QA Тур в Геленджик (Блок 8)…», «bla bla meet», organizers «QA Block8», «kornkorn10») by listing them with SELECTs, then UNPUBLISH (`UPDATE events SET status='draft' WHERE id IN (…)`), never DELETE. Verify the feed no longer shows them. Grep `backend/db/seed/seed.sql` to confirm none are re-seeded.
- [ ] **Step 5:** Recreate containers (backend: all 4 compose files + `--no-build`; frontend container `lia-frontend-presence` on :3002). Verify live at https://presence.tarski.ru: feed in Swiss Grid, fonts loading (no FOUT to system serif on Cyrillic), event detail, login page, 404, gated routes; API health; migration `21|f`.
- [ ] **Step 6:** Docker prune on the box (builder prune + dangling images + trim `rollback-*` to last ~3) — the 20 GB disk fills otherwise. Update `docs/HANDOFF.md` + write the deploy runbook `docs/superpowers/runbooks/<date>-swiss-p1p2-deploy.md`.

---

## Self-Review

- **Spec coverage:** U1 ✅ (Task 5), U2 ✅ (Tasks 6–7), U7 ✅ (Tasks 3–4), U8 ✅ (Tasks 2, 8; loading states in 5/8; full-event waitlist ghost handled by SignupCTA mapping in 7). Hygiene: taxonomy ✅ (Task 5 chips from API), reading-group ✅ (Task 9), QA purge ✅ (Task 12 Step 4, gated on user), dev-note /search stub — **out of scope here** (U3 is Phase 4; the master plan's old 0.1 stopgap is superseded — the `/search` ComingSoon page remains until Phase 4; acceptable because it's behind a nav item, but flag to the user at merge time).
- **Placeholder scan:** clean — every code step has full code; Task 7 is a mapping-table restyle by design (438-line file, logic untouched, reviewer-gated); Task 9 Step 1 instructs adapting to 000006's real column names (a lookup, with the expected shape given).
- **Type consistency:** `eventToModuleProps` returns exactly `EventModuleProps` minus `matchReason`/`className` (spread works); `DiscoveryFeed` new props match `app/page.tsx` call; `AuthForm` mode prop matches both pages and LoginModal; `priceLabel(priceMin, priceType)` signature matches Phase 1 (`PriceKind` = backend `PriceType`); `Cell value` accepts ReactNode (Phase 1 defined `value: ReactNode`) — organizer-link cell relies on it.
- **Known risks carried into tasks:** mobile double-fixed-bars stacking (Task 6 gives the explicit rule + visual check), next/font network at Docker build (Task 12 Step 1), categories-empty offline degradation (Task 5 note).
