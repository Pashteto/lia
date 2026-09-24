# Сообща Rebrand + Intro Splash Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace every user-visible "Presence" with the Сообща brand (wordmark, titles, icons, emails) and add the once-per-session CSS intro splash that lands in the U1 feed header.

**Architecture:** One source of truth for brand strings (`frontend/lib/brand.ts`) feeds a single `<Wordmark>` component used by every header, the page-title helper, and the splash. Icons come from a committed generator script, so they can be rebuilt. The splash is an SSR overlay on `/` only. A pre-paint inline script decides "play / skip" (no flash). A client component measures the real header wordmark (FLIP), so the landing is pixel-identical whatever the header padding. Everything is CSS keyframes, with no animation library.

**Tech Stack:** Next.js 16 (app router) + React 19 + Tailwind v4, vitest (node env, pure-logic tests only — no DOM renderer in this repo), Go (backend mailer, GateGuard templates), Python + fontTools/Pillow (one-off icon generator, scratch venv).

**Spec:** `docs/Redesign/5/design_handoff_presence_swiss_grid/` — `README.md` (rebrand note at top, *Header* section, *Assets*), `INTRO-SPLASH.md`, `CLAUDE.md`, `tokens.ts`/`tokens.css` (brand block), visual refs `Soobscha Swiss Grid - Full System.dc.html`, `Presence Brand - Сообща.dc.html`, `intro/*.dc.html`.

## Global Constraints

- Product name **Сообща** (capitalised in prose; it does not decline: «в Сообща», «на Сообща»). Wordmark **`сообща`**, always lowercase. **Never render "Presence"/"PRESENCE" in UI, titles, or emails.**
- Wordmark: weight 900, tracking `-0.04em`, line-height 1, **15px desktop / 13px mobile** (README + `tokens.css`; `INTRO-SPLASH.md`'s "17px mobile" conflicts with them and loses).
- **Font substitution (standing decision, see `frontend/app/layout.tsx:14-16`):** Archivo → Golos Text (`font-ui`), Space Grotesk → Manrope (`font-alt`), JetBrains Mono unchanged. Archivo/Space Grotesk have no Cyrillic (verified on Google Fonts 2026-09-24: latin, latin-ext, vietnamese only), so the mock's `сообща` is a fallback-font rendering. Do not add Archivo.
- Admin wordmark: `сообща` + `ADMIN` (8px, tracking `0.2em`, colour `#8A857C` = `muted-2`, margin-left 6px), inverted surface.
- Monogram `сб`: weight 900, tracking `-0.08em`, white on `#111` square. Used for favicon and app icons.
- Domain in code: `soobscha.ru`, from `BRAND.domain` only. **This is not final.** `soobscha.ru` is registered to a third party (whois 2026-09-24: private person, reg.ru, since 2019). `soobshcha.ru`/`soobscha.com` are free. Keep the value in one constant and don't write the domain anywhere else. DNS/infra is out of scope here.
- Splash: CSS-only (`@keyframes` + `animation-delay`), 3.6 s, once per browser session (`sessionStorage`), skippable by click/tap anywhere, skipped entirely under `prefers-reduced-motion: reduce`. Wait for `document.fonts.ready` before playing. No images, no video, no library. Keep it small (< 5 KB of component code). Do not port `intro/animations-v3.jsx`.
- Zero border radius, zero shadows (README → *Things not to get wrong*).

## Review Focus

1. **Storage unavailable** (Safari private mode, blocked site data): `sessionStorage` throws. Expected: the splash must never cover the page permanently. Skip it, and never throw during render. → Task 5 tests `shouldPlayIntro` with a throwing storage.
2. **JS disabled / hydration fails / chunk load error**: the SSR overlay would cover the feed forever. Expected: a pure-CSS fail-safe hides the overlay at 4 s regardless. → Task 6 step "fail-safe" + manual check with JS disabled.
3. **Deep link to non-feed routes** (`/events/123`, `/login`): expected no splash; the splash only mounts on `/`. It should also not burn the "seen" flag, so the user still sees it on their first feed visit. → Task 6 manual check.
4. **Mobile viewport / header layout changes**: the landing target must be measured from the live header wordmark, not hard-coded `(48,30)` / `(18,58)`. Expected: the flying word ends exactly over the header word at 390×844 and 1440×900. → Task 5 tests `landingTransform`; Task 6 screenshot comparison.
5. **Leftover brand strings** (page `<title>`, `aria-label`, email subject/signature, organizer name in DB). Expected: nothing says Presence. → Task 3 guard test greps the source tree; Task 7 covers emails; Task 8 covers the DB row.

---

## File Structure

| File | Responsibility |
|---|---|
| `frontend/lib/brand.ts` (new) | `BRAND` constant + `pageTitle()` helper. Only place brand strings live. |
| `frontend/lib/__tests__/brand.test.ts` (new) | Brand constants, title format, no-"Presence" source guard. |
| `frontend/components/ui/Wordmark.tsx` (new) | The `сообща` mark (+ optional `ADMIN` suffix). |
| `frontend/components/ui/AppHeader.tsx` (modify) | Uses `<Wordmark>`; carries `data-wordmark` for the splash to measure. |
| `frontend/components/ui/BrandLogo.tsx` (delete) | Dead "Presence.tarski" logo, no importers. |
| `frontend/app/login/page.tsx`, `frontend/app/signup/page.tsx` (modify) | Panel wordmark → `<Wordmark>`. |
| `frontend/app/**/page.tsx`, `frontend/app/layout.tsx` (modify) | Titles via `pageTitle()`. |
| `frontend/app/globals.css` (modify) | `.wordmark` + splash keyframes. |
| `frontend/scripts/brand-icons.py` (new) | Generates `app/icon.svg`, `app/apple-icon.png`, `app/favicon.ico` from Golos Text 900. |
| `frontend/lib/intro-splash.ts` (new) | Pure splash logic: play decision, scatter offsets, FLIP maths. |
| `frontend/lib/__tests__/intro-splash.test.ts` (new) | Tests for the above. |
| `frontend/components/IntroSplash.tsx` (new) | Overlay markup + client driver. |
| `frontend/app/page.tsx` (modify) | Mounts `<IntroSplash>` on the feed only. |
| `backend/internal/notifications/mailer.go` (+test) | Invitation subject. |
| `gateguard/internal/pkg/notificator/templates/email_verification.go`, `.../html/email_verification.go` (+test) | Verification subject + signature. |

---

### Task 1: Brand module

**Files:**
- Create: `frontend/lib/brand.ts`
- Test: `frontend/lib/__tests__/brand.test.ts`

**Interfaces:**
- Produces: `BRAND: { name: "Сообща"; wordmark: "сообща"; domain: string; monogram: "сб" }`, `pageTitle(section?: string): string`.

- [ ] **Step 1: Write the failing test**

```ts
// frontend/lib/__tests__/brand.test.ts
import { describe, expect, it } from "vitest";
import { BRAND, pageTitle } from "../brand";

describe("brand", () => {
  it("carries the Сообща identity from the handoff tokens", () => {
    expect(BRAND.name).toBe("Сообща");
    expect(BRAND.wordmark).toBe("сообща");
    expect(BRAND.monogram).toBe("сб");
    expect(BRAND.domain).toBe("soobscha.ru");
  });
  it("titles a section page as «<section> — Сообща»", () => {
    expect(pageTitle("Вход")).toBe("Вход — Сообща");
  });
  it("titles the root as «Сообща — События»", () => {
    expect(pageTitle()).toBe("Сообща — События");
  });
});
```

- [ ] **Step 2: Run it and check that it fails**

Run: `cd frontend && npx vitest run lib/__tests__/brand.test.ts`
Expected: FAIL, `Cannot find module '../brand'`.

- [ ] **Step 3: Implement**

```ts
// frontend/lib/brand.ts
/** Brand identity (handoff tokens.ts → brand). The only place brand strings
 * live — headers, titles, icons and the intro splash all read from here.
 * `domain` is pending a registrar decision (soobscha.ru is third-party owned
 * as of 2026-09-24); change it here and nowhere else. */
export const BRAND = {
  name: "Сообща",
  wordmark: "сообща",
  domain: "soobscha.ru",
  monogram: "сб",
} as const;

/** Page <title>: «Вход — Сообща»; the root is «Сообща — События». */
export function pageTitle(section?: string): string {
  return section ? `${section} — ${BRAND.name}` : `${BRAND.name} — События`;
}
```

- [ ] **Step 4: Run it and check that it passes**

Run: `cd frontend && npx vitest run lib/__tests__/brand.test.ts` → PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/brand.ts frontend/lib/__tests__/brand.test.ts
git commit -m "feat(brand): add the Сообща brand constants"
```

---

### Task 2: Wordmark in every header

**Files:**
- Create: `frontend/components/ui/Wordmark.tsx`
- Modify: `frontend/components/ui/AppHeader.tsx` (lines 16, 45-56), `frontend/app/login/page.tsx:17`, `frontend/app/signup/page.tsx:13`, `frontend/app/globals.css` (append `.wordmark`)
- Delete: `frontend/components/ui/BrandLogo.tsx` (no importers; confirm with `grep -rn BrandLogo frontend/app frontend/components`)

**Interfaces:**
- Consumes: `BRAND` (Task 1).
- Produces: `<Wordmark admin?: boolean />`. The element carries `data-wordmark`, which Task 6 queries with `document.querySelector("header [data-wordmark]")`. Global class `.wordmark`, which Task 6's splash reuses.

No DOM test harness exists, so this task is verified visually (step 4).

- [ ] **Step 1: Add the CSS class** to the end of `frontend/app/globals.css`:

```css
/* Сообща wordmark (handoff README → Header; tokens.css brand block).
 * Archivo 900 in the spec → Golos Text 900 (Archivo has no Cyrillic). */
.wordmark {
  font-family: var(--font-ui);
  font-weight: 900;
  font-size: 15px;
  letter-spacing: -0.04em;
  line-height: 1;
  text-transform: lowercase;
}
@media (max-width: 639px) {
  .wordmark { font-size: 13px; }
}
.wordmark-suffix {
  margin-left: 6px;
  font-size: 8px;
  font-weight: 400;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--muted-2);
}
```

- [ ] **Step 2: Create the component**

```tsx
// frontend/components/ui/Wordmark.tsx
import { BRAND } from "@/lib/brand";
import { cn } from "@/lib/cn";

/** The сообща wordmark. `admin` appends the small tracked ADMIN suffix.
 * `data-wordmark` is the landing target the intro splash measures. */
export function Wordmark({ admin, className }: { admin?: boolean; className?: string }) {
  return (
    <span data-wordmark className={cn("wordmark", className)}>
      {BRAND.wordmark}
      {admin ? <span className="wordmark-suffix">ADMIN</span> : null}
    </span>
  );
}
```

- [ ] **Step 3: Wire it in**

In `AppHeader.tsx`, update the prop doc to `/** Admin variant: wordmark сообща + ADMIN suffix, paper bottom rule (inside data-surface="ink"). */`. Replace the `<Link>` className and children:

```tsx
      <Link href={admin ? "/admin" : "/"} aria-label={admin ? `${BRAND.name} — администрирование` : BRAND.name} className="swiss-focus shrink-0 pr-[10px]">
        <Wordmark admin={admin} />
      </Link>
```

(import `Wordmark` from `@/components/ui/Wordmark` and `BRAND` from `@/lib/brand`). The old mobile admin variant (just `ADMIN`) is gone because the suffix fits at 13px.

In `login/page.tsx` and `signup/page.tsx`, replace
`<span className="text-[13px] font-black tracking-[-0.01em]">PRESENCE</span>` with `<Wordmark />`.

Then `git rm frontend/components/ui/BrandLogo.tsx`.

- [ ] **Step 4: Verify**

Run: `cd frontend && npx tsc --noEmit && npm run lint && npx vitest run`. Expected: all pass.
Run `npm run dev` and screenshot these pages with the claude-in-chrome tools at 1440×900 and 390×844: `/`, `/login`, `/organizer`, `/admin`. Compare them with `Soobscha Swiss Grid - Full System.dc.html` (U1 header, admin header). Expected: lowercase `сообща`, same baseline as the nav, `ADMIN` suffix grey on the admin page, no "PRESENCE" anywhere.

- [ ] **Step 5: Commit**

```bash
git add -A frontend/components/ui frontend/app/login/page.tsx frontend/app/signup/page.tsx frontend/app/globals.css
git commit -m "feat(brand): сообща wordmark in every header"
```

---

### Task 3: Titles, metadata, and the no-Presence guard

**Files:**
- Modify: `frontend/app/layout.tsx:39`, `frontend/app/{login,signup,search,map,me,me/calendar,me/organizer}/page.tsx` (the `metadata` line), `frontend/app/globals.css:4`, `frontend/components/ui/SquareCheck.tsx:8`, `frontend/lib/types.ts:1,127` (comments)
- Test: `frontend/lib/__tests__/brand.test.ts` (extend)

**Interfaces:**
- Consumes: `pageTitle`, `BRAND` (Task 1).

- [ ] **Step 1: Add the guard test** (append to `brand.test.ts`):

```ts
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    if (name === "node_modules" || name === "__tests__" || name.startsWith(".")) return [];
    return statSync(p).isDirectory() ? sourceFiles(p) : /\.(tsx?|css)$/.test(name) ? [p] : [];
  });
}

describe("rebrand guard", () => {
  it("no source file under app/, components/, lib/ says Presence", () => {
    const root = join(__dirname, "..", "..");
    const offenders = ["app", "components", "lib"]
      .flatMap((d) => sourceFiles(join(root, d)))
      .filter((f) => /presence/i.test(readFileSync(f, "utf8")));
    expect(offenders).toEqual([]);
  });
});
```

`next.config.ts` is not scanned on purpose: its `presencehq.ru` / `presence.tarski.ru` image hosts are live infrastructure and must stay until the domain moves.

- [ ] **Step 2: Run it and check that it fails**

Run: `cd frontend && npx vitest run lib/__tests__/brand.test.ts`
Expected: FAIL. The offenders list names layout.tsx, the 7 pages, globals.css, SquareCheck.tsx, types.ts.

- [ ] **Step 3: Implement**
  - `layout.tsx`: `title: pageTitle(),` and add `applicationName: BRAND.name,` next to it.
  - Each page: `export const metadata = { title: pageTitle("Вход") };` (sections: Регистрация, Подбор, Карта, Профиль, Календарь, Профиль организатора).
  - Comments: `globals.css:4` → `* Сообща — Swiss Grid design tokens.`. `SquareCheck.tsx:8` → `any Сообща surface`. `types.ts:1` → `// Domain types for the Сообща frontend.`. `types.ts:127` → `(Lia API Event model)`.

- [ ] **Step 4: Run all tests**

Run: `cd frontend && npx vitest run && npx tsc --noEmit` → PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/app frontend/components frontend/lib
git commit -m "feat(brand): Сообща page titles; guard against Presence strings"
```

---

### Task 4: Monogram icons

**Files:**
- Create: `frontend/scripts/brand-icons.py`, `frontend/app/icon.svg`, `frontend/app/apple-icon.png`
- Replace: `frontend/app/favicon.ico`

**Interfaces:** none. Next's file conventions (`app/icon.svg`, `app/apple-icon.png`, `app/favicon.ico`) emit the `<link>` tags automatically.

- [ ] **Step 1: Write the generator**

```python
# frontend/scripts/brand-icons.py
"""Generate the сб monogram icons (handoff README → Assets: Archivo 900,
tracking -.08em, white on #111 — Golos Text 900 substitutes for Archivo).

Run once from frontend/:  python3 scripts/brand-icons.py
Needs: pip install fonttools pillow  (use a throwaway venv)."""
import io, urllib.request
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from PIL import Image, ImageDraw, ImageFont

SRC = "https://github.com/google/fonts/raw/main/ofl/golostext/GolosText%5Bwght%5D.ttf"
TEXT, INK, PAPER, TRACK = "сб", "#111111", "#FFFFFF", -0.08

raw = urllib.request.urlopen(SRC).read()
font = instantiateVariableFont(TTFont(io.BytesIO(raw)), {"wght": 900})
buf = io.BytesIO(); font.save(buf); ttf = buf.getvalue()
upm = font["head"].unitsPerEm
cmap, gs, hmtx = font.getBestCmap(), font.getGlyphSet(), font["hmtx"]

# --- SVG: glyph outlines as paths, no font dependency at runtime ---
size = 64; em = 40  # glyph em in px inside a 64px square
scale = em / upm
names = [cmap[ord(c)] for c in TEXT]
advance = sum(hmtx[n][0] for n in names) + TRACK * upm * (len(names) - 1)
x0 = (size - advance * scale) / 2
cap = font["OS/2"].sxHeight or upm * 0.5
baseline = size / 2 + cap * scale / 2
paths, x = [], 0.0
for n in names:
    pen = SVGPathPen(gs)
    gs[n].draw(TransformPen(pen, (scale, 0, 0, -scale, x0 + x * scale, baseline)))
    paths.append(pen.getCommands())
    x += hmtx[n][0] + TRACK * upm
svg = (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {size} {size}">'
       f'<rect width="{size}" height="{size}" fill="{INK}"/>'
       f'<path fill="{PAPER}" d="{" ".join(paths)}"/></svg>\n')
open("app/icon.svg", "w").write(svg)

# --- PNG/ICO via Pillow from the same static instance ---
def raster(px: int) -> Image.Image:
    img = Image.new("RGB", (px, px), INK)
    d = ImageDraw.Draw(img)
    f = ImageFont.truetype(io.BytesIO(ttf), int(px * em / size))
    widths = [d.textlength(c, font=f) for c in TEXT]
    track = TRACK * f.size
    total = sum(widths) + track * (len(TEXT) - 1)
    x = (px - total) / 2
    for c, w in zip(TEXT, widths):
        d.text((x, px / 2), c, font=f, fill=PAPER, anchor="lm")
        x += w + track
    return img

raster(180).save("app/apple-icon.png")
raster(48).save("app/favicon.ico", sizes=[(16, 16), (32, 32), (48, 48)])
print("wrote app/icon.svg app/apple-icon.png app/favicon.ico")
```

- [ ] **Step 2: Run it**

```bash
S=$(mktemp -d); python3 -m venv $S/v && $S/v/bin/pip -q install fonttools pillow
cd frontend && $S/v/bin/python scripts/brand-icons.py
```
Expected: `wrote app/icon.svg app/apple-icon.png app/favicon.ico`.

- [ ] **Step 3: Verify**

Open `frontend/app/apple-icon.png` and `frontend/app/icon.svg` with the Read tool. Expected: white lowercase `сб`, tight tracking, optically centred on a black square, square corners. Then `npm run dev` and `curl -sI localhost:3000/icon.svg localhost:3000/apple-icon.png localhost:3000/favicon.ico`. Expected: three `200` responses. `curl -s localhost:3000/ | grep -o '<link rel="icon[^>]*>'` should list the svg icon.

- [ ] **Step 4: Commit**

```bash
git add frontend/scripts/brand-icons.py frontend/app/icon.svg frontend/app/apple-icon.png frontend/app/favicon.ico
git commit -m "feat(brand): сб monogram favicon and app icons"
```

---

### Task 5: Intro splash — pure logic

**Files:**
- Create: `frontend/lib/intro-splash.ts`
- Test: `frontend/lib/__tests__/intro-splash.test.ts`

**Interfaces:**
- Produces:
  - `INTRO_SEEN_KEY = "soobscha:intro-seen"`
  - `readSeen(storage: Pick<Storage,"getItem"> | null): boolean`. Returns true when the flag is set *or when storage is unusable* (no storage means skip).
  - `markSeen(storage: Pick<Storage,"setItem"> | null): void`. Never throws.
  - `shouldPlayIntro(o: { storage: Pick<Storage,"getItem"> | null; reducedMotion: boolean }): boolean`
  - `type Offset = { x: number; y: number; r: number }`, `SCATTER_DESKTOP: Offset[]`, `SCATTER_MOBILE: Offset[]` (6 each, order с о о б щ а)
  - `type Box = { left: number; top: number; width: number; height: number }`
  - `landingTransform(from: Box, to: Box): { dx: number; dy: number; scale: number }`. Translation plus uniform scale (origin top-left) that maps `from` onto `to`. Scale = `to.height / from.height`.
  - `INTRO_INLINE_SCRIPT: string`. Pre-paint script body. It sets `document.documentElement.dataset.intro` to `"play"` or `"skip"`.

- [ ] **Step 1: Write the failing tests**

```ts
// frontend/lib/__tests__/intro-splash.test.ts
import { describe, expect, it } from "vitest";
import {
  INTRO_SEEN_KEY, SCATTER_DESKTOP, SCATTER_MOBILE,
  landingTransform, markSeen, readSeen, shouldPlayIntro,
} from "../intro-splash";

const mem = (init: Record<string, string> = {}) => {
  const m = new Map(Object.entries(init));
  return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v), m };
};
const throwing = {
  getItem: () => { throw new DOMException("denied", "SecurityError"); },
  setItem: () => { throw new DOMException("denied", "SecurityError"); },
};

describe("shouldPlayIntro", () => {
  it("plays on a fresh session", () => {
    expect(shouldPlayIntro({ storage: mem(), reducedMotion: false })).toBe(true);
  });
  it("skips once seen this session", () => {
    expect(shouldPlayIntro({ storage: mem({ [INTRO_SEEN_KEY]: "1" }), reducedMotion: false })).toBe(false);
  });
  it("skips under prefers-reduced-motion", () => {
    expect(shouldPlayIntro({ storage: mem(), reducedMotion: true })).toBe(false);
  });
  it("skips (never blocks the page) when storage throws", () => {
    expect(shouldPlayIntro({ storage: throwing, reducedMotion: false })).toBe(false);
  });
  it("skips when storage is missing", () => {
    expect(shouldPlayIntro({ storage: null, reducedMotion: false })).toBe(false);
  });
});

describe("seen flag", () => {
  it("round-trips", () => {
    const s = mem();
    expect(readSeen(s)).toBe(false);
    markSeen(s);
    expect(readSeen(s)).toBe(true);
  });
  it("markSeen swallows storage errors", () => {
    expect(() => markSeen(throwing)).not.toThrow();
    expect(() => markSeen(null)).not.toThrow();
  });
});

describe("scatter offsets (INTRO-SPLASH.md)", () => {
  it("has one offset per letter of сообща", () => {
    expect(SCATTER_DESKTOP).toHaveLength(6);
    expect(SCATTER_MOBILE).toHaveLength(6);
  });
  it("matches the spec's first and last letters", () => {
    expect(SCATTER_DESKTOP[0]).toEqual({ x: -560, y: -300, r: -18 });
    expect(SCATTER_DESKTOP[5]).toEqual({ x: -140, y: 380, r: -24 });
    expect(SCATTER_MOBILE[3]).toEqual({ x: 150, y: 260, r: -14 });
  });
});

describe("landingTransform", () => {
  it("maps the 220px centred word onto a 15px header word", () => {
    const from = { left: 400, top: 340, width: 640, height: 220 };
    const to = { left: 20, top: 13, width: 44, height: 15 };
    const t = landingTransform(from, to);
    expect(t.scale).toBeCloseTo(15 / 220, 6);
    expect(from.left + t.dx).toBe(to.left);
    expect(from.top + t.dy).toBe(to.top);
  });
  it("is the identity when boxes coincide", () => {
    const b = { left: 1, top: 2, width: 3, height: 4 };
    expect(landingTransform(b, b)).toEqual({ dx: 0, dy: 0, scale: 1 });
  });
});
```

- [ ] **Step 2: Run them and check that they fail**

Run: `cd frontend && npx vitest run lib/__tests__/intro-splash.test.ts` → FAIL (module missing).

- [ ] **Step 3: Implement**

```ts
// frontend/lib/intro-splash.ts
/** Pure logic for the first-visit splash (handoff INTRO-SPLASH.md).
 * Anything that can fail degrades to "skip" — the splash must never trap
 * the page behind an overlay. */

export const INTRO_SEEN_KEY = "soobscha:intro-seen";

type Get = Pick<Storage, "getItem">;
type Set = Pick<Storage, "setItem">;

export function readSeen(storage: Get | null): boolean {
  if (!storage) return true;
  try {
    return storage.getItem(INTRO_SEEN_KEY) === "1";
  } catch {
    return true;
  }
}

export function markSeen(storage: Set | null): void {
  try {
    storage?.setItem(INTRO_SEEN_KEY, "1");
  } catch {
    /* private mode / blocked storage: nothing to remember, nothing to break */
  }
}

export function shouldPlayIntro(o: { storage: Get | null; reducedMotion: boolean }): boolean {
  return !o.reducedMotion && !readSeen(o.storage);
}

export type Offset = { x: number; y: number; r: number };
const o = (x: number, y: number, r: number): Offset => ({ x, y, r });

/** Start offsets (px, deg), letters с о о б щ а. */
export const SCATTER_DESKTOP: Offset[] = [o(-560, -300, -18), o(-260, 330, 12), o(80, -380, 22), o(380, 300, -14), o(620, -200, 9), o(-140, 380, -24)];
export const SCATTER_MOBILE: Offset[] = [o(-150, -300, -18), o(-110, 280, 12), o(40, -360, 22), o(150, 260, -14), o(160, -180, 9), o(-40, 340, -24)];

export type Box = { left: number; top: number; width: number; height: number };

/** FLIP: translate + uniform scale (transform-origin top left) taking the
 * big centred word onto the header wordmark. Letter-spacing stays -0.04em
 * throughout (em-relative), so a uniform scale is exact. */
export function landingTransform(from: Box, to: Box): { dx: number; dy: number; scale: number } {
  return { dx: to.left - from.left, dy: to.top - from.top, scale: to.height / from.height };
}

/** Runs before first paint (inline <script> in the overlay) so returning
 * visitors never see a flash of the overlay. Mirrors shouldPlayIntro. */
export const INTRO_INLINE_SCRIPT = `(function(){var d=document.documentElement,p="skip";try{if(!matchMedia("(prefers-reduced-motion: reduce)").matches&&sessionStorage.getItem("${INTRO_SEEN_KEY}")!=="1")p="play"}catch(e){}d.dataset.intro=p})();`;
```

- [ ] **Step 4: Run them and check that they pass**

Run: `cd frontend && npx vitest run lib/__tests__/intro-splash.test.ts` → PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/intro-splash.ts frontend/lib/__tests__/intro-splash.test.ts
git commit -m "feat(intro): splash play decision, scatter offsets, landing maths"
```

---

### Task 6: `<IntroSplash>` on the feed

**Files:**
- Create: `frontend/components/IntroSplash.tsx`
- Modify: `frontend/app/globals.css` (append splash CSS), `frontend/app/page.tsx` (mount)

**Interfaces:**
- Consumes: `BRAND` (Task 1), `.wordmark` + `[data-wordmark]` (Task 2), everything from Task 5.
- Produces: `<IntroSplash />`, a client component with no props.

**How it works:**
1. SSR renders `<div class="intro" aria-hidden>` with the inline script as its first child. The script sets `html[data-intro]` before paint. CSS shows the overlay only under `html[data-intro="play"]`.
2. On mount, the client waits for `document.fonts.ready`. It measures the big word (`.intro-word`) and the header `[data-wordmark]`, writes `--dx/--dy/--s` on the overlay, then sets `data-state="run"`. All keyframes are gated on `[data-state="run"]`, so nothing moves before fonts are ready and the target is measured.
3. `animationend` of the overlay's own fade (the last animation, ending at 3.6 s) → `markSeen`, and `html[data-intro]="done"` (overlay `display:none`). A click/tap anywhere does the same immediately.
4. Fail-safe: `.intro` has a 0-duration animation at 4.5 s to `visibility:hidden; pointer-events:none`. It runs even with no JS, so a failed hydration can't trap the page.

- [ ] **Step 1: Append the CSS** to `frontend/app/globals.css`:

```css
/* ── Intro splash (handoff INTRO-SPLASH.md) ─────────────────────────── */
.intro { display: none; }
html[data-intro="play"] .intro {
  position: fixed; inset: 0; z-index: 100; display: block;
  background: var(--paper); color: var(--ink); cursor: pointer;
  animation: intro-failsafe 0s 4.5s forwards;
}
@keyframes intro-failsafe { to { visibility: hidden; pointer-events: none; } }
html[data-intro="play"] { overflow: hidden; }

.intro-stage { position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%); }
.intro-word { display: block; font-size: 220px; white-space: nowrap; }
@media (max-width: 639px) { .intro-word { font-size: 84px; } }
.intro-letter { display: inline-block; opacity: 0;
  transform: translate(var(--x), var(--y)) rotate(var(--r)); }
.intro-rule { height: 2px; background: var(--ink); transform: scaleX(0); transform-origin: left; margin-top: 12px; }
.intro-sign { display: flex; justify-content: space-between; margin-top: 10px; opacity: 0;
  font-family: var(--font-jbmono), monospace; font-size: 11px; letter-spacing: 0.14em; color: var(--muted-2); }
.intro-caption { position: absolute; left: 0; right: 0; bottom: 40px; text-align: center; opacity: 0;
  font-family: var(--font-alt); font-size: 10px; letter-spacing: 0.2em; text-transform: uppercase; color: var(--muted-2); }

/* Everything below is gated on data-state="run" (fonts ready + target measured). */
.intro[data-state="run"] .intro-letter {
  animation:
    intro-in 0.3s cubic-bezier(.33,1,.68,1) calc(var(--i) * 0.05s) forwards,
    intro-gather 0.75s cubic-bezier(.76,0,.24,1) calc(0.5s + var(--i) * 0.025s) forwards;
}
@keyframes intro-in { to { opacity: 1; } }
@keyframes intro-gather { from { opacity: 1; } to { opacity: 1; transform: none; } }

.intro[data-state="run"] .intro-caption { animation: intro-caption 1.4s linear 0.7s forwards; }
@keyframes intro-caption { 0% { opacity: 0 } 20% { opacity: 1 } 80% { opacity: 1 } 100% { opacity: 0 } }

.intro[data-state="run"] .intro-rule {
  animation: intro-rule 0.4s cubic-bezier(.33,1,.68,1) 1.4s forwards, intro-fade 0.25s linear 2.1s forwards;
}
@keyframes intro-rule { to { transform: scaleX(1); } }
.intro[data-state="run"] .intro-sign {
  animation: intro-show 0.3s linear 1.6s forwards, intro-fade 0.25s linear 2.1s forwards;
}
@keyframes intro-show { to { opacity: 1; } }
@keyframes intro-fade { from { opacity: 1; } to { opacity: 0; } }

/* Open: the word flies onto the measured header wordmark. */
.intro[data-state="run"] .intro-flight {
  transform-origin: 0 0;
  animation: intro-flight 0.7s cubic-bezier(.65,0,.35,1) 2.1s forwards;
}
@keyframes intro-flight { to { transform: translate(var(--dx), var(--dy)) scale(var(--s)); } }

/* Paper lifts off the already-rendered feed; the real header word sits exactly under the landed word. */
.intro[data-state="run"] { animation: intro-lift 0.45s cubic-bezier(.33,1,.68,1) 2.8s forwards, intro-failsafe 0s 4.5s forwards; }
@keyframes intro-lift { to { background-color: transparent; } }
html[data-intro="play"] .intro-content { opacity: 0; transform: translateY(18px); }
html[data-intro="play"] .intro[data-state="run"] ~ .intro-content,
html[data-intro="done"] .intro-content { animation: intro-content 0.45s cubic-bezier(.33,1,.68,1) 2.8s forwards; }
html[data-intro="done"] .intro-content { animation-delay: 0s; }
@keyframes intro-content { to { opacity: 1; transform: none; } }
html[data-intro="done"] .intro { display: none; }
```

The overlay fades to transparent at 2.8–3.25 s. The landed word stays visible on top of the identical header word until `done` (3.6 s), then the overlay is removed. The seam is invisible because both words are the same glyphs at the same box.

- [ ] **Step 2: Write the component**

```tsx
// frontend/components/IntroSplash.tsx
"use client";

import { useEffect, useRef, useState } from "react";
import { BRAND } from "@/lib/brand";
import {
  INTRO_INLINE_SCRIPT, SCATTER_DESKTOP, SCATTER_MOBILE,
  landingTransform, markSeen,
} from "@/lib/intro-splash";

function storage(): Storage | null {
  try { return window.sessionStorage; } catch { return null; }
}

/** First-visit splash (handoff INTRO-SPLASH.md): сообща gathers from scattered
 * letters and flies into the feed header. CSS keyframes only; this component
 * just measures the landing target and ends the show. */
export function IntroSplash() {
  const root = useRef<HTMLDivElement>(null);
  const word = useRef<HTMLSpanElement>(null);
  const [state, setState] = useState<"idle" | "run">("idle");

  useEffect(() => {
    if (document.documentElement.dataset.intro !== "play") return;
    let cancelled = false;
    const finish = () => {
      markSeen(storage());
      document.documentElement.dataset.intro = "done";
    };
    document.fonts.ready.then(() => {
      if (cancelled || !root.current || !word.current) return;
      const target = document.querySelector("header [data-wordmark]");
      if (!target) return finish();
      const t = landingTransform(word.current.getBoundingClientRect(), target.getBoundingClientRect());
      root.current.style.setProperty("--dx", `${t.dx}px`);
      root.current.style.setProperty("--dy", `${t.dy}px`);
      root.current.style.setProperty("--s", String(t.scale));
      setState("run");
    });
    const el = root.current;
    const onEnd = (e: AnimationEvent) => { if (e.target === el && e.animationName === "intro-lift") setTimeout(finish, 350); };
    el?.addEventListener("animationend", onEnd);
    el?.addEventListener("click", finish);
    return () => { cancelled = true; el?.removeEventListener("animationend", onEnd); el?.removeEventListener("click", finish); };
  }, []);

  const mobile = typeof window !== "undefined" && window.matchMedia("(max-width: 639px)").matches;
  const scatter = mobile ? SCATTER_MOBILE : SCATTER_DESKTOP;

  return (
    <div ref={root} className="intro" data-state={state} aria-hidden="true">
      <script dangerouslySetInnerHTML={{ __html: INTRO_INLINE_SCRIPT }} />
      <div className="intro-stage">
        <div className="intro-flight">
          <span ref={word} className="wordmark intro-word">
            {[...BRAND.wordmark].map((ch, i) => (
              <span key={i} className="intro-letter" style={{ "--i": i, "--x": `${scatter[i].x}px`, "--y": `${scatter[i].y}px`, "--r": `${scatter[i].r}deg` } as React.CSSProperties}>
                {ch}
              </span>
            ))}
          </span>
        </div>
        <div className="intro-rule" />
        <div className="intro-sign"><span>{BRAND.domain.toUpperCase()}</span><span>МОСКВА · 2026</span></div>
      </div>
      <p className="intro-caption">Вместе · общими усилиями</p>
    </div>
  );
}
```

Note on `mobile`: SSR renders the desktop offsets. Because letters are invisible until `run`, a client re-render with mobile offsets before `run` is harmless. If React warns about a hydration mismatch on the `style` attribute, move the offsets to CSS instead. Put desktop offsets in `.intro-letter:nth-child(n)` rules and mobile ones in a `max-width:639px` block, and drop the `mobile` branch. That is the fallback if the warning appears.

- [ ] **Step 3: Mount on the feed only.** In `frontend/app/page.tsx`, import `IntroSplash` and render `<IntroSplash />` as the first child of the fragment. Wrap `<DiscoveryFeed …/>` in `<div className="intro-content">…</div>` so the content rise from the spec applies. Put it after the `AppHeader`, which must stay visible under the landing word. `IntroSplash` must come before `.intro-content` in the same parent (the `~` selector needs a sibling).

- [ ] **Step 4: Verify — manual, both viewports**

Run: `cd frontend && npx tsc --noEmit && npm run lint && npx vitest run` → PASS.
Then `npm run dev`, and with claude-in-chrome record a GIF (`intro_splash_desktop.gif`, `intro_splash_mobile.gif`):
1. Fresh tab at 1440×900 on `/`. Expected: letters scatter in, gather, rule + `SOOBSCHA.RU / МОСКВА · 2026`, the word flies into the header, the feed rises. Take a screenshot at ~3.7 s: the header must look identical to a plain `/` reload (compare the two crops).
2. Reload in the same tab: no splash, no flash.
3. New session at 390×844: same, landing on the 13px mobile wordmark.
4. Click during the gather: splash ends at once, feed is usable.
5. DevTools → Rendering → emulate `prefers-reduced-motion: reduce`, new session: no splash.
6. New session straight to `/events/<any id>` or `/login`: no splash. Then navigate to `/`: splash plays (the flag was not burned).
7. JS disabled (DevTools → Settings → Disable JavaScript), new session: no overlay (the inline script doesn't run, so `data-intro` is never `play`).
8. Throttle to "Slow 3G" and block `*.js` in the network panel: the overlay may show but must disappear by 4.5 s (fail-safe).

- [ ] **Step 5: Commit**

```bash
git add frontend/components/IntroSplash.tsx frontend/app/globals.css frontend/app/page.tsx
git commit -m "feat(intro): first-visit сообща splash landing in the feed header"
```

---

### Task 7: Emails say Сообща

**Files:**
- Modify: `backend/internal/notifications/mailer.go:65`, `backend/internal/notifications/mailer_test.go`
- Modify: `gateguard/internal/pkg/notificator/templates/email_verification.go:32`, `gateguard/internal/pkg/notificator/templates/html/email_verification.go:22`, `gateguard/internal/pkg/notificator/templates/email_verification_test.go`

- [ ] **Step 1: Write the failing tests**

In `backend/internal/notifications/mailer_test.go`, add:

```go
func TestRenderInvitationEmail_Brand(t *testing.T) {
	subject, _ := notifications.RenderInvitationEmail("Лекция", "https://example.test/invite/x")
	if subject != "Subject: Сообща: приглашение на событие" {
		t.Fatalf("unexpected subject %q", subject)
	}
}
```

In `gateguard/internal/pkg/notificator/templates/email_verification_test.go` (package `templates_test`; `os`, `strings`, `clog`, `templates` already imported), add:

```go
func Test_EmailVerification_Brand(t *testing.T) {
	log := clog.NewCustomLogger(os.Stdout, clog.LevelDebug, false)
	tmpl := templates.NewEmailVerification(log, "042173")
	if got := tmpl.Subject(); got != "Subject: Сообща: код подтверждения почты" {
		t.Fatalf("unexpected subject %q", got)
	}
	body, err := tmpl.GetTemplateAsString(context.Background())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(body, "Presence") || !strings.Contains(body, "Команда Сообща") {
		t.Fatalf("signature must read «Команда Сообща»; got: %s", body)
	}
}
```

- [ ] **Step 2: Run them and check that they fail**

Run: `cd backend && go test ./internal/notifications/...` and `cd gateguard && go test ./internal/pkg/notificator/templates/...` → FAIL on the new tests.

- [ ] **Step 3: Implement.** `Presence:` → `Сообща:` in both subjects. `Команда Presence` → `Команда Сообща`.

- [ ] **Step 4: Run them and check that they pass.** Same commands → PASS. Also run `grep -rn "Presence" backend/internal gateguard/internal --include='*.go' | grep -v _test`. Expected: no user-facing hits.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/notifications gateguard/internal/pkg/notificator/templates
git commit -m "feat(brand): emails sign as Сообща"
```

---

### Task 8: Deploy + data rename (ops, after review)

Follow the standing deploy runbook pattern (`docs/superpowers/runbooks/2026-09-15-selectel-migration.md` for host facts; memory: tag `:latest` before `up -d --force-recreate`, take rollback tags from the running container, prune images afterwards).

- [ ] **Step 1:** Build and ship frontend, backend (`-tags nodynamic`, check with `file`) and gateguard images. Tag rollbacks `rollback-soobscha-YYYYMMDD` first.
- [ ] **Step 2: Rename the house organizer** on prod (DB `lia_prod`):

```sql
BEGIN;
SELECT id, name FROM organizers WHERE name ILIKE '%presence%';
UPDATE organizers SET name = 'Редакция Сообща' WHERE name = 'Редакция PRESENCE';
-- expect UPDATE 1, then COMMIT
COMMIT;
```

- [ ] **Step 3: Check the mail sender display name.** Look for `NOTIFICATOR_*` in `/opt/lia/.env.prod` (GateGuard) and the backend mailer `from` env. If the display name says Presence, change it to `Сообща <info@tarski.ru>` and recreate the affected service.
- [ ] **Step 4: Verify live**: `curl -s https://presencehq.ru/ | grep -o '<title>[^<]*'` → `Сообща — События`. Check that `/icon.svg` returns 200. Run the splash checks 1–3 from Task 6 on the live host, from a phone as well (memory: mobile TLS can only be tested from a phone). Send one invitation and check the subject.
- [ ] **Step 5:** Write the deploy runbook `docs/superpowers/runbooks/YYYY-MM-DD-soobscha-rebrand-deploy.md`, then prune images on the box.

## Out of scope (separate decisions, tracked here so they don't get lost)

- **Domain.** `soobscha.ru` is third-party owned. Choose: buy it, or switch `BRAND.domain` to the free `soobshcha.ru`. Then do DNS, TLS, the nginx host and 301s from `presencehq.ru` / `presence.tarski.ru` for 6 months (brand sheet), plus `next.config.ts` image hosts and `NEXT_PUBLIC_API_URL`.
- Trademark filing (naming-clearance README → classes 9, 35, 41, 42).
- `@soobshcha` Telegram handle, and the mail sender domain (`info@tarski.ru`).
- **Site footer «© 2026 Сообща · soobscha.ru».** It appears only in the brand sheet; README has no site footer. Add it when the domain is settled.
- **Organizer/admin overline variant** (brand sheet: section name *above* the word). This conflicts with README's `сообща ADMIN` suffix, and README wins; ask the designer if the overline is wanted.
- **`Soobscha Swiss Grid - Full System.dc.html`** was cut from the pre-cover mock, so it lacks the U1 feed covers. Our covers spec (`docs/superpowers/specs/2026-07-30-u1-feed-covers-design.md`) still rules.
