# Walk/Excursion Category + Filter-Bar Axis Separation — Design

**Date:** 2026-08-11 · **Status:** approved (variant B)

## Goal

1. Add the event format «Прогулки и экскурсии» to the curated category taxonomy.
2. Visually separate the two filter axes in the feed filter bar — when/where chips
   (Сегодня, Выходные, Рядом со мной) vs topic chips (categories) — which currently
   render as identical Chips in one undifferentiated row.

Requested alongside: redeploy to presence.tarski.ru and a full QA scenario pass
(including admin flows) with a written per-scenario/per-screen analysis. Those are
operations, not design; they follow the existing runbooks and QA docs.

## 1. Taxonomy: «Прогулки и экскурсии»

Exact precedent: migration `000021_reading_group_category` (added «Читательские группы»).

- **Migration `000025_walk_category`**: `INSERT INTO categories (id, slug, label, sort_order)
  VALUES (gen_random_uuid(), 'walk', 'Прогулки и экскурсии', 90) ON CONFLICT (slug) DO NOTHING;`
  Down: `DELETE FROM categories WHERE slug = 'walk';` (guarded like 000021's down).
- **Seed**: no change needed — `seed.sql` only references existing slugs for demo events;
  categories themselves are seeded by migrations. (Optionally tag one seeded event later.)
- **Frontend**: zero code change. Categories come from `GET /api/v1/categories`
  (ISR 300s), create-event form and feed chips render the API list, numerals are
  positional (`categoryNumeral`).

## 2. Filter bar: variant B (approved)

Swiss Grid principle: structure is information; categories are numerals, never colours.

In `frontend/components/DiscoveryFeed.tsx` filter bar:

- **Axis micro-labels**: non-interactive 9px/0.12em uppercase muted labels prefixing
  each group — `КОГДА И ГДЕ` before the time/nearby chips, `ТЕМА` before the category
  chips. Rendered as plain `<span>` (not Chip) so they read as structure, not controls.
- **Vertical rule**: 1px ink rule (`self-stretch w-px bg-ink`) between the two groups,
  full bar height.
- **Category numerals**: each category chip label becomes `NN Label`
  (`01 Лекции` … via existing `categoryNumeral(slug, categories)`), numeral in the
  chip set in font-mono to match card numerals. Time/nearby chips stay numeral-free —
  the numeral itself is the species marker linking filter chips to card numerals.
- Overflow behaviour unchanged (single `overflow-x-auto` row, `justify-between`);
  labels and rule scroll with content. No new components; no Chip API change.

## 3. Out of scope

- Other chip rows (map browse, admin filters) — feed only, per request.
- Featured/taxonomy admin UI (#4/#5) — unchanged.

## 4. Verification

- `pnpm build && pnpm test && pnpm lint`; migration applies on local DB.
- Visual check at desktop + 390px (chips scroll, labels legible, rule visible).
- Post-deploy: `/api/v1/categories` returns `walk`; feed shows «09 Прогулки и экскурсии»
  chip; create-event form offers the new format.

## 5. Deploy + QA pass (operational)

- Deploy per `docs/superpowers/runbooks/2026-08-09-qa-09aug-deploy.md` (build on Mac →
  save|ssh|load; scp `000025` to `/opt/lia/backend/db/migrations` before migrate; static
  binary check; image prune after verify).
- QA: Chrome automation over `docs/qa/2026-07-14-first-clients-test-scenarios.md` +
  design-pass prompts: anon → register/login → organizer (create/edit/cancel; new walk
  format) → RSVP/applications → admin (`poulissimo@gmail.com`, temporary password —
  not recorded here): moderation, complaints, organizers, overview. Output: analysis
  report per scenario and per screen.
