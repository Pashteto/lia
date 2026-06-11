# Apple Design System Rework — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rework the `design/` folder from the bespoke "Curatorial Minimalist" system to an Apple HIG-based design system: a new `DESIGN.md` (tokens, type, components, light+dark) plus 4 standalone reference screens, with READMEs updated and `stitch_/` kept as historical archive.

**Architecture:** Static design deliverables only — no application code. The 4 reference screens already exist and are user-approved as visual-companion mockups (content fragments) under `.superpowers/brainstorm/83441-1781186107/content/`. We promote each into a self-contained HTML document under `design/screens/`, author `design/DESIGN.md` from the approved spec, add a small gallery index, update READMEs, then commit and push.

**Tech Stack:** Plain HTML + inline CSS (system font stack, Apple semantic colors, Liquid Glass via `backdrop-filter`). No build step, no JS framework. Source of truth = `docs/superpowers/specs/2026-06-11-apple-design-system-rework-design.md`.

**Verification note:** There are no automated tests for static design assets. "Verify" steps mean: file exists, HTML is well-formed (opens in a browser with no console errors), and both light + dark blocks render. Use `open <file>` (macOS) to eyeball when a step calls for it.

---

## File Structure

- `design/DESIGN.md` — **new** active design system (Apple HIG). Replaces curatorial-minimalist as the canonical direction.
- `design/screens/discovery.html` — reference screen 1 (full HTML doc).
- `design/screens/create-event.html` — reference screen 2.
- `design/screens/event-detail.html` — reference screen 3.
- `design/screens/ai-search.html` — reference screen 4.
- `design/screens/index.html` — small gallery linking the 4 screens.
- `design/README.md` — **modify**: point to Apple system + screens; mark `stitch_/` historical.
- `README.md` (root) — **modify**: design row points to Apple system.
- `frontend/README.md` — **modify**: design reference points to Apple system + screens.

Each reference screen is a focused, self-contained file (one screen, its own scoped CSS) — easy to open and reason about independently.

---

## Task 1: Promote reference screen 1 (Discovery) to a standalone HTML doc

**Files:**
- Source (approved mockup, content fragment): `.superpowers/brainstorm/83441-1781186107/content/screen1-discovery.html`
- Create: `design/screens/discovery.html`

- [ ] **Step 1: Create the standalone file**

Copy the **entire** `<style>...</style>` and the `<div class="lia">...</div>` markup from the source fragment (the `<h2>`, `.subtitle`, and the final `.options` chooser block at the bottom are visual-companion scaffolding — **omit those three**). Wrap the kept content in a full HTML document:

```html
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Lia — Discovery (Apple design system)</title>
  <style>
    body{ margin:0; padding:32px; background:#e9e9ee; font-family:-apple-system,system-ui,sans-serif; }
    h1.page{ font:600 15px/1 -apple-system,system-ui,sans-serif; color:#3c3c43; opacity:.6; text-transform:uppercase; letter-spacing:.04em; margin:0 0 20px; }
  </style>
  <!-- paste the .lia scoped <style> block from the source fragment here -->
</head>
<body>
  <h1 class="page">Lia · Главная / Discovery</h1>
  <!-- paste the <div class="lia">...</div> markup from the source fragment here -->
</body>
</html>
```

- [ ] **Step 2: Verify it renders standalone**

Run: `open design/screens/discovery.html`
Expected: page opens; desktop (light) + mobile (dark) frames render with covers, capsule filters, glass nav/tab bar. No missing styles (the scoped `.lia` CSS must be present in `<head>`).

- [ ] **Step 3: Commit**

```bash
git add design/screens/discovery.html
git commit -m "design: add Apple-system discovery reference screen"
```

---

## Task 2: Promote reference screen 2 (Create event) to a standalone HTML doc

**Files:**
- Source: `.superpowers/brainstorm/83441-1781186107/content/screen2-create.html`
- Create: `design/screens/create-event.html`

- [ ] **Step 1: Create the standalone file**

Same transformation as Task 1: omit the `<h2>`, `.subtitle`, and trailing `.options` block; keep the `<style>` (the `.lia2` scoped block) and the `<div class="lia2">...</div>` markup. Use the same full-HTML wrapper as Task 1 but with `<title>Lia — Create event (Apple design system)</title>` and `<h1 class="page">Lia · Создание события</h1>`.

- [ ] **Step 2: Verify it renders standalone**

Run: `open design/screens/create-event.html`
Expected: grouped inset form renders (sections, segmented controls, switch, stepper, glass nav) in light (desktop) + dark (mobile).

- [ ] **Step 3: Commit**

```bash
git add design/screens/create-event.html
git commit -m "design: add Apple-system create-event reference screen"
```

---

## Task 3: Promote reference screen 3 (Event detail) to a standalone HTML doc

**Files:**
- Source: `.superpowers/brainstorm/83441-1781186107/content/screen3-detail.html`
- Create: `design/screens/event-detail.html`

- [ ] **Step 1: Create the standalone file**

Same transformation: keep the `.lia3` scoped `<style>` and `<div class="lia3">...</div>`; omit `<h2>`, `.subtitle`, trailing `.options`. Wrapper with `<title>Lia — Event detail (Apple design system)</title>` and `<h1 class="page">Lia · Детали события + регистрация</h1>`.

- [ ] **Step 2: Verify it renders standalone**

Run: `open design/screens/event-detail.html`
Expected: cover → large title → facts grid → sections → sticky glass bottom bar with "Записаться", light (desktop) + dark (mobile).

- [ ] **Step 3: Commit**

```bash
git add design/screens/event-detail.html
git commit -m "design: add Apple-system event-detail reference screen"
```

---

## Task 4: Promote reference screen 4 (AI search) to a standalone HTML doc

**Files:**
- Source: `.superpowers/brainstorm/83441-1781186107/content/screen4-ai.html`
- Create: `design/screens/ai-search.html`

- [ ] **Step 1: Create the standalone file**

Same transformation: keep the `.lia4` scoped `<style>` and `<div class="lia4">...</div>`; omit `<h2>`, `.subtitle`, trailing `.options`. Wrapper with `<title>Lia — AI search (Apple design system)</title>` and `<h1 class="page">Lia · AI-поиск / ассистент</h1>`.

- [ ] **Step 2: Verify it renders standalone**

Run: `open design/screens/ai-search.html`
Expected: user bubble → assistant reply → result cards with "почему подошло"; glass input bar; light (desktop) + dark (mobile).

- [ ] **Step 3: Commit**

```bash
git add design/screens/ai-search.html
git commit -m "design: add Apple-system ai-search reference screen"
```

---

## Task 5: Add a gallery index for the screens

**Files:**
- Create: `design/screens/index.html`

- [ ] **Step 1: Create the index**

A single self-contained HTML page (system font stack, light theme) with a title "Lia — Apple design system · reference screens" and 4 cards linking to the screens. Exact content:

```html
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Lia — reference screens</title>
  <style>
    body{ margin:0; padding:40px; background:#F2F2F7; font-family:-apple-system,system-ui,sans-serif; color:#000; }
    h1{ font-size:34px; font-weight:700; letter-spacing:-.022em; margin:0 0 4px; }
    p.sub{ color:rgba(60,60,67,.6); font-size:17px; margin:0 0 28px; }
    .grid{ display:grid; grid-template-columns:repeat(auto-fill,minmax(220px,1fr)); gap:16px; max-width:760px; }
    a.card{ display:block; background:#fff; border-radius:16px; padding:18px; text-decoration:none; color:#000;
            box-shadow:0 1px 3px rgba(0,0,0,.08); }
    a.card .k{ font-size:12px; font-weight:600; text-transform:uppercase; letter-spacing:.03em; color:#007AFF; }
    a.card h3{ font-size:17px; font-weight:600; margin:4px 0 0; }
  </style>
</head>
<body>
  <h1>Lia · reference screens</h1>
  <p class="sub">Apple design system (HIG), web-first, светлая + тёмная тема. См. <a href="../DESIGN.md">DESIGN.md</a>.</p>
  <div class="grid">
    <a class="card" href="discovery.html"><span class="k">Экран 1</span><h3>Главная / Discovery</h3></a>
    <a class="card" href="create-event.html"><span class="k">Экран 2</span><h3>Создание события</h3></a>
    <a class="card" href="event-detail.html"><span class="k">Экран 3</span><h3>Детали + регистрация</h3></a>
    <a class="card" href="ai-search.html"><span class="k">Экран 4</span><h3>AI-поиск / ассистент</h3></a>
  </div>
</body>
</html>
```

- [ ] **Step 2: Verify links work**

Run: `open design/screens/index.html`
Expected: 4 cards; clicking each opens the corresponding screen.

- [ ] **Step 3: Commit**

```bash
git add design/screens/index.html
git commit -m "design: add reference-screens gallery index"
```

---

## Task 6: Write the Apple design system doc (`design/DESIGN.md`)

**Files:**
- Create: `design/DESIGN.md`

- [ ] **Step 1: Author the doc**

Write `design/DESIGN.md` as the canonical Apple-based design system. Source the exact values from the spec at `docs/superpowers/specs/2026-06-11-apple-design-system-rework-design.md` — reproduce these sections as the doc's body (not by reference, copy the actual tables):

1. **Philosophy** — adopt Apple HIG, invent minimally; distinctiveness from content + Russian curatorial voice; design lives in code. (Spec §1, §3 "stays/changes".)
2. **Color tokens** — the full light/dark table verbatim from spec §4 "Color".
3. **Typography** — system font stack string + the type ramp table from spec §4.
4. **Spacing / radius** — from spec §4.
5. **Materials & elevation** — Liquid Glass on chrome only; `backdrop-filter: saturate(180%) blur(20px)`; subtle system shadows on cards/sheets.
6. **Component mapping** — the table from spec §5.
7. **Reference screens** — the list from spec §6, each linking to its file in `screens/` (e.g. `[Главная / Discovery](screens/discovery.html)`).
8. **Decisions** — accent = systemBlue (swappable token); system-font-stack only (no SF webfont; Legal note); light+dark from day one. (Spec §2, §9.)

Open the doc with a one-line note: "Active design direction. Supersedes the `stitch_/curatorial_minimalist` system, which is kept as historical reference."

- [ ] **Step 2: Verify completeness**

Run: `grep -c "007AFF\|0A84FF\|-apple-system\|Liquid Glass\|systemBlue" design/DESIGN.md`
Expected: count ≥ 4 (color tokens, font stack, materials, and accent decision all present). Visually confirm the color table has both light and dark columns.

- [ ] **Step 3: Commit**

```bash
git add design/DESIGN.md
git commit -m "design: add Apple HIG design system doc"
```

---

## Task 7: Update READMEs to point at the Apple system

**Files:**
- Modify: `design/README.md`
- Modify: `README.md` (root)
- Modify: `frontend/README.md`

- [ ] **Step 1: Rewrite `design/README.md`**

Replace its body so it describes the active Apple system. Exact content:

```markdown
# Design

Design system and reference screens for Lia.

**Active direction:** Apple HIG-based design system — web-first, light + dark. See [`DESIGN.md`](DESIGN.md).

- `DESIGN.md` — the design system: tokens (light/dark), typography (system font stack), materials (Liquid Glass), component mapping.
- `screens/` — 4 reference screens as standalone HTML (open [`screens/index.html`](screens/index.html)): discovery, create-event, event-detail, ai-search. Each shows light (desktop) + dark (mobile).
- `stitch_/` — **historical archive.** The earlier bespoke "Curatorial Minimalist" exploration, superseded by the Apple system. Kept for reference only.

Source brief: [`../docs/design_agent_prompt.md`](../docs/design_agent_prompt.md). Rework spec: [`../docs/superpowers/specs/2026-06-11-apple-design-system-rework-design.md`](../docs/superpowers/specs/2026-06-11-apple-design-system-rework-design.md).
```

- [ ] **Step 2: Update the design row in root `README.md`**

Find the table row that begins with `| `design/` |` and replace its description cell with:

```
Design system + reference screens. Active direction: **Apple HIG-based** (web-first, light/dark) — see `design/DESIGN.md`. `stitch_/` kept as historical archive.
```

- [ ] **Step 3: Update `frontend/README.md` design reference**

In `frontend/README.md`, replace the sentence that references the design direction so it reads:

```markdown
Planned stack: **Next.js + TypeScript + Tailwind**, implementing the Apple HIG-based design system in [`../design/DESIGN.md`](../design/DESIGN.md) (system font stack, Apple semantic colors, Liquid Glass, light + dark). Reference screens: [`../design/screens/`](../design/screens/).
```

- [ ] **Step 4: Verify**

Run: `grep -rl "Apple HIG" design/README.md README.md frontend/README.md`
Expected: all three paths listed.

- [ ] **Step 5: Commit**

```bash
git add design/README.md README.md frontend/README.md
git commit -m "docs: point READMEs at the Apple design system"
```

---

## Task 8: Push and clean up

**Files:** none (git + process)

- [ ] **Step 1: Push all commits**

Run: `git push origin main`
Expected: commits from Tasks 1–7 land on `Pashteto/lia`.

- [ ] **Step 2: Stop the visual companion server**

Run: `/Users/dodonovpavel/.claude/plugins/cache/claude-plugins-official/superpowers/5.1.0/skills/brainstorming/scripts/stop-server.sh /Users/dodonovpavel/gateway_fm/REAL_WORLD_ASSETS/1-lia/.superpowers/brainstorm/83441-1781186107`
Expected: server stops. (Mockup files persist under `.superpowers/`, which is gitignored — fine.)

- [ ] **Step 3: Final verification**

Run: `git status && ls design/screens/`
Expected: clean working tree; `screens/` contains `index.html`, `discovery.html`, `create-event.html`, `event-detail.html`, `ai-search.html`.

---

## Self-Review

**Spec coverage:**
- Spec §6 (4 screens) → Tasks 1–4. ✓
- Spec §4 tokens, §5 components, §3 philosophy → Task 6 (`DESIGN.md`). ✓
- Spec §7 deliverables: `DESIGN.md` (T6), `design/screens/` (T1–5), `design/README.md` + root/frontend READMEs (T7). ✓
- Spec §2 decisions (accent, font, themes) → Task 6 §8. ✓
- `stitch_/` kept as historical archive → Task 6 opening note + Task 7. ✓
- Out of scope (frontend code, backend, iOS) → not in plan. ✓

**Placeholder scan:** No TBD/TODO. Screen contents reference exact approved source files + an exact, mechanical transformation (omit 3 scaffolding blocks, wrap in given HTML shell). `index.html` and README contents are given verbatim. `DESIGN.md` content is sourced from named spec sections with explicit instruction to copy the actual tables. No undefined references.

**Type/path consistency:** File names used consistently across tasks and READMEs: `discovery.html`, `create-event.html`, `event-detail.html`, `ai-search.html`, `index.html`. Scoped CSS class prefixes (`.lia`, `.lia2`, `.lia3`, `.lia4`) match their source fragments. Branch is `main`; remote is `origin` (`Pashteto/lia`).
