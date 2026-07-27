# Swiss Grid Phase 1 — Foundation (tokens, fonts, primitives) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Liquid Glass token layer with the Swiss Grid design system and build the shared primitives every screen phase depends on.

**Architecture:** All tokens live in `frontend/app/globals.css` (Tailwind v4 CSS-first, `@theme inline`). Surface inversion (admin) is a `data-surface="ink"` attribute scope that flips semantic variables — components never branch on "admin". Fonts load via `next/font/google` into CSS variables. Primitives live in `frontend/components/ui/`.

**Tech Stack:** Next.js 16 App Router, Tailwind v4, next/font, Vitest 4 (node-only — tests cover pure helpers, components are verified in the browser via the `/design-preview` route).

**Spec:** `docs/Redesign/5/design_handoff_presence_swiss_grid/README.md` (sections *Design tokens*, *Shared components*, *Interactions & behaviour*). Reference visuals: `Presence Swiss Grid - Full System.dc.html`.

## Global Constraints

- Zero border radius, zero shadows, 1px hairline rules everywhere.
- Colours/type sizes only from the token sheet; never hardcode a hex or px that a token carries.
- All numbers (dates, counts, prices, IDs) in JetBrains Mono; free price is the literal string `FREE`.
- Uppercase runs keep their tracking (`.cap` 0.13em, `.lbl` 0.14em, `.kick` 0.18em, chip 0.12em, button 0.07em, nav 0.14em).
- Hover inverts (`#111` bg / `#F2F0EC` text; `#1C1A18` on ink surface); focus = `outline: 2px solid` current surface's on-colour, offset 0; transitions `120ms linear` background/color only.
- Signal red `#E2231A` only for needs-attention/destructive.
- **Fonts (decision checkpoint 2 of the master plan): Archivo & Space Grotesk have NO Cyrillic on Google Fonts (verified 2026-07-28). Substitutes: Golos Text (UI/headings role), Manrope (caption/kicker role), JetBrains Mono as specified.** Loading is variable-indirected so a swap is one line.
- This phase merges only together with Phase 2 (deploying it alone leaves screens half-migrated). It must still build (`pnpm build`), lint, and pass all existing Vitest suites at every commit.
- Working directory for all commands: `frontend/`. Run in an isolated worktree (`superpowers:using-git-worktrees`), branch `redesign/swiss-grid-p1`.

## File Structure

```
frontend/
  app/globals.css                     REWRITE — Swiss Grid tokens + surface scopes + type utilities
  app/layout.tsx                      MODIFY — next/font, drop theme script, mount TabBarGate
  app/design-preview/page.tsx         CREATE — dev-only primitive gallery (both surfaces)
  lib/cn.ts                           MODIFY — clsx + tailwind-merge
  lib/category-numerals.ts            CREATE — slug → "01".."NN" from ordered categories
  lib/price-label.ts                  CREATE — kopecks/rubles → "FREE" | "800 ₽"
  lib/status-chip.ts                  CREATE — status string → chip variant
  lib/__tests__/category-numerals.test.ts   CREATE
  lib/__tests__/price-label.test.ts         CREATE
  lib/__tests__/status-chip.test.ts         CREATE
  components/ui/Chip.tsx              CREATE
  components/ui/Button.tsx            REWRITE — cta variants
  components/ui/Cell.tsx              CREATE — cap-over-val atomic module
  components/ui/AppHeader.tsx         CREATE — wordmark + nav (user/organizer/admin)
  components/ui/EventModule.tsx       CREATE — Swiss feed card (EventCard stays until Phase 2 swaps call sites)
  components/ui/BottomTabBar.tsx      CREATE — 4-tab mobile bar (TabBar.tsx deleted in Phase 2)
  components/ui/Field.tsx             CREATE — Input / Textarea / Select with cap label + error
  components/ui/ProgressBar.tsx       CREATE
  components/ui/StatusChip.tsx        CREATE
  components/ui/Stepper.tsx           CREATE
  components/ui/Skeleton.tsx          CREATE
  components/ui/EmptyState.tsx        CREATE — U8 pattern
  components/ui/TabBarGate.tsx        CREATE — mounts BottomTabBar on user-layer routes only
```

Deleted in this phase: nothing (old `EventCard`, `TabBar`, `GlassNav`, `ThemeSwitch` keep compiling until Phase 2 swaps their call sites — the app must build at every commit). `ThemeSwitch` usage in `app/page.tsx` is stubbed out in Task 2.

---

### Task 1: `cn()` upgrade (clsx + tailwind-merge)

Mass-restyling relies on `className` overrides actually winning; the naive joiner can't guarantee that.

**Files:**
- Modify: `frontend/lib/cn.ts`
- Test: `frontend/lib/__tests__/cn.test.ts`

**Interfaces:**
- Produces: `cn(...inputs: ClassValue[]): string` — same import path `@/lib/cn`, now conflict-resolving. All later tasks use it.

- [ ] **Step 1: Install deps**

```bash
pnpm add clsx tailwind-merge
```

- [ ] **Step 2: Write the failing test** — `frontend/lib/__tests__/cn.test.ts`

```ts
import { describe, expect, it } from "vitest";
import { cn } from "../cn";

describe("cn", () => {
  it("joins and drops falsy", () => {
    expect(cn("a", false, null, undefined, "b")).toBe("a b");
  });
  it("later tailwind class wins on conflict", () => {
    expect(cn("px-4 text-ink", "px-2")).toBe("text-ink px-2");
  });
});
```

- [ ] **Step 3: Run it to verify it fails**

Run: `pnpm vitest run lib/__tests__/cn.test.ts`
Expected: FAIL — conflict case returns `"px-4 text-ink px-2"`.

- [ ] **Step 4: Implement** — `frontend/lib/cn.ts`

```ts
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/** Classname joiner with Tailwind conflict resolution (last one wins). */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
```

- [ ] **Step 5: Run all tests**

Run: `pnpm test`
Expected: all suites PASS (existing 8 suites + cn).

- [ ] **Step 6: Commit**

```bash
git add lib/cn.ts lib/__tests__/cn.test.ts package.json pnpm-lock.yaml
git commit -m "feat(frontend): cn() with clsx + tailwind-merge for restyle safety"
```

---

### Task 2: Token layer — `globals.css` rewrite + fonts in `layout.tsx`

**Files:**
- Modify (full rewrite): `frontend/app/globals.css`
- Modify: `frontend/app/layout.tsx`
- Modify: `frontend/app/page.tsx` (remove `ThemeSwitch` import/usage — replace with `null`; GlassNav itself is replaced in Phase 2)

**Interfaces:**
- Produces (Tailwind utilities all later tasks use): colors `ink paper signal signal-tint muted muted-2 body-dim field-text rule-light rule-grid rule-dark admin-head text-dim-dark text-dim-dark-2 inactive table-head cell-blank surface on-surface rule-inner text-dim` (e.g. `bg-surface text-on-surface border-rule-inner`); fonts `font-ui font-alt font-mono`; CSS utility classes `cap`, `lbl`, `kick`, `swiss-focus`, `hover-invert`.
- Produces: `[data-surface="ink"]` scope — flips `surface/on-surface/rule-inner/text-dim/text-muted-x` semantic vars for admin + inverted blocks (U7 left panel, 404).

- [ ] **Step 1: Rewrite `frontend/app/globals.css`**

```css
@import "tailwindcss";

/*
 * Presence — Swiss Grid design tokens.
 * Source of truth: docs/Redesign/5/design_handoff_presence_swiss_grid/README.md
 * Identity: paper + ink, zero radius, zero shadows, 1px rules, tracked caps,
 * all numerals in mono. Red = needs attention, never decorative.
 */

:root {
  /* palette (verbatim from handoff tokens.css) */
  --ink: #111111;
  --paper: #f2f0ec;
  --white: #ffffff;
  --signal: #e2231a;
  --signal-tint: #ffd9d6;
  --muted: #7d786e;
  --muted-2: #8a857c;
  --body-dim: #4f4a42;
  --field-text: #6b665e;
  --rule-light: #dddddd;
  --rule-grid: #e0dcd4;
  --rule-dark: #3a3733;
  --admin-head: #1c1a18;
  --text-dim-dark: #cfcabf;
  --text-dim-dark-2: #a8a299;
  --inactive: #dcd8d0;
  --table-head: #e6e3dc;
  --cell-blank: #eceae4;

  /* semantic surface (paper by default; flipped by data-surface="ink") */
  --surface: var(--paper);
  --on-surface: var(--ink);
  --rule-inner: var(--rule-light);
  --text-dim: var(--body-dim);
  --surface-head: var(--table-head);
  --hover-fill: var(--ink);
  --hover-on: var(--paper);
}

/* Inverted surface — admin, U7 ink panel, 404. Components are surface-blind:
 * they use surface/on-surface/rule-inner and invert for free. */
[data-surface="ink"] {
  --surface: var(--ink);
  --on-surface: var(--paper);
  --rule-inner: var(--rule-dark);
  --text-dim: var(--text-dim-dark);
  --surface-head: var(--admin-head);
  --hover-fill: var(--admin-head);
  --hover-on: var(--paper);
}

@theme inline {
  --color-ink: var(--ink);
  --color-paper: var(--paper);
  --color-white: var(--white);
  --color-signal: var(--signal);
  --color-signal-tint: var(--signal-tint);
  --color-muted: var(--muted);
  --color-muted-2: var(--muted-2);
  --color-body-dim: var(--body-dim);
  --color-field-text: var(--field-text);
  --color-rule-light: var(--rule-light);
  --color-rule-grid: var(--rule-grid);
  --color-rule-dark: var(--rule-dark);
  --color-admin-head: var(--admin-head);
  --color-text-dim-dark: var(--text-dim-dark);
  --color-text-dim-dark-2: var(--text-dim-dark-2);
  --color-inactive: var(--inactive);
  --color-table-head: var(--table-head);
  --color-cell-blank: var(--cell-blank);
  --color-surface: var(--surface);
  --color-on-surface: var(--on-surface);
  --color-rule-inner: var(--rule-inner);
  --color-text-dim: var(--text-dim);
  --color-surface-head: var(--surface-head);

  /* fonts — variables provided by next/font in app/layout.tsx */
  --font-ui: var(--font-golos), "Helvetica Neue", Arial, sans-serif;
  --font-alt: var(--font-manrope), var(--font-golos), sans-serif;
  --font-mono: var(--font-jbmono), "SFMono-Regular", Menlo, monospace;

  /* the system has no radii and no shadows */
  --radius-*: initial;
  --shadow-*: initial;
}

html {
  background: var(--paper);
  color: var(--ink);
  font-family: var(--font-ui);
  -webkit-font-smoothing: antialiased;
  text-rendering: optimizeLegibility;
}

/* ——— Swiss type utilities ——— */

/* Caption: 9px / 0.13em / uppercase / muted-2. The workhorse label. */
@utility cap {
  font-family: var(--font-alt);
  font-size: 9px;
  font-weight: 400;
  letter-spacing: 0.13em;
  text-transform: uppercase;
  line-height: 1.3;
  color: var(--muted-2);
}

/* Label: 10px / 700 / 0.14em / uppercase, current colour. */
@utility lbl {
  font-family: var(--font-alt);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

/* Kicker: 11px / 700 / 0.18em / uppercase / muted. */
@utility kick {
  font-family: var(--font-alt);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--muted);
}

/* Square 2px focus outline on the current surface's on-colour. */
@utility swiss-focus {
  &:focus-visible {
    outline: 2px solid var(--on-surface);
    outline-offset: 0;
  }
}

/* Hover inverts — never tints. 120ms linear, background+color only. */
@utility hover-invert {
  transition: background-color 120ms linear, color 120ms linear;
  @media (hover: hover) {
    &:hover {
      background: var(--hover-fill);
      color: var(--hover-on);
    }
  }
}
```

- [ ] **Step 2: Rewrite `frontend/app/layout.tsx`**

```tsx
import type { Metadata } from "next";
import { Golos_Text, Manrope, JetBrains_Mono } from "next/font/google";
import { Providers } from "./providers";
import { TabBarGate } from "@/components/ui/TabBarGate";
import { VerifyEmailBanner } from "@/components/VerifyEmailBanner";
import "./globals.css";

/* Swiss Grid faces. The handoff specifies Archivo / Space Grotesk, which have
 * no Cyrillic on Google Fonts — Golos Text / Manrope are the Cyrillic-complete
 * substitutes (master plan, decision checkpoint 2). Self-hosted by next/font. */
const golos = Golos_Text({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "600", "700", "900"],
  variable: "--font-golos",
  display: "swap",
});
const manrope = Manrope({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "700"],
  variable: "--font-manrope",
  display: "swap",
});
const jbmono = JetBrains_Mono({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "700"],
  variable: "--font-jbmono",
  display: "swap",
});

export const metadata: Metadata = {
  title: "PRESENCE — События",
  description:
    "Медиации, лекции и разговоры об искусстве. Участливые культурные события Москвы.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      lang="ru"
      className={`h-full antialiased ${golos.variable} ${manrope.variable} ${jbmono.variable}`}
    >
      <body className="min-h-full bg-paper font-ui text-ink">
        <Providers>
          <VerifyEmailBanner />
          {children}
        </Providers>
        <TabBarGate />
      </body>
    </html>
  );
}
```

Notes: the pre-hydration theme script, `suppressHydrationWarning`, and `data-theme` are **removed** (single-theme system — master plan checkpoint 1). `TabBar` import is replaced by `TabBarGate` (Task 9 creates it; until then keep the old `<TabBar />` import so each commit builds — this file is finalized in Task 9, in this task only fonts/theme-script/body classes change).

- [ ] **Step 3: Stub `ThemeSwitch` usage**

In `app/page.tsx`, remove the `ThemeSwitch` import and its JSX usage (leave `GlassNav`/`AuthButton` untouched — Phase 2 replaces them). Grep for other usages:

Run: `grep -rn "ThemeSwitch" app components --include="*.tsx"`
Expected: zero remaining usages (the component file itself stays until Phase 2 deletes it).

- [ ] **Step 4: Build + test**

Run: `pnpm build && pnpm test`
Expected: build PASS (old screens will look wrong — that's expected; they must still compile). All Vitest suites PASS.
Known fallout to fix here if build errors: any `rounded-control`/`shadow-card` utilities now resolve to nothing (`--radius-*: initial`) — Tailwind v4 emits unknown-utility errors at build for these. If the build fails on a `rounded-*`/`shadow-*` token utility, re-add a temporary compatibility block at the bottom of `@theme inline` (to delete in Phase 2):

```css
  /* TEMP Phase-1 compat: legacy radii/shadows resolve to nothing (flat). Remove in Phase 2. */
  --radius-control: 0px;
  --radius-card-sm: 0px;
  --radius-card: 0px;
  --radius-fact: 0px;
  --radius-sheet: 0px;
  --radius-seg: 0px;
  --radius-capsule: 0px;
  --shadow-card: 0 0 #0000;
  --shadow-card-subtle: 0 0 #0000;
  --shadow-phone: 0 0 #0000;
  --shadow-glass: 0 0 #0000;
```

Also: the `glass` utility was deleted — `GlassNav`, `TabBar`, `AuthButton`, `ConfirmModal`, `admin/layout.tsx` reference `glass` as a plain class name (string), which Tailwind v4 treats as unknown at build. If the build rejects it, add `@utility glass { background: var(--paper); border-bottom: 1px solid var(--ink); }` as a TEMP shim with the same removal note.

- [ ] **Step 5: Verify fonts in the browser**

Run: `pnpm dev`, open `http://localhost:3000` (Claude in Chrome). Confirm: page background `#F2F0EC`; «Летний фестиваль медиаискусства» (feed titles) renders in Golos Text (inspect computed `font-family`), digits in any mono context render JetBrains Mono. No FOUT into Times/system serif for Cyrillic.

- [ ] **Step 6: Commit**

```bash
git add app/globals.css app/layout.tsx app/page.tsx
git commit -m "feat(frontend): Swiss Grid token layer, Cyrillic webfonts, retire dark mode"
```

---

### Task 3: Pure helpers — category numerals, price label, status→chip map (TDD)

**Files:**
- Create: `frontend/lib/category-numerals.ts`, `frontend/lib/price-label.ts`, `frontend/lib/status-chip.ts`
- Test: `frontend/lib/__tests__/category-numerals.test.ts`, `price-label.test.ts`, `status-chip.test.ts`

**Interfaces:**
- Produces: `categoryNumeral(slug: string, ordered: ReadonlyArray<{ slug: string }>): string` — `"01"`-based positional numeral, `"—"` for unknown.
- Produces: `priceLabel(priceMinor: number | null | undefined): string` — `"FREE"` for null/0, else `"1 500 ₽"` (non-breaking thin space grouping, ruble sign; input is minor units/kopecks ÷ 100 as used by `lib/api.ts` — **verify the unit in `apiEventToLia` before implementing; adjust divisor accordingly**).
- Produces: `statusChipVariant(status: string): "active" | "default" | "signal"` per the handoff map (Опубликовано/Подтверждено/Верифицирован → active; Черновик/Прошедшее → default; На модерации/Ожидает/На проверке/Тестовый → signal; unknown → default).

- [ ] **Step 1: Write the failing tests**

`frontend/lib/__tests__/category-numerals.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { categoryNumeral } from "../category-numerals";

const CATS = [{ slug: "festival" }, { slug: "mediation" }, { slug: "lecture" }];

describe("categoryNumeral", () => {
  it("is 1-based and zero-padded", () => {
    expect(categoryNumeral("festival", CATS)).toBe("01");
    expect(categoryNumeral("lecture", CATS)).toBe("03");
  });
  it("unknown slug → em dash", () => {
    expect(categoryNumeral("nope", CATS)).toBe("—");
  });
  it("pads to width 2 up to 99", () => {
    const many = Array.from({ length: 12 }, (_, i) => ({ slug: `c${i}` }));
    expect(categoryNumeral("c11", many)).toBe("12");
  });
});
```

`frontend/lib/__tests__/price-label.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { priceLabel } from "../price-label";

describe("priceLabel", () => {
  it("free is the literal FREE", () => {
    expect(priceLabel(0)).toBe("FREE");
    expect(priceLabel(null)).toBe("FREE");
    expect(priceLabel(undefined)).toBe("FREE");
  });
  it("formats rubles with narrow nbsp grouping", () => {
    expect(priceLabel(80000)).toBe("800 ₽");
    expect(priceLabel(150000)).toBe("1 500 ₽");
  });
});
```

`frontend/lib/__tests__/status-chip.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { statusChipVariant } from "../status-chip";

describe("statusChipVariant", () => {
  it("published family → active", () => {
    for (const s of ["Опубликовано", "Подтверждено", "Верифицирован"])
      expect(statusChipVariant(s)).toBe("active");
  });
  it("draft/past → default", () => {
    for (const s of ["Черновик", "Прошедшее"])
      expect(statusChipVariant(s)).toBe("default");
  });
  it("attention family → signal", () => {
    for (const s of ["На модерации", "Ожидает", "На проверке", "Тестовый"])
      expect(statusChipVariant(s)).toBe("signal");
  });
  it("unknown → default", () => {
    expect(statusChipVariant("Что-то ещё")).toBe("default");
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `pnpm vitest run lib/__tests__/category-numerals.test.ts lib/__tests__/price-label.test.ts lib/__tests__/status-chip.test.ts`
Expected: FAIL — modules not found.

- [ ] **Step 3: Implement**

`frontend/lib/category-numerals.ts`:

```ts
/** Positional category numeral per the Swiss Grid rule: categories are
 * numerals, never colours. Order comes from GET /api/v1/categories. */
export function categoryNumeral(
  slug: string,
  ordered: ReadonlyArray<{ slug: string }>,
): string {
  const i = ordered.findIndex((c) => c.slug === slug);
  if (i === -1) return "—";
  return String(i + 1).padStart(2, "0");
}
```

`frontend/lib/price-label.ts`:

```ts
const RUB = new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 0 });

/** Swiss Grid price string: free is the literal FREE (bottom-right of every
 * EventModule); otherwise grouped rubles + nbsp + ₽. Input in kopecks. */
export function priceLabel(priceMinor: number | null | undefined): string {
  if (!priceMinor) return "FREE";
  return `${RUB.format(Math.round(priceMinor / 100))} ₽`;
}
```

(If `apiEventToLia` turns out to store whole rubles, drop the `/ 100` and fix the test inputs — the contract is the output strings.)

`frontend/lib/status-chip.ts`:

```ts
export type ChipTone = "active" | "default" | "signal";

const ACTIVE = new Set(["Опубликовано", "Подтверждено", "Верифицирован"]);
const SIGNAL = new Set(["На модерации", "Ожидает", "На проверке", "Тестовый"]);

/** Handoff status→chip map. Red strictly means "needs attention". */
export function statusChipVariant(status: string): ChipTone {
  if (ACTIVE.has(status)) return "active";
  if (SIGNAL.has(status)) return "signal";
  return "default";
}
```

- [ ] **Step 4: Run tests**

Run: `pnpm test` — all PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/category-numerals.ts lib/price-label.ts lib/status-chip.ts lib/__tests__/
git commit -m "feat(frontend): swiss-grid pure helpers — category numerals, price label, status chips"
```

---

### Task 4: `Chip`

**Files:**
- Create: `frontend/components/ui/Chip.tsx`

**Interfaces:**
- Produces: `<Chip variant?: "default"|"active"|"signal"|"dark-active"|"dark-muted", as?: "button"|"span", …buttonProps>` — used by every filter row, status display, and tab-chip strip in Phases 2–6. `StatusChip` (Task 8) wraps it.

- [ ] **Step 1: Implement** — `frontend/components/ui/Chip.tsx`

```tsx
import { cn } from "@/lib/cn";
import type { ButtonHTMLAttributes } from "react";

export type ChipVariant =
  | "default" // 1px currentColor border, transparent
  | "active" // ink fill, paper text
  | "signal" // red border + text
  | "dark-active" // paper fill, ink text (on ink surface)
  | "dark-muted"; // muted border + text (on ink surface)

const VARIANTS: Record<ChipVariant, string> = {
  default: "border-current",
  active: "border-ink bg-ink text-paper",
  signal: "border-signal text-signal",
  "dark-active": "border-paper bg-paper text-ink",
  "dark-muted": "border-muted-2 text-muted-2",
};

export interface ChipProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ChipVariant;
  /** Render a non-interactive span (status display) instead of a button. */
  as?: "button" | "span";
}

/** Swiss Grid chip: 9px / 0.12em / uppercase / 1px border / zero radius.
 * Counters live inside the label: `Все · 6`. Interactive chips hover-invert. */
export function Chip({ variant = "default", as = "button", className, children, ...props }: ChipProps) {
  const base = cn(
    "inline-flex items-center whitespace-nowrap border px-[9px] py-[4px] font-alt text-[9px] uppercase tracking-[0.12em]",
    VARIANTS[variant],
    as === "button" && "cursor-pointer swiss-focus hover-invert",
    className,
  );
  if (as === "span") return <span className={base}>{children}</span>;
  return (
    <button type="button" className={base} {...props}>
      {children}
    </button>
  );
}
```

- [ ] **Step 2: Typecheck/lint**

Run: `pnpm lint && npx tsc --noEmit`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add components/ui/Chip.tsx
git commit -m "feat(frontend): Chip primitive (swiss grid)"
```

---

### Task 5: `Button` rewrite (`.cta`)

**Files:**
- Modify (full rewrite): `frontend/components/ui/Button.tsx`

**Interfaces:**
- Consumes: `cn` (Task 1).
- Produces: `<Button variant?: "primary"|"ghost"|"inverted"|"destructive"|"dark-ghost", size?: "md"|"sm", …buttonProps>`. **Breaking change:** old variants `filled/tinted/plain` are gone. Callers keep compiling only if they don't pass `variant` (default) — Step 2 maps existing call sites.

- [ ] **Step 1: Implement** — `frontend/components/ui/Button.tsx`

```tsx
import { cn } from "@/lib/cn";
import type { ButtonHTMLAttributes } from "react";

type Variant = "primary" | "ghost" | "inverted" | "destructive" | "dark-ghost";
type Size = "md" | "sm";

const VARIANTS: Record<Variant, string> = {
  // Ink fill, white text; hover deepens to black.
  primary: "bg-ink text-white hover:bg-black",
  // Transparent, 1px ink border, ink text; hover inverts.
  ghost: "border border-ink text-ink hover-invert",
  // Admin primary on ink surface: paper fill, ink text.
  inverted: "bg-paper text-ink hover:opacity-90",
  // Red fill — destructive only (ОТКЛОНИТЬ etc.).
  destructive: "bg-signal text-white hover:opacity-90",
  // Admin tertiary on ink surface.
  "dark-ghost": "border border-muted-2 text-muted-2 hover:opacity-90",
};

const SIZES: Record<Size, string> = {
  md: "px-[11px] py-[11px] text-[11px]",
  sm: "px-[4px] py-[7px] text-[9px]",
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
}

/** Swiss Grid CTA: uppercase 700 / 0.07em, zero radius, no motion. */
export function Button({ variant = "primary", size = "md", className, ...props }: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center whitespace-nowrap text-center font-bold uppercase tracking-[0.07em] transition-colors duration-[120ms] ease-linear select-none swiss-focus disabled:bg-inactive disabled:text-muted-2 disabled:border-0",
        VARIANTS[variant],
        SIZES[size],
        className,
      )}
      {...props}
    />
  );
}
```

- [ ] **Step 2: Migrate existing variant usages**

Run: `grep -rln 'variant="tinted"\|variant="plain"\|variant="filled"' app components`
Map mechanically: `filled` → remove (default `primary`), `tinted` → `variant="ghost"`, `plain` → `variant="ghost"` (Phase 2+ refines per screen; the goal here is compile + coherent interim look).

- [ ] **Step 3: Build + test**

Run: `pnpm build && pnpm test` — PASS.

- [ ] **Step 4: Commit**

```bash
git add components/ui/Button.tsx app components
git commit -m "feat(frontend): Button rewritten to swiss-grid cta variants"
```

---

### Task 6: `Cell` + `ProgressBar` + `Skeleton`

**Files:**
- Create: `frontend/components/ui/Cell.tsx`, `frontend/components/ui/ProgressBar.tsx`, `frontend/components/ui/Skeleton.tsx`

**Interfaces:**
- Produces: `<Cell caption value mono? dense? invert? className?>`; `<CellStrip cols n?>` wrapper producing the bordered `repeat(N,1fr)` strip.
- Produces: `<ProgressBar value max thin?>` — 8px (or 5–7px thin) ink-bordered bar, ink fill at `value/max`.
- Produces: `<Skeleton className?>` — `#ECEAE4` block at final dimensions (no shimmer); numbers render `—` while loading (caller's job).

- [ ] **Step 1: Implement** — `frontend/components/ui/Cell.tsx`

```tsx
import { cn } from "@/lib/cn";
import type { ReactNode } from "react";

export interface CellProps {
  caption: ReactNode;
  value: ReactNode;
  /** Numeric value → JetBrains Mono (handoff: ALL numbers in mono). */
  mono?: boolean;
  /** Roomy 16/20 padding instead of dense 10/14. */
  roomy?: boolean;
  /** Ink-filled emphasis cell (e.g. O1 «На модерации»). */
  invert?: boolean;
  className?: string;
}

/** The atomic module of the system: uppercase caption over a bold value. */
export function Cell({ caption, value, mono, roomy, invert, className }: CellProps) {
  return (
    <div
      className={cn(
        "flex min-w-0 flex-col gap-[4px]",
        roomy ? "px-[20px] py-[16px]" : "px-[14px] py-[10px]",
        invert && "bg-ink text-paper",
        className,
      )}
    >
      <span className={cn("cap", invert && "text-text-dim-dark-2")}>{caption}</span>
      <span className={cn("text-[12px] font-bold leading-[1.25]", mono && "font-mono")}>
        {value}
      </span>
    </div>
  );
}

/** Horizontal strip of cells divided by hairlines, closed by a bottom rule. */
export function CellStrip({ children, cols, className }: { children: ReactNode; cols: number; className?: string }) {
  return (
    <div
      className={cn("grid border-b border-on-surface [&>*+*]:border-l [&>*+*]:border-rule-inner", className)}
      style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}
    >
      {children}
    </div>
  );
}
```

- [ ] **Step 2: Implement** — `frontend/components/ui/ProgressBar.tsx`

```tsx
import { cn } from "@/lib/cn";

export interface ProgressBarProps {
  value: number;
  max: number;
  /** 5px in-row variant (O3 seat bars) instead of the 8px default. */
  thin?: boolean;
  className?: string;
}

/** Ink-bordered fill bar. No radius, no gradient, no animation. */
export function ProgressBar({ value, max, thin, className }: ProgressBarProps) {
  const pct = max > 0 ? Math.min(100, Math.max(0, (value / max) * 100)) : 0;
  return (
    <div
      role="progressbar"
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={max}
      className={cn("w-full border border-ink", thin ? "h-[5px]" : "h-[8px]", className)}
    >
      <div className="h-full bg-ink" style={{ width: `${pct}%` }} />
    </div>
  );
}
```

- [ ] **Step 3: Implement** — `frontend/components/ui/Skeleton.tsx`

```tsx
import { cn } from "@/lib/cn";

/** Loading placeholder: hairline-boxed block at final dimensions.
 * Never a spinner, never shimmer (handoff → States → Loading). */
export function Skeleton({ className }: { className?: string }) {
  return <div aria-hidden className={cn("border border-rule-inner bg-cell-blank", className)} />;
}
```

- [ ] **Step 4: Lint + commit**

Run: `pnpm lint && npx tsc --noEmit`

```bash
git add components/ui/Cell.tsx components/ui/ProgressBar.tsx components/ui/Skeleton.tsx
git commit -m "feat(frontend): Cell/CellStrip, ProgressBar, Skeleton primitives"
```

---

### Task 7: `AppHeader`

**Files:**
- Create: `frontend/components/ui/AppHeader.tsx`

**Interfaces:**
- Consumes: `cn`.
- Produces: `<AppHeader nav={Array<{ href: string; label: string }>} admin? mobileCaption? actions?>` — client component; active item derived from `usePathname()` (exact match or prefix for nested routes). Used by every screen phase; admin variant renders `PRESENCE / ADMIN` and expects to sit inside `[data-surface="ink"]`.

- [ ] **Step 1: Implement** — `frontend/components/ui/AppHeader.tsx`

```tsx
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/cn";
import type { ReactNode } from "react";

export interface NavItem {
  href: string;
  label: string;
}

export interface AppHeaderProps {
  nav?: NavItem[];
  /** Admin variant: wordmark PRESENCE / ADMIN, paper bottom rule (inside data-surface="ink"). */
  admin?: boolean;
  /** Mobile shows a context caption instead of nav. */
  mobileCaption?: string;
  /** Optional right-side extras (auth control) appended after nav. */
  actions?: ReactNode;
}

function isActive(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

/** Swiss Grid header: wordmark left, 9px tracked uppercase nav right,
 * active item = 2px bottom rule in currentColor. */
export function AppHeader({ nav = [], admin, mobileCaption, actions }: AppHeaderProps) {
  const pathname = usePathname();
  return (
    <header
      className={cn(
        "flex items-baseline justify-between border-b px-[20px] py-[13px] max-sm:px-[14px] max-sm:py-[11px]",
        admin ? "border-paper" : "border-ink",
      )}
    >
      <Link href="/" className="swiss-focus text-[13px] font-black tracking-[-0.01em] max-sm:text-[11px]">
        PRESENCE{admin ? " / ADMIN" : ""}
      </Link>
      <nav className="flex items-baseline gap-[14px] max-sm:hidden">
        {nav.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            aria-current={isActive(pathname, item.href) ? "page" : undefined}
            className={cn(
              "swiss-focus font-alt text-[9px] uppercase tracking-[0.14em]",
              isActive(pathname, item.href) && "border-b-2 border-current pb-[2px] font-bold",
            )}
          >
            {item.label}
          </Link>
        ))}
        {actions}
      </nav>
      {mobileCaption ? <span className="cap sm:hidden">{mobileCaption}</span> : null}
    </header>
  );
}

/** Canonical nav sets (handoff → Interactions → Navigation). */
export const USER_NAV: NavItem[] = [
  { href: "/", label: "События" },
  { href: "/search", label: "Подбор" },
  { href: "/me/calendar", label: "Календарь" },
  { href: "/map", label: "Карта" },
  { href: "/organizer", label: "Организаторам" },
];
export const ORG_NAV: NavItem[] = [
  { href: "/organizer", label: "Кабинет" },
  { href: "/events/mine", label: "Мои события" },
  { href: "/organizer/applications", label: "Заявки" },
  { href: "/me/organizer", label: "Профиль" },
];
export const ADMIN_NAV: NavItem[] = [
  { href: "/admin", label: "Обзор" },
  { href: "/admin/moderation/events", label: "Модерация" },
  { href: "/admin/organizers", label: "Организаторы" },
  { href: "/admin/users", label: "Пользователи" },
];
```

(«Войти» is appended via `actions` with the restyled auth control in Phase 2 — auth state is out of scope here.)

- [ ] **Step 2: Lint + commit**

Run: `pnpm lint && npx tsc --noEmit`

```bash
git add components/ui/AppHeader.tsx
git commit -m "feat(frontend): AppHeader with user/organizer/admin nav sets"
```

---

### Task 8: `EventModule` + `StatusChip`

**Files:**
- Create: `frontend/components/ui/EventModule.tsx`, `frontend/components/ui/StatusChip.tsx`

**Interfaces:**
- Consumes: `categoryNumeral` output (caller passes the resolved `numeral` string), `priceLabel` output (caller passes `price` string), `Chip`, `statusChipVariant`.
- Produces: `<EventModule numeral category title venue date price href matchReason? mobile?>` — the feed card. Whole module is ONE `<Link>` (preserves the React #418 no-nested-anchor fix — nothing inside may be an `<a>`).
- Produces: `<StatusChip status>` — `Chip as="span"` with variant from `statusChipVariant(status)`.

- [ ] **Step 1: Implement** — `frontend/components/ui/EventModule.tsx`

```tsx
import Link from "next/link";
import { cn } from "@/lib/cn";

export interface EventModuleProps {
  numeral: string; // "01" — from categoryNumeral()
  category: string; // "Фестивали"
  title: string;
  venue: string;
  date: string; // preformatted, renders in mono
  price: string; // from priceLabel() — "FREE" | "800 ₽"
  href: string;
  /** U3 AI variant: match reason on a hairline top rule. */
  matchReason?: string;
  className?: string;
}

/** Swiss Grid feed card. Desktop: numeral/category row, 15px/900 title,
 * venue caption, footer date(mono)+price pinned bottom. Hover inverts.
 * Single <Link> wrapper — never nest an <a> inside (React #418 fix). */
export function EventModule({
  numeral, category, title, venue, date, price, href, matchReason, className,
}: EventModuleProps) {
  return (
    <Link
      href={href}
      className={cn(
        "flex min-w-0 flex-col px-[14px] py-[12px] swiss-focus hover-invert",
        className,
      )}
    >
      <div className="flex items-baseline justify-between">
        <span className="font-mono text-[11px] font-bold">{numeral}</span>
        <span className="cap">{category}</span>
      </div>
      <h3 className="mt-[8px] text-[15px] font-black leading-[1.02] tracking-[-0.02em] max-sm:text-[12.5px] max-sm:font-bold">
        {title}
      </h3>
      <p className="cap mt-[6px]">{venue}</p>
      <div className="mt-auto flex items-baseline justify-between pt-[10px]">
        <span className="font-mono text-[11px]">{date}</span>
        <span className="text-[12px] font-black">{price}</span>
      </div>
      {matchReason ? (
        <p className="mt-[8px] border-t border-rule-inner pt-[8px] text-[10px]">
          Совпало: {matchReason}
        </p>
      ) : null}
    </Link>
  );
}
```

- [ ] **Step 2: Implement** — `frontend/components/ui/StatusChip.tsx`

```tsx
import { Chip, type ChipVariant } from "@/components/ui/Chip";
import { statusChipVariant } from "@/lib/status-chip";

const TONE_TO_VARIANT: Record<ReturnType<typeof statusChipVariant>, ChipVariant> = {
  active: "active",
  default: "default",
  signal: "signal",
};

/** Status chip per the handoff map (published→ink fill, draft→outline,
 * moderation/waiting/test→signal). Non-interactive. */
export function StatusChip({ status, className }: { status: string; className?: string }) {
  return (
    <Chip as="span" variant={TONE_TO_VARIANT[statusChipVariant(status)]} className={className}>
      {status}
    </Chip>
  );
}
```

- [ ] **Step 3: Lint + test + commit**

Run: `pnpm lint && npx tsc --noEmit && pnpm test`

```bash
git add components/ui/EventModule.tsx components/ui/StatusChip.tsx
git commit -m "feat(frontend): EventModule and StatusChip primitives"
```

---

### Task 9: `BottomTabBar` + `TabBarGate` + finalize `layout.tsx`

**Files:**
- Create: `frontend/components/ui/BottomTabBar.tsx`, `frontend/components/ui/TabBarGate.tsx`
- Modify: `frontend/app/layout.tsx` (swap `TabBar` → `TabBarGate` — completes Task 2's note)

**Interfaces:**
- Produces: `<BottomTabBar />` — fixed bottom, mobile-only (`sm:hidden`), 4 tabs **Лента `/` · Подбор `/search` · Карта `/map` · Я `/me/practices`** (`/me/practices` until the U6 `/me` page exists in Phase 3 — then one-line change). Square 16×16 icon (bordered, filled when active), 8px/0.1em label.
- Produces: `<TabBarGate />` — client; renders `BottomTabBar` only on user-layer routes: hidden when `pathname` starts with `/admin`, `/organizer`, `/events/new`, `/events/mine`, `/login`, `/signup`, `/auth`, or matches `/events/[id]/edit`.

- [ ] **Step 1: Implement** — `frontend/components/ui/BottomTabBar.tsx`

```tsx
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/cn";

const TABS = [
  { href: "/", label: "Лента" },
  { href: "/search", label: "Подбор" },
  { href: "/map", label: "Карта" },
  { href: "/me/practices", label: "Я" }, // → "/me" after Phase 3 builds U6
] as const;

function isActive(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`) || (href === "/me/practices" && pathname.startsWith("/me"));
}

/** Mobile-only 4-tab bar. Icons are plain squares by design — the system has
 * no icon library. Active tab: filled square + bold ink label. */
export function BottomTabBar() {
  const pathname = usePathname();
  return (
    <nav className="fixed inset-x-0 bottom-0 z-40 flex border-t border-ink bg-paper sm:hidden">
      {TABS.map((tab) => {
        const active = isActive(pathname, tab.href);
        return (
          <Link
            key={tab.href}
            href={tab.href}
            aria-current={active ? "page" : undefined}
            className={cn(
              "flex min-h-[48px] flex-1 flex-col items-center justify-center gap-[4px] py-[6px] swiss-focus",
              active ? "font-bold text-ink" : "text-muted-2",
            )}
          >
            <span
              aria-hidden
              className={cn("h-[16px] w-[16px] border-[1.5px] border-current", active && "bg-current")}
            />
            <span className="font-alt text-[8px] uppercase tracking-[0.1em]">{tab.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
```

- [ ] **Step 2: Implement** — `frontend/components/ui/TabBarGate.tsx`

```tsx
"use client";

import { usePathname } from "next/navigation";
import { BottomTabBar } from "@/components/ui/BottomTabBar";

const HIDDEN_PREFIXES = [
  "/admin",
  "/organizer",
  "/events/new",
  "/events/mine",
  "/login",
  "/signup",
  "/auth",
];

/** The tab bar is user-layer navigation only (handoff → Navigation):
 * organizer/admin mobile use header context, auth screens are chromeless. */
export function TabBarGate() {
  const pathname = usePathname();
  if (HIDDEN_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`))) return null;
  if (/^\/events\/[^/]+\/edit/.test(pathname)) return null;
  return <BottomTabBar />;
}
```

- [ ] **Step 3: Swap in `app/layout.tsx`** — replace the `TabBar` import/JSX with `TabBarGate` (per Task 2's target listing).

- [ ] **Step 4: Build + verify + commit**

Run: `pnpm build && pnpm test`
Browser: at 390px the bar shows on `/` and `/map`, hides on `/organizer` and `/admin`; tabs are ≥44px tall.

```bash
git add components/ui/BottomTabBar.tsx components/ui/TabBarGate.tsx app/layout.tsx
git commit -m "feat(frontend): BottomTabBar (4 tabs) gated to user-layer routes"
```

---

### Task 10: `Field` (Input / Textarea / Select), `Stepper`, `EmptyState`

**Files:**
- Create: `frontend/components/ui/Field.tsx`, `frontend/components/ui/Stepper.tsx`, `frontend/components/ui/EmptyState.tsx`

**Interfaces:**
- Produces: `<Input label error? …inputProps>`, `<Textarea label error? …textareaProps>`, `<Select label error? …selectProps>` — cap label above, 1px ink box, 12.5px text, error flips border+label+message to signal. Kills the 13-file duplicated input class as Phases 2–6 migrate call sites.
- Produces: `<Stepper steps: string[] current: number>` — equal-width segments, index < current → completed (ink fill), index === current → active (ink fill), rest paper; 1px dividers.
- Produces: `<EmptyState numeral? title text? actions?>` — the U8 pattern (mono numeral, 17px/900 headline, 11.5px explanation, ≤2 actions, self-start).

- [ ] **Step 1: Implement** — `frontend/components/ui/Field.tsx`

```tsx
import { cn } from "@/lib/cn";
import { useId } from "react";
import type { InputHTMLAttributes, SelectHTMLAttributes, TextareaHTMLAttributes } from "react";

const BOX =
  "w-full border bg-transparent px-[11px] py-[9px] text-[12.5px] text-ink placeholder:text-field-text swiss-focus disabled:bg-inactive disabled:text-muted-2";

function boxClass(error?: string) {
  return cn(BOX, error ? "border-signal" : "border-ink");
}

function Wrap({ id, label, error, children }: { id: string; label: string; error?: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-[5px]">
      <label htmlFor={id} className={cn("cap", error && "text-signal")}>
        {label}
      </label>
      {children}
      {error ? <p className="text-[11px] text-signal">{error}</p> : null}
    </div>
  );
}

export function Input({ label, error, className, id, ...props }: { label: string; error?: string } & InputHTMLAttributes<HTMLInputElement>) {
  const auto = useId();
  const inputId = id ?? auto;
  return (
    <Wrap id={inputId} label={label} error={error}>
      <input id={inputId} className={cn(boxClass(error), className)} {...props} />
    </Wrap>
  );
}

export function Textarea({ label, error, className, id, ...props }: { label: string; error?: string } & TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const auto = useId();
  const inputId = id ?? auto;
  return (
    <Wrap id={inputId} label={label} error={error}>
      <textarea id={inputId} className={cn(boxClass(error), "min-h-[52px] resize-y", className)} {...props} />
    </Wrap>
  );
}

export function Select({ label, error, className, id, children, ...props }: { label: string; error?: string } & SelectHTMLAttributes<HTMLSelectElement>) {
  const auto = useId();
  const inputId = id ?? auto;
  return (
    <Wrap id={inputId} label={label} error={error}>
      <select id={inputId} className={cn(boxClass(error), "appearance-none", className)} {...props}>
        {children}
      </select>
    </Wrap>
  );
}
```

- [ ] **Step 2: Implement** — `frontend/components/ui/Stepper.tsx`

```tsx
import { cn } from "@/lib/cn";

/** O2/O5 stepper: equal-width segments; completed + active are ink-filled
 * with paper text, upcoming stay on paper; 1px rules divide. */
export function Stepper({ steps, current, className }: { steps: string[]; current: number; className?: string }) {
  return (
    <ol className={cn("grid border-y border-ink [&>*+*]:border-l [&>*+*]:border-ink", className)} style={{ gridTemplateColumns: `repeat(${steps.length}, 1fr)` }}>
      {steps.map((step, i) => (
        <li
          key={step}
          aria-current={i === current ? "step" : undefined}
          className={cn("flex min-w-0 flex-col gap-[4px] px-[14px] py-[10px]", i <= current && "bg-ink text-paper")}
        >
          <span className={cn("cap", i <= current && "text-text-dim-dark-2")}>
            {String(i + 1).padStart(2, "0")}
          </span>
          <span className="truncate text-[12px] font-bold">{step}</span>
        </li>
      ))}
    </ol>
  );
}
```

- [ ] **Step 3: Implement** — `frontend/components/ui/EmptyState.tsx`

```tsx
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

/** U8 pattern: name the situation, explain in one sentence, offer exactly one
 * obvious next action (two at most). Used for empty / gated / error surfaces. */
export function EmptyState({
  numeral = "00",
  title,
  text,
  actions,
  className,
}: {
  numeral?: string;
  title: string;
  text?: string;
  actions?: ReactNode; // pass ≤2 <Button>s / links
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col items-start gap-[10px] px-[20px] py-[26px]", className)}>
      <span className="font-mono text-[38px] font-bold leading-none">{numeral}</span>
      <h2 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">{title}</h2>
      {text ? <p className="max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim">{text}</p> : null}
      {actions ? <div className="mt-[6px] flex gap-[8px]">{actions}</div> : null}
    </div>
  );
}
```

- [ ] **Step 4: Lint + commit**

Run: `pnpm lint && npx tsc --noEmit`

```bash
git add components/ui/Field.tsx components/ui/Stepper.tsx components/ui/EmptyState.tsx
git commit -m "feat(frontend): Field/Stepper/EmptyState primitives"
```

---

### Task 11: `/design-preview` gallery + browser verification

**Files:**
- Create: `frontend/app/design-preview/page.tsx`

**Interfaces:**
- Consumes: every primitive from Tasks 4–10 + helpers from Task 3.

- [ ] **Step 1: Implement** — `frontend/app/design-preview/page.tsx`

```tsx
import { notFound } from "next/navigation";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { Chip } from "@/components/ui/Chip";
import { Button } from "@/components/ui/Button";
import { Cell, CellStrip } from "@/components/ui/Cell";
import { EventModule } from "@/components/ui/EventModule";
import { StatusChip } from "@/components/ui/StatusChip";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Skeleton } from "@/components/ui/Skeleton";
import { Stepper } from "@/components/ui/Stepper";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input, Textarea } from "@/components/ui/Field";
import { priceLabel } from "@/lib/price-label";
import { categoryNumeral } from "@/lib/category-numerals";

const CATS = [
  { slug: "festival" }, { slug: "mediation" }, { slug: "lecture" },
  { slug: "cinema" }, { slug: "performance" }, { slug: "concert" },
];

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="border-b border-on-surface p-[20px]">
      <p className="kick mb-[14px]">{title}</p>
      <div className="flex flex-wrap items-start gap-[10px]">{children}</div>
    </section>
  );
}

function Gallery() {
  return (
    <>
      <Section title="Chips">
        <Chip>Все · 6</Chip>
        <Chip variant="active">Сегодня</Chip>
        <Chip variant="signal">Модерация · 2</Chip>
        <StatusChip status="Опубликовано" />
        <StatusChip status="Черновик" />
        <StatusChip status="На модерации" />
      </Section>
      <Section title="Buttons">
        <Button>Записаться</Button>
        <Button variant="ghost">Черновик</Button>
        <Button variant="destructive">Отклонить</Button>
        <Button variant="inverted">Одобрить</Button>
        <Button variant="dark-ghost">На доработку</Button>
        <Button size="sm">Принять</Button>
        <Button disabled>Недоступно</Button>
      </Section>
      <Section title="Cells">
        <CellStrip cols={4} className="w-full">
          <Cell caption="Когда" value="12 июля, сб" />
          <Cell caption="Начало" value="16:00" mono />
          <Cell caption="Места" value="12 / 40" mono />
          <Cell caption="На модерации" value="02" mono invert />
        </CellStrip>
      </Section>
      <Section title="EventModule">
        <div className="grid w-full grid-cols-3 border-y border-on-surface [&>*+*]:border-l [&>*+*]:border-rule-inner max-sm:grid-cols-1">
          <EventModule
            numeral={categoryNumeral("festival", CATS)}
            category="Фестивали" title="Летний фестиваль медиаискусства"
            venue="Музей «Гараж»" date="12.07 · 16:00" price={priceLabel(0)} href="/"
          />
          <EventModule
            numeral={categoryNumeral("lecture", CATS)}
            category="Лекции" title="Разговор о новой вещественности"
            venue="ГМИИ им. Пушкина" date="15.07 · 19:00" price={priceLabel(80000)} href="/"
            matchReason="тихое, вечером, вдвоём"
          />
          <EventModule
            numeral={categoryNumeral("mediation", CATS)}
            category="Медиации" title="Медиация по выставке «Свет»"
            venue="Винзавод" date="18.07 · 12:00" price={priceLabel(150000)} href="/"
          />
        </div>
      </Section>
      <Section title="Progress / Skeleton">
        <div className="w-[200px]"><ProgressBar value={28} max={40} /></div>
        <div className="w-[200px]"><ProgressBar value={28} max={40} thin /></div>
        <Skeleton className="h-[40px] w-[200px]" />
      </Section>
      <Section title="Stepper">
        <Stepper className="w-full" steps={["Основное", "Когда и где", "Билеты", "Публикация"]} current={1} />
      </Section>
      <Section title="Fields">
        <div className="grid w-full max-w-[400px] gap-[11px]">
          <Input label="Название" placeholder="Название события" />
          <Input label="Почта" error="Неверный формат почты" defaultValue="user@" />
          <Textarea label="Описание" placeholder="Коротко о событии" />
        </div>
      </Section>
      <Section title="Empty state (U8)">
        <EmptyState
          title="Записей пока нет"
          text="Когда вы запишетесь на событие, оно появится здесь."
          actions={<Button>Найти событие</Button>}
        />
      </Section>
    </>
  );
}

/** Dev-only primitive gallery: paper surface + ink surface side by side. */
export default function DesignPreviewPage() {
  if (process.env.NODE_ENV === "production") notFound();
  return (
    <main className="mx-auto max-w-[1360px]">
      <AppHeader nav={USER_NAV} />
      <div className="grid grid-cols-2 max-lg:grid-cols-1">
        <div className="bg-surface text-on-surface">
          <Gallery />
        </div>
        <div data-surface="ink" className="bg-surface text-on-surface">
          <Gallery />
        </div>
      </div>
    </main>
  );
}
```

- [ ] **Step 2: Full verification**

Run: `pnpm build && pnpm test && pnpm lint`
Browser (Claude in Chrome), `pnpm dev` → `http://localhost:3000/design-preview`, at desktop and 390px, check against `Presence Swiss Grid - Full System.dc.html` opened side-by-side:
- zero rounded corners / shadows anywhere; hairlines 1px
- Cyrillic headings in Golos Text (not fallback), all numbers in JetBrains Mono
- tracked uppercase on caps/labels/chips/buttons
- hover on chips/EventModule inverts to ink (paper text); on the ink half inverts to `#1C1A18`
- keyboard Tab shows 2px square outlines
- ink-surface half renders correct inverted variants automatically

- [ ] **Step 3: Commit**

```bash
git add app/design-preview/page.tsx
git commit -m "feat(frontend): /design-preview primitive gallery (dev-only)"
```

---

## Self-Review (done at write time)

- **Spec coverage:** all six handoff shared components (AppHeader, Chip, Button, Cell, EventModule, BottomTabBar) + all listed secondary components (ProgressBar, Field, StatusChip, Stepper) + skeleton/empty-state state patterns + tokens + fonts. Screens themselves are Phases 2–7.
- **Known deviations, all deliberate and flagged:** Golos/Manrope instead of non-Cyrillic Archivo/Space Grotesk (checkpoint 2); tab «Я» targets `/me/practices` until U6 exists; TEMP compat shims for legacy `rounded-*`/`shadow-*`/`glass` classes are explicitly marked for Phase 2 removal.
- **Type consistency:** `ChipVariant` exported from `Chip.tsx` and consumed by `StatusChip`; `statusChipVariant` return type shared via `ReturnType`; `cn` signature identical at old import path.
- **Placeholder scan:** no TBDs; every step has runnable code/commands.
