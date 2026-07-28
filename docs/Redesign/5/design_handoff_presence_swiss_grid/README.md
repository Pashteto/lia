# Handoff: Presence — Swiss Grid redesign (full system)

## Overview
Presence (presence.tarski.ru) is a Moscow platform for participatory cultural events —
mediations, lectures, film screenings, festivals. This handoff covers a complete visual and
structural redesign of the product in a **Swiss International Typographic** system: one
grotesque typeface, a six-step type scale, 1px hairline rules, modular cells, and almost no
colour. It spans **17 screens across three layers**:

- **User** (public) — 8 screens
- **Organizer** (workspace) — 5 screens
- **Admin** (internal, inverted to black) — 4 screens

Every screen is specified at two breakpoints (desktop ≥1024, mobile ≤430) except admin
A2–A4, which are desktop-only by design.

## About the design files
The files in this bundle are **design references authored in HTML**. They are prototypes that
show intended look, structure, and content — **not production code to copy**. The task is to
**recreate these designs in the target codebase's own environment** (React/Next, Vue, etc.)
using its established patterns, routing, and data layer. If no frontend exists yet, choose the
framework and implement there. Do not import the prototype HTML, its `support.js` runtime, or
its `<sc-for>`/`<x-dc>` custom tags — those belong to the design tool, not to the product.

Two things in the prototype ARE meant to be carried over as real implementations:
- `map-embed.html` — a Leaflet + OpenStreetMap embed, desaturated via CSS filter. Reimplement
  as a real map component (react-leaflet or MapLibre) with the same visual treatment.
- `image-slot.js` — a drag-and-drop image placeholder used only for design purposes. Replace
  with the product's real image/upload component.

## Fidelity
**High fidelity.** Colours, type sizes, weights, letter-spacing, borders, and grid column
widths in this document are final and should be matched. Copy (Russian) is final unless the
content team overrides it. Interaction states beyond what is described below (focus rings,
transitions) are left to the implementer, following the rules in *Interaction* → *States*.

---

## Design tokens

Machine-readable versions are in `tokens.css` (CSS custom properties) and `tokens.ts`
(typed object). Values below are the source of truth.

### Colour
| Token | Hex | Use |
|---|---|---|
| `ink` | `#111111` | Text, rules, fills, buttons, admin background |
| `paper` | `#F2F0EC` | Surface / app background |
| `canvas` | `#DEDBD4` | Page background behind surfaces (design-doc only; in-app use `paper`) |
| `white` | `#FFFFFF` | Preview cards, inverted button text |
| `signal` | `#E2231A` | **Signal only** — moderation flags, test data, alerts, destructive |
| `muted` | `#7D786E` | Kicker / eyebrow text |
| `muted-2` | `#8A857C` | Caption text, admin secondary text |
| `body-dim` | `#4F4A42` | Body copy on paper |
| `field-text` | `#6B665E` | Placeholder / filled-field text |
| `rule-light` | `#DDDDDD` | Inner hairline on paper |
| `rule-grid` | `#E0DCD4` | Calendar cell rules |
| `rule-dark` | `#3A3733` | Inner hairline on admin black |
| `admin-head` | `#1C1A18` | Admin table header row |
| `text-dim-dark` | `#CFCABF` | Body copy on black |
| `text-dim-dark-2` | `#A8A299` | Secondary copy on black |
| `signal-tint` | `#FFD9D6` | Caption on a red-filled cell |
| `inactive` | `#DCD8D0` | Empty progress segment, grid demo blocks |
| `table-head` | `#E6E3DC` | Light table header row |

**Colour rules (non-negotiable):**
1. Categories are **numerals**, never colours: `01` Фестивали, `02` Медиации, `03` Лекции,
   `04` Кино, `05` Спектакли, `06` Концерты.
2. `signal` red is reserved for *needs attention / is wrong*. Never decorative, never a brand
   accent. Occurrences: moderation status chip, "N заявок ждут" left border, admin
   "Ждут модерации" tile, test-data rows, "ОТКЛОНИТЬ" / destructive buttons.
3. Max two background colours per screen (`paper` + `ink`).

### Typography
Three families, loaded from Google Fonts:
- **Archivo** — 400 / 500 / 600 / 700 / 900. Default UI + all headings.
- **Space Grotesk** — 400 / 500 / 700. Section headings in the doc, kickers, captions, labels, long prose.
- **JetBrains Mono** — 400 / 700. **All numbers**: dates, counts, seats, IDs, times, prices in tables.

Scale (px, all with the listed letter-spacing — do not substitute):

| Role | Size | Weight | Tracking | Line-height |
|---|---|---|---|---|
| Page hero | 60 | 900 | -0.03em | 0.94 |
| Section head | 42 / 30 | 800–900 | -0.02/-0.03em | 0.94–1.05 |
| Screen title (desktop) | 38 / 34 / 30 / 26 | 900 | -0.03em | 0.94 |
| Screen title (mobile) | 22 / 21 / 20 | 900 | -0.02/-0.03em | 1.05 |
| Card title (desktop) | 15 | 900 | -0.02em | 1.02 |
| Card title (mobile) | 12–13 | 700 | 0 | 1.1 |
| Big number | 26 / 22 | 700 (mono) | 0 | 1 |
| Value (`.val`) | 12 | 700 | 0 | 1.25 |
| Body | 12.5 / 11.5 | 400 | 0 | 1.4–1.5 |
| Caption (`.cap`) | 9–10 | 400 | 0.13em | 1.3, UPPERCASE |
| Label (`.lbl`) | 10 | 700 | 0.14em | UPPERCASE |
| Kicker (`.kick`) | 11 | 700 | 0.18em | UPPERCASE |
| Chip | 8–9 | 400 | 0.12em | UPPERCASE |
| Button | 9–11 | 700 | 0.07em | UPPERCASE |
| Nav item | 9 | 400 | 0.14em | UPPERCASE |

Mobile sizes are the small end of each pair. **Never round tracking to 0** — the letter-spacing
on caps text is what makes the system read as Swiss rather than generic.

### Space & geometry
- Spacing scale: **2, 4, 5, 6, 8, 10, 12, 14, 16, 18, 20, 26, 40, 48, 56 px**. Cell padding is
  `10px 14px` (dense) or `16px 20px` (roomy); mobile cell padding `9px 14px`.
- **Border radius: 0 everywhere.** The only rounded thing in the whole system is the phone
  bezel in the design doc, which is not part of the product.
- **Rules: 1px solid.** `#111` on paper, `#F2F0EC` on black for structural divisions;
  `#DDD` / `#3A3733` for inner separations. Section dividers in the doc are 2px.
- **Shadows: none in-product.** The prototype's card shadows are the design-doc presentation
  frame only.
- Grid: 12 columns / 6 modules conceptually; in practice screens use explicit
  `grid-template-columns` per screen, listed below.
- Desktop viewport modelled at **680px wide** in the doc (900px for admin A2–A4) — treat these
  as proportional, not literal. Build fluid to a 1360px max content width with 48px gutters.
- Mobile modelled at **262px** interior — treat as a 375–430px phone. Scale type up to the
  mobile column of the scale table; **tap targets minimum 44px** (the prototype's dense rows
  must grow on real devices).

---

## Shared components

Build these six before any screen. Each maps to a prototype class.

### 1. `AppHeader` (`.hd` / `.hdm`)
Padding `13px 20px` desktop / `11px 14px` mobile. `display:flex; justify-content:space-between;
align-items:baseline`. Left: wordmark `PRESENCE` (Archivo 900, 13px desktop / 11px mobile,
tracking -0.01em). Right: nav (`.nv`) — 9px uppercase, 0.14em tracking, 14px gap; the active
item carries `border-bottom: 2px solid currentColor; padding-bottom: 2px`. Admin variant:
wordmark is `PRESENCE / ADMIN`, colours inverted, bottom rule `1px solid #F2F0EC`.
Mobile header shows a context caption instead of nav.

### 2. `Chip` (`.chip`)
`border: 1px solid currentColor; padding: 4px 9px; font-size: 9px; letter-spacing: .12em;
text-transform: uppercase; white-space: nowrap; border-radius: 0`.
Variants: **default** (ink border, transparent bg), **active** (`.on` — ink bg, white text),
**signal** (`border-color/color: #E2231A`), **dark-active** (paper bg, ink text — admin),
**dark-muted** (`#8A857C` border and text — admin).
Used for filters, statuses, counters (label carries its count: `Все · 6`).

### 3. `Button` (`.cta`)
Primary: `background:#111; color:#fff; padding:11px; font-size:11px; font-weight:700;
letter-spacing:.07em; text-align:center; text-transform:uppercase`.
Ghost (`.cta.gh`): transparent bg, `1px solid #111`, ink text.
Inverted (admin primary): `#F2F0EC` bg, `#111` text.
Destructive (admin): `#E2231A` bg, `#fff` text.
Dark ghost (admin tertiary): `1px solid #8A857C`, `#8A857C` text.
Small size: `padding:7px 4px; font-size:9px` (in-row actions).

### 4. `Cell` (`.cell` + `.cap` + `.val`)
The atomic module of the whole system: padding `10px 14px`, an uppercase caption
(9px / 0.13em / `#8A857C`) over a value (12px / 700, or mono if numeric). Cells sit in
`grid-template-columns: repeat(N,1fr)` strips with `border-right:1px solid` between them and a
`border-bottom` closing the strip. This one component builds the fact rows on U2, U5, U6, O1,
O5, A1, A2.

### 5. `EventModule` (feed card)
Desktop: a grid cell, padding `12px 14px`, flex column, `min-width: 0`.
Row 1 — `justify-content: space-between; align-items: baseline`: category numeral (mono, 11px,
700) left, category name (`.cap`) right.
Row 2 — title, Archivo 900 / 15px / -0.02em / line-height 1.02.
Row 3 — venue (`.cap`).
Footer — pushed with `margin-top:auto; padding-top:10px`: date (mono 11px) left, price
(Archivo 900, 12px) right. Price is **always** bottom-right; free is the literal string `FREE`.
Mobile: `grid-template-columns: 22px 1fr auto`, numeral / title / price on line 1, venue · date
spanning line 2, title drops to 700 / 12.5px.
AI variant (U3) appends a match reason on a `1px solid #DDD` top rule: 10px, `Совпало: …`.

### 6. `BottomTabBar` (`.tabs`)
Mobile only, `flex`, four equal tabs, `border-top:1px solid #111`. Each tab: 16×16 square icon
(`1.5px solid currentColor`, filled solid when active), 8px uppercase label, 0.1em tracking,
`#8A857C` inactive / `#111` + 700 active. Tabs: **Лента · Подбор · Карта · Я** (Календарь
replaces Карта on U4's variant — pick one four-item set at build time and keep it fixed).

Secondary components: `ProgressBar` (`height:8px; border:1px solid #111`, inner `background:#111`
at percentage width — 5–7px in dense rows); `Field` (label `.cap` above, box
`1px solid #111; padding:9px 11px; font-size:12.5px`, no radius, no focus glow — use a 2px ink
outline on focus); `StatusChip` (Chip with the status colour map below); `Stepper` (O2/O5 —
equal-width segments, completed/active filled ink with white text, upcoming on paper, divided
by 1px rules).

**Status → chip variant map**
| Status | Variant |
|---|---|
| Опубликовано / Подтверждено / Верифицирован | active (ink fill) |
| Черновик / Прошедшее | default |
| На модерации / Ожидает / На проверке / Тестовый | signal |

---

## Screens

Prototype file: `Presence Swiss Grid - Full System.dc.html`. Each screen is anchored by a badge
(`U1`, `O2`, `A3`…) in that file — search the badge string to find its markup.

### Part 1 — User

#### U1 · Лента событий (feed) — `/events`
**Purpose:** browse what's on; the product's home screen.
**Desktop layout:** header → title block (`padding:18px 20px`, caption `Москва · 42 события ·
Июль 2026`, H1 38px/900 two lines) → filter bar (`justify-content:space-between`, 9px 20px):
left group = time filters `Все / Сегодня / Выходные / Бесплатно`, right group = category
filters `Медиации / Лекции / Кино` → catalogue grid `grid-template-columns: 1fr 1fr 1fr`,
`flex:1`, cells divided by `border-right`/`border-bottom` hairlines, each an `EventModule`.
Six modules shown; real feed paginates or infinite-scrolls in the same grid.
**Mobile:** header (wordmark + `МСК · 42`) → title 22px → one horizontal filter row
(3 chips, overflow hidden — make it a scrollable row) → stacked `EventModule` mobile rows →
`BottomTabBar` with Лента active.
**Data:** `{ i, cat, t, v, d, p }` — index numeral, category, title, venue, date, price string.
**Copy:** titles as in `feed` array of the prototype (real data replaces them).

#### U2 · Страница события (event detail) — `/events/:id`
**Purpose:** decide and register.
**Desktop:** header with `← СОБЫТИЯ / 01` breadcrumb → **cover strip**: two images side by side,
`1fr 1fr`, 190px tall, divided by a hairline → **title block** `1fr 200px`: left = category line
(`01 · Фестивали · Очно`), H1 30px/900, 12px description capped at 52ch; right = price cell
(caption `Цена`, value 26px/900 `FREE`) with the primary `ЗАПИСАТЬСЯ` button pinned to the
bottom (`margin-top:auto`) → **fact strip**: four `Cell`s — Когда / Начало / Места
(mono `12 / 40`) / Организатор (`Гараж ✓`) → **venue block** `1fr 200px`: left = live map,
right = address cell (`Крымский Вал, 9, стр. 32`, metro line 11px `#6B665E`) with ghost
`МАРШРУТ` at the bottom.
**Mobile:** back / wordmark / ♡ header → single 130px cover → title block → two-cell fact strip
(Когда, Места) → map filling remaining height → sticky footer bar: price 17px/900 + full-width
`ЗАПИСАТЬСЯ`.
**Map:** centre `55.7351, 37.6053`, zoom 15 desktop / 15 mobile, label `ГАРАЖ`, paper background.

#### U3 · AI-подбор (assisted discovery) — `/discover`
**Purpose:** natural-language search that never dead-ends.
**Desktop:** header (Подбор active) → prompt block: caption `AI-подбор`, H1 34px `Что вам сейчас
откликается?`, input row (`1px solid #111`, text 12px `#6B665E`, ink submit square with `→`,
`padding:11px 20px`), four suggestion chips (`Тихое в выходные`, `Бесплатно рядом`,
`Для двоих вечером`, `С детьми`) → **answer line**: single 12.5px sentence in its own bordered
strip → three result modules in `1fr 1fr 1fr`, each with a match reason → **escape hatch
footer**: `Не то? Соберите вручную` + chip `Точные фильтры →`.
**Rule:** the answer is one sentence, never a chat transcript. The escape hatch is mandatory.
**Mobile:** same order, results as stacked rows, tab bar with Подбор active.

#### U4 · Календарь — `/calendar`
**Purpose:** see the month, drill into a day.
**Desktop:** header → month bar (`Июль 2026` 26px/900 left; chips `Месяц / Список / ← / →`
right) → body `1fr 230px`: left = weekday strip (`ПН…ВС`, captions, centred) over a
`repeat(7,1fr)` × 5 grid with `grid-auto-rows:1fr`; each day cell has a mono date top-left and,
if it has events, an **ink-filled background with white numeral** plus the category numeral
bottom-left at 8px/700. Leading/trailing blanks are `#ECEAE4`. Cell rules are `#E0DCD4`.
Right rail = selected-day agenda: header cell (`Выбрано` / `12 июля, вс`), then one row per
event (mono time, optional `Записан` chip, 12.5px/700 title, venue caption), and a ghost
`ДОБАВИТЬ В КАЛЕНДАРЬ` pinned bottom.
**Mobile:** compact 26px-row month grid (numeral only, same fill logic), selected-date caption
strip, agenda rows, tab bar.
**Marked days in the mock:** 12→02, 15→03, 18→05, 20→04, 25→01.

#### U5 · Карта — `/map`
**Purpose:** find events by place.
**Desktop:** header → four-`Cell` stat strip (Всего 42 / Радиус 5 км / Бесплатно 31 / Сегодня 6,
all mono) → body `230px 1fr`: left rail = filter chips row then numbered list rows
(mono numeral, 12px/700 title, `venue · distance` caption) — **list numerals must match the map
pin numerals**; right = the map, with a floating `ИСКАТЬ В ЭТОЙ ОБЛАСТИ` pill centred 12px from
the top (paper bg, 1px ink border, 9px/700, 0.1em, z-index above tiles).
**Mobile:** two-cell stat strip → map → one selected-event card at the bottom
(mono `01 · 0.8 КМ` + `FREE`, title 13px/700) → tab bar.
**Map:** centre `55.7420, 37.6180`, zoom 12 desktop / 11 mobile.

#### U6 · Мои записи и профиль — `/me`
**Purpose:** the user's registrations, as a table of facts.
**Desktop:** header (Профиль active) → identity strip `1fr 200px`: left = caption
`Участник с марта 2026` + name 30px/900; right = two stacked `Cell`s (Посещено 14,
Подписки 5) → tab chips `Предстоящие · 2 / Прошедшие · 14 / Избранное · 7 / Подписки · 5` →
registration rows on `grid-template-columns: 56px 1fr 118px 134px`: mono date, title + venue·time
caption, organizer cell, status chip centred. Ends with an inline empty note
(`Больше записей пока нет` + chip `Найти события →`).
**Mobile:** identity block → three-cell stat strip → two tab chips → compact rows
(mono `12.07 · 16:00` + short status chip `ОК` / `ЖДЁМ`) → tab bar with Я active.

#### U7 · Вход и регистрация — `/login`, `/signup`
**Desktop:** two equal halves, no header. Left = **ink panel**: wordmark top, headline
34px/900 pushed to the bottom (`Медиации, лекции и разговоры об искусстве`, four lines),
caption `42 события в Москве · июль 2026` in `#8A857C`. Right = paper form: caption `Вход`,
H2 22px `С возвращением`, Почта field, Пароль field, primary `ВОЙТИ`, and a footer row pushed
to the bottom: `Нет аккаунта?` + underlined `Регистрация`.
**Mobile:** ink header block (wordmark + 22px headline) over the form; primary `ВОЙТИ` plus
ghost `СОЗДАТЬ АККАУНТ`.
**Registration** reuses the same frame: swap the right-hand copy and add Имя + confirm-password
fields.

#### U8 · Состояния (designed states)
Three 326×250 frames — apply the pattern to every empty/blocked/error surface in the app.
1. **Empty** — mono `00` at 38px, headline 17px/900 `Записей пока нет`, 11.5px explanation,
   primary `НАЙТИ СОБЫТИЕ` (self-start, not full width).
2. **Auth required** — caption `Доступ`, headline `Войдите, чтобы видеть свои записи`,
   reassurance `Лента и карта доступны без входа.`, primary `ВОЙТИ` + ghost `К ЛЕНТЕ`.
3. **404** — **inverted to ink**, mono `404` at 44px / -0.04em, headline
   `Событие не найдено`, 11.5px `#A8A299` explanation, paper button `ВЕРНУТЬСЯ К ЛЕНТЕ`.
**Rule:** every state names the situation, explains in one sentence, and offers exactly one
obvious next action (two at most).

### Part 2 — Organizer

#### O1 · Кабинет (dashboard) — `/org`
Header with organizer nav (`Кабинет / Мои события / Заявки / Профиль`) → identity strip
`1fr 190px`: caption `Организатор · Студия «Лиа» ✓`, H1 `Кабинет` 30px; right = full-width
primary `+ СОЗДАТЬ СОБЫТИЕ` bottom-aligned → **status strip**, four cells, mono 26px numbers:
Опубликовано 03 / **На модерации 02 (ink-filled cell, inverted)** / Черновики 01 /
Всего записей 86 → **action banner**: `border-left: 4px solid #E2231A`, mono 22px red `05`,
12.5px/700 `новых заявок ждут подтверждения`, chip `Смотреть →` → bottom `1fr 1fr`:
left = next event (title 17px/900, `12 июля · 16:00 · ГМИИ`, fill bar `28 / 40` at 70%),
right = activity log, three rows separated by `#DDD` rules, each `label · relative time`.
**Mobile:** three-cell status strip, the same red banner, next-event block, sticky
`+ СОЗДАТЬ СОБЫТИЕ`.

#### O2 · Создание события (creation wizard) — `/org/events/new`
Four steps: **01 Основное · 02 Когда и где · 03 Билеты · 04 Публикация**.
Header right shows autosave state: `ЧЕРНОВИК СОХРАНЁН · 14:22`.
Step strip = four `Cell`s; the active step is ink-filled with `#A8A299` caption.
Body `1fr 240px`: left = the step's fields with 11px gaps (Название → Категория as numbered
chips → Описание textarea (52px) → Обложка dropzone (62px, 3:2, placeholder
`Перетащите обложку · 3:2`)) and a bottom action row: primary `ДАЛЕЕ · КОГДА И ГДЕ` + ghost
`ЧЕРНОВИК`. Right rail = **live preview** of the feed module on a white card
(`1px solid #111`, `background:#fff`) that updates as fields change, plus a bottom note
`После отправки — Модерация занимает до 24 часов. Событие появится в ленте после одобрения.`
**Mobile:** four-segment progress bar (5px, ink filled / `#DCD8D0` empty, 1px dividers),
`01/04` in the header, fields stacked, moderation note above the `ДАЛЕЕ` button.
**Behaviour:** autosave draft on blur; the next-step label always names the destination;
preview reflects the current field values live.

#### O3 · Мои события — `/org/events`
Title bar (`Мои события` 26px + small primary `+ СОЗДАТЬ`) → counting tab chips
(`Все · 6 / Опубликовано · 3 / Модерация · 2 (signal) / Черновики · 1`) → table header row on
`56px 1fr 96px 110px 92px` with `background:#E6E3DC` → rows: mono date / title + venue /
seats `28 / 40` with a 5px fill bar underneath / status chip / actions (`Ред.` `Копия`
stacked, 9px caption, line-height 1.7).
**Mobile:** chips row then stacked rows — mono date + status chip on line 1, title, seats + bar.

#### O4 · Заявки участников — `/org/events/:id/applications`
Context strip `1fr 200px`: left = caption `Событие · 12 июля` + event title 22px; right =
`Заполнено` cell with mono `28 / 40` and a 6px fill bar → tab chips `Новые · 5 / Принятые · 28 /
Отклонённые · 2` with `Выбрать все` on the right → rows on `26px 1fr 120px 150px`:
11×11 square checkbox / name 12.5px/700 + context caption (`Была на 3 событиях · медиации`,
`Первая заявка`, `Была отмена · 1 раз`) / `Заявка` cell with relative mono time / inline
`ПРИНЯТЬ` (primary) + `ОТКЛОНИТЬ` (ghost) → **bulk bar** pinned bottom: `Выбрано: N` +
`ПРИНЯТЬ ВЫБРАННЫЕ`.
**Mobile:** capacity block at top, then per-person cards with the two actions full width.
**Rule:** the seat counter must update optimistically as applications are accepted.

#### O5 · Профиль и верификация — `/org/profile`
**Verification stepper** across the top: `01 Заявка подана` / `02 Документы проверены` /
`03 Верифицирован ✓` — completed steps ink-filled with white text and paper dividers, the
current/last step on paper.
Body `1fr 250px`: left form = Название, Описание (46px), a row with a 62×62 logo slot and two
contact fields (`lia@studio.ru`, `t.me/liastudio`), primary `СОХРАНИТЬ` pinned bottom.
Right = **public preview** card on white: 34px ink avatar square, name + ✓, caption
`Проверенный организатор`, description, and a footer with mono `Событий 08` /
`Подписчиков 142`.
**Mobile frame shows the public organizer page** (what a viewer sees): identity block,
two stat cells, upcoming event rows, primary `ПОДПИСАТЬСЯ`.

### Part 3 — Admin (inverted)

The admin surface is **`#111` background with `#F2F0EC` text**, structural rules in
`#F2F0EC`, inner rules `#3A3733`, table header rows `#1C1A18`, secondary text `#8A857C`,
body copy `#CFCABF`. Same grid, same type, same components — inversion is the only signal that
this is an internal tool. Desktop only (A1 also has a mobile "duty mode").

#### A1 · Обзор — `/admin`
Four-tile stat strip: Событий всего 142 / **Ждут модерации 07 on a `#E2231A` fill** (caption
`#FFD9D6`, number white) / Организаторов 38 / Пользователей 2 914 → two queues side by side:
left = **Очередь модерации** (event title 11.5px/700 + relative mono age, three rows) with a
paper `ОТКРЫТЬ ОЧЕРЕДЬ` button pinned bottom; right = **Заявки на верификацию · 3** rows plus a
`Сигналы` footer showing `2 события с тестовыми данными` in red 11px/700.
**Mobile duty mode:** two tiles (Модерация red / Верификация), three queue rows with
`age · organizer` captions (test-data row in red), paper CTA.

#### A2 · Модерация событий — `/admin/moderation`
900×460 working surface, `250px 1fr`.
Left = queue: filter chips (`Ждут · 7` active in paper, `Все` muted), then rows — mono ID
(`EV-1042`), relative age, title, organizer caption. The **selected row is paper-filled with ink
text**; test-data rows render their title in `#E2231A`.
Right = the record: cover 120px + meta block (`EV-1042 · подано 2 ч назад`, title 20px,
`Музей «Гараж» ✓ · Фестивали`) → four-cell fact strip (Дата / Цена / Мест / Формат) →
description block → **rejection reasons** as muted chips (`Тестовые данные`, `Нет описания`,
`Обложка низкого качества`, `Дубликат`) → action bar: `ОДОБРИТЬ` (paper), `ОТКЛОНИТЬ` (red),
`НА ДОРАБОТКУ` (dark ghost).
**Behaviour:** rejecting requires ≥1 reason chip; on decision, advance to the next queue item
without leaving the screen.

#### A3 · Организаторы и верификация — `/admin/organizers`
Filter chips (`Все · 38` / `На проверке · 3` signal / `Верифицированы · 29` / `С жалобами · 2`)
with `Поиск ⌕` right → table on `44px 1fr 110px 90px 90px 170px`: mono ID, name + email
caption, status chip (colour by map — `На проверке`/`Тестовый` in signal), mono event count,
mono complaint count (red when > 0), row actions (contextual primary
`ПРОВЕРИТЬ`/`ОТКРЫТЬ`/`УДАЛИТЬ` + `···` overflow).
Test organizers (`QA Block8`) render the whole name in red.

#### A4 · Пользователи и контент-гигиена — `/admin/users`
Split `1fr 300px`.
Left = user registry, `44px 1fr 96px 84px 96px`: mono ID, name + email, mono registration
month, mono booking count, role chip (`Зритель` / `Организатор` / `Тестовый`). Test accounts
in red.
Right = **Гигиена контента · 4** panel: one block per issue — issue type caption
(`Тестовые данные`, `Подозрительная цена`), the offending value in 11px/700, the source
organizer as a caption. Footer: destructive `СКРЫТЬ ВСЁ ИЗ ЛЕНТЫ`.
**Why it exists:** test events and nonsense prices currently leak into the public feed. This
panel is the operational fix.

---

## Interactions & behaviour

### Navigation
- User nav (desktop): События · Подбор · Календарь · Карта · Организаторам · Войти.
- Organizer nav: Кабинет · Мои события · Заявки · Профиль.
- Admin nav: Обзор · Модерация · Организаторы · Пользователи.
- Active item = 2px bottom rule, current colour. No hover colour change on nav; underline only.
- Mobile user navigation is the bottom tab bar; organizer/admin mobile use header context only.

### States
- **Hover** (desktop, pointer devices): interactive cells and rows invert to
  `background:#111; color:#F2F0EC` — never a tint or a shadow. Chips invert the same way.
  Primary buttons hover to `#000`. On admin, hover fills with `#1C1A18`.
- **Focus:** `outline: 2px solid #111; outline-offset: 0` (paper) / `#F2F0EC` (admin). No radius.
- **Active/pressed:** no scale, no translate. Ink fill stays; opacity 0.9 is acceptable.
- **Disabled:** `#8A857C` text on `#DCD8D0`, no border change.
- **Loading:** hairline-boxed skeleton cells at final dimensions (`background:#ECEAE4`), never
  spinners or shimmer gradients. Numbers show `—`.
- **Empty / auth-gate / error:** always follow the U8 pattern.
- **Transitions:** 120ms linear on background and colour only. No easing curves, no motion on
  layout. The system should feel printed, not animated.

### Forms & validation
- Label above field, always `.cap` style. No placeholder-as-label.
- Errors: field border and label switch to `#E2231A`; the message sits below the field at
  11px in `#E2231A`. No icons.
- Event creation: title required (≤120 chars), category required, description required at
  step 04, cover optional but warned, seats numeric > 0. Draft autosaves and shows the time.
- Registration on an event that is full → `ЗАПИСАТЬСЯ` becomes `В ЛИСТ ОЖИДАНИЯ` (ghost).

### Responsive
- Two designed breakpoints. Below 720px the desktop layouts collapse to the mobile
  compositions described per screen — multi-column cell strips reduce from 4 → 2 or 3 cells,
  side rails move below the primary content, tables become stacked rows.
- Content width caps at 1360px with 48px gutters; below 900px gutters drop to 20px.
- Tap targets ≥44px on touch — the prototype's 9–11px row padding must grow.

### Data & state per screen
| Screen | Client state | Fetches |
|---|---|---|
| U1 | active time filter, active categories, page | `GET /events?filters` |
| U2 | registration status | `GET /events/:id`, `POST /events/:id/register` |
| U3 | prompt, results, answer sentence, loading | `POST /discover` |
| U4 | month, selected day | `GET /events?month` |
| U5 | viewport bounds, selected pin, filters | `GET /events?bbox` |
| U6 | active tab | `GET /me/registrations?tab` |
| U7 | form + error | `POST /auth/login`, `/auth/signup` |
| O1 | — | `GET /org/summary` |
| O2 | step, draft object, preview derived from draft | `PATCH /org/drafts/:id`, `POST /org/events` |
| O3 | active tab | `GET /org/events?status` |
| O4 | selection set, tab | `GET /org/events/:id/applications`, `POST …/accept|reject` (single + bulk) |
| O5 | form, verification status | `GET/PATCH /org/profile` |
| A1 | — | `GET /admin/summary` |
| A2 | queue, selected id, chosen reasons | `GET /admin/moderation`, `POST /admin/events/:id/decision` |
| A3 | filter, search | `GET /admin/organizers` |
| A4 | filter | `GET /admin/users`, `GET /admin/hygiene` |

---

## Maps
All maps are **real interactive Leaflet maps on OpenStreetMap tiles**, desaturated to match the
system. See `map-embed.html` for the exact treatment. Reimplement natively:
- Tiles: standard OSM raster.
- Filter: `grayscale(1) contrast(1.05)` with the paper colour showing through — the map must
  read as a printed plan, not a colour map.
- Markers: square, ink-filled, **numbered in JetBrains Mono**, numbers matching the list order.
  No teardrop pins, no shadows, no rounded corners.
- Controls: default zoom controls restyled square/ink, or hidden in embedded contexts.
- Attribution required by OSM must remain visible (9px caption style).

## Assets
- **Event covers** — real photographs from `https://presence.tarski.ru/covers/*.jpg`
  (`festival.jpg`, `mediation.jpg` used in the mocks). Production must serve properly sized,
  cropped 3:2 images; covers are always `object-fit: cover`, never letterboxed.
- **Fonts** — Archivo, Space Grotesk, JetBrains Mono (Google Fonts, SIL Open Font License).
  Self-host in production; preload the Archivo 900 and JetBrains Mono 700 subsets.
- **Icons** — the system deliberately has almost none. The tab bar uses plain squares; `✓`,
  `→`, `←`, `♡`, `⌕`, `···` are typographic characters. **Do not introduce an icon library.**
- **No illustrations, no logos beyond the `PRESENCE` wordmark set in Archivo 900.**

## Files in this bundle
| File | What it is |
|---|---|
| `README.md` | This document — the specification. |
| `CLAUDE.md` | Build brief for an agent implementing this in a codebase. |
| `tokens.css` | Design tokens as CSS custom properties. |
| `tokens.ts` | The same tokens as a typed TS object. |
| `Presence Swiss Grid - Full System.dc.html` | The 17-screen design reference. Open in a browser. |
| `Presence Map Screens.html` | Earlier map explorations. |
| `map-embed.html` | The Leaflet/OSM map treatment to reproduce. |
| `image-slot.js` | Design-time image placeholder. Not for production. |
| `support.js` | Runtime required to open the `.dc.html` reference. Not for production. |

## Build order (recommended)
1. **Tokens + the six shared components.** Nothing else starts until `Cell`, `Chip`,
   `EventModule`, `Button`, `AppHeader`, `BottomTabBar` exist and match.
2. **U1 → U2 → U7 → U8.** The public core plus designed states.
3. **U4, U5, U6.** Calendar, map, profile — all reuse `EventModule` and `Cell`.
4. **U3.** Needs the discovery endpoint; ship the escape hatch even if the model is stubbed.
5. **O1 → O3 → O2 → O4 → O5.** Organizer workspace; the wizard is the largest single piece.
6. **A1 → A2 → A4 → A3.** Admin; A2 and A4 deliver the most operational value.

## Things not to get wrong
1. **Zero border radius. Anywhere.**
2. Categories are numerals, not colours.
3. Red only ever means "needs attention".
4. All numbers in JetBrains Mono, including dates and seat counts.
5. Keep the letter-spacing on every uppercase run.
6. Hover inverts; it does not tint.
7. Every empty, loading, gated, and error surface is designed — no blank pages.
8. Maps are real and desaturated; markers are numbered squares.
