# Design Agent Prompt — Event Discovery MVP (Web)

> Master brief for an AI agent (v0, Lovable, Bolt, Cursor, Claude, GPT, Figma AI) tasked with designing and implementing the web app pages for the **core functionality** of an event-discovery MVP. Output: working Next.js + TypeScript + Tailwind code. Visual direction: Luma-like — minimal, white, type-driven.

---

## 0. Role and mission

You are a senior product designer + front-end engineer. You will design and implement the web pages for a city event-discovery marketplace. Your output must be **production-quality, working Next.js code** that runs locally with mock data, not just static mockups. The design must feel like Luma (lu.ma): generous whitespace, big covers, large type, restrained color, and absolute clarity of hierarchy.

Treat this brief as the source of truth. Where details are missing, make reasoned product decisions and document them in `DESIGN_NOTES.md` alongside the code.

---

## 1. Product context (read this first)

The product is a **web-first discovery platform for participatory cultural practices** in a single Russian city on launch (Moscow), with iOS to follow later. It is **not** a generic city afisha and **not** a ticketing marketplace. It is a curated home for events where the visitor is a **participant**, not a spectator.

### What kind of events live here

- **Медиации** — facilitated walks through exhibitions, where a mediator opens a conversation with the group rather than narrating.
- **Мастер-классы и мастерские** — hands-on workshops led by artists, craftspeople, or practitioners.
- **Открытые лекции и беседы** — lectures conceived as dialogue, with built-in discussion.
- **Обсуждения и читательские группы** — book/film/exhibition discussions, critique circles, reading rooms.
- **Встречи с художниками и кураторами** — studio visits, Q&A, artist talks designed around exchange.
- **Перформативные практики и совместные действия** — collective practices, workshops, residency open days.

What is **not** in scope: passive concerts, stand-up gigs, club nights, paid masterclass courses with multi-week structure, conferences, sports, "go to a bar" listings. If an event has rows of seats and a stage, it likely doesn't belong here.

### Three audiences share the same backend

- **Visitors and registered participants** discover events, filter, save, sign up, add to calendar, and use an AI search assistant tuned for curatorial language ("хочу что-то про память и архив", "куда пойти одному поговорить про современное искусство").
- **Organizers** — museums, galleries, independent curators, artists, cultural projects — manage their profile, create and edit events, upload images, set capacity and participant criteria, submit for moderation, and see basic stats.
- **Admins and moderators** uphold the curatorial bar: review organizer accounts and events, handle complaints, manage categories and tags, surface featured practices.

### Voice and tone

The product is in Russian. **All visible UI copy must be in Russian.** The voice is curatorial and human, not commercial. Prefer "записаться" over "купить билет", "встреча" over "мероприятие", "участники" over "посетители", "ведущий / медиатор / художник" over "спикер". Avoid hype words ("крутое", "топ", "must-see"). Restraint is part of the brand.

Code, comments, file names, and component names stay in English.

Target load on MVP: up to 10 000 visits/day with weekend peaks. SEO matters for public event pages, so they must be server-rendered.

### Key product rules that affect design

- Events go through moderation. Status lifecycle: `draft → pending_review → published → rejected | cancelled`.
- **Capacity is almost always limited** (typical 8–30 participants). Show seats remaining prominently. When full, offer a waitlist.
- **Sign-up may be curated.** Some events accept everyone (open RSVP); some require a short answer to a curator's question before confirmation (application). The UI must support both.
- Payments are out of scope. Paid events show an **external link** (`external_ticket_url` or `external_registration_url`), never an inline checkout. Many events here are free or by donation.
- Calendar integration is `.ics` only — no Google/Apple OAuth.
- Auth is **email magic link / OTP**, no SMS, no password on day one.
- The AI assistant is a **search helper**, never a generator. It returns real event IDs only and must never invent events. Its prompts are tuned for curatorial intent ("какая практика мне сейчас нужна?"), not commercial intent ("что подешевле").
- Geography matters: events have city, district, metro, venue, coordinates. But venue type (музей, галерея, мастерская, независимое пространство, парк, онлайн) is its own first-class filter.

---

## 2. Tech stack and constraints

Build with this exact stack. Do not substitute.

- **Next.js 14+ App Router**, TypeScript, React Server Components where appropriate.
- **Tailwind CSS** for styling. Use only core utilities and the project's `tailwind.config.ts` tokens (defined below). No arbitrary CSS unless unavoidable.
- **shadcn/ui** (Radix-based) for primitives: Button, Input, Dialog, Sheet, Popover, Command, Tabs, Tooltip, Toast, Skeleton, DropdownMenu, Calendar, Form, Select, Switch, Checkbox, RadioGroup, Avatar, Badge.
- **TanStack Query** for client data, **Zod + React Hook Form** for forms.
- **lucide-react** for icons (no other icon library).
- **date-fns** with `ru` locale for dates and times.
- **Maps**: stub the map as a `<MapPlaceholder city venue lat lon />` component. Do not call a real provider; leave a clear integration seam.
- **Images**: use `next/image`. Source from `unsplash` or `picsum` as mock; treat S3 URLs as the real source in the data layer.

State and data:

- Routes use the App Router. Public pages are server components and statically rendered where possible.
- Mock data lives in `/lib/mock/*.ts` and is typed. Build a `MockApi` layer that the UI talks to so a real backend can replace it without UI changes.
- Use suspense + skeletons. Every fetch path has a loading state and an empty state.

---

## 3. Visual direction — Luma-like

The brief is **minimal, white, type-driven**. Beautiful by restraint. The product should feel premium without looking corporate.

### 3.1 Color tokens (Tailwind theme)

```ts
// tailwind.config.ts (excerpt)
colors: {
  bg:        '#FFFFFF',           // page background
  surface:   '#FAFAF7',           // cards on hover, secondary surfaces
  ink:       '#0A0A0A',           // primary text
  ink2:      '#3F3F46',           // secondary text
  muted:     '#8A8A8E',           // tertiary text, captions
  hairline:  '#ECECEC',           // borders, dividers
  accent:    '#111111',           // primary action (deliberately near-black)
  accentInk: '#FFFFFF',           // text on accent
  success:   '#16A34A',
  warning:   '#D97706',
  danger:    '#DC2626',
  // single optional warm tint, used sparingly
  blush:     '#F5EDE6',
}
```

Dark mode is **out of scope for MVP**.

### 3.2 Typography

- Display + UI: `Inter` (Latin + Cyrillic), with `font-feature-settings: "ss01","cv11"`.
- Optional accent serif for hero headlines: `PT Serif` or `Source Serif 4`. Use sparingly.
- Type scale (Tailwind):
  - `text-5xl` (48/56) — hero
  - `text-4xl` (36/44) — page H1
  - `text-2xl` (24/32) — section H2
  - `text-xl` (20/28) — card title
  - `text-base` (16/24) — body
  - `text-sm` (14/20) — meta
  - `text-xs` (12/16) — caption, badges
- Weights: 400 body, 500 UI, 600 titles. Never use 700+ for body.

### 3.3 Layout system

- Max content width: `max-w-6xl` (1152px) for most pages; `max-w-3xl` (768px) for forms and reading.
- Page padding: `px-5 md:px-8` with `mx-auto`.
- Vertical rhythm: section spacing `space-y-12 md:space-y-16`.
- Cards: `rounded-2xl border border-hairline bg-bg hover:bg-surface transition`.
- Buttons: `rounded-full` for primary CTAs, `rounded-lg` for icon and secondary buttons.
- Shadows: use sparingly. Prefer `border-hairline` over `shadow-*`. Allow `shadow-sm` on floating popovers only.
- Images: rounded `rounded-xl`, never circular except avatars.

### 3.4 Motion

- Use `transition-colors`, `transition-opacity`, `transition-transform` with `duration-150` to `duration-200`.
- Page transitions are subtle: fade-in only.
- Avoid bouncy spring animations. No parallax. No marquees.

---

## 4. Information architecture

### 4.1 Public + user routes

```
/                          — Home / discovery (state-aware: anonymous vs. signed-in)
/events                    — Browse with filters + search
/events/[slug]             — Event detail (public; signed-in users see sign-up state)
/events/ai                 — AI conversational search (3 free queries for anonymous, then prompt)
/organizers/[slug]         — Public organizer page
/hosts/[slug]              — Public host page (mediator / artist / curator)

/auth/sign-in              — Email entry
/auth/verify               — Magic link / OTP entry
/auth/callback             — Token exchange landing
/auth/onboarding           — First-time onboarding (3 steps, skippable)

/me                        — Personal home: next practice + applications + saved   (auth)
/me/practices              — Upcoming + past sign-ups                              (auth)
/me/applications           — Pending and historical applications                   (auth)
/me/saved                  — Saved practices                                       (auth)
/me/follows                — Followed organizers and hosts                         (auth)
/me/notifications          — Inbox                                                 (auth)
/me/interests              — Edit topics, districts, venues                        (auth)
/me/settings               — Account, privacy, email preferences                   (auth)
/me/profile                — Public profile preview + edit                         (auth)

/about                     — About the product
/legal/privacy             — Privacy policy
/legal/terms               — Terms of service
/legal/consent             — PD consent text
```

For backwards compatibility a `/saved` and `/profile` redirect to `/me/saved` and `/me/profile`.

### 4.2 Organizer routes (under `/o`)

```
/o                        — Organizer dashboard (org switcher if multi-org)
/o/sign-up                — Organizer onboarding
/o/profile                — Organizer profile (public-facing preview + edit)
/o/events                 — Organizer's events list, filtered by status
/o/events/new             — Create event (multi-step)
/o/events/[id]            — Event detail in organizer view
/o/events/[id]/edit       — Edit event
/o/events/[id]/stats      — Event stats
/o/members                — Organizer team (membership)
/o/settings               — Notification + account settings
```

### 4.3 Admin routes (under `/admin`)

```
/admin                          — Admin home: queues + alerts
/admin/moderation/events        — Event moderation queue
/admin/moderation/organizers    — Organizer verification queue
/admin/complaints               — Complaints inbox
/admin/categories               — Categories CRUD
/admin/interests                — Interests CRUD
/admin/featured                 — Manual featured events
/admin/users                    — User search and detail
/admin/organizers               — Organizer search and detail
```

---

## 5. Page specifications

For every page below, deliver:

1. A typed Next.js route file under `app/...`.
2. Page-specific components under `app/.../_components/`.
3. Shared components under `components/`.
4. Loading skeleton (`loading.tsx`), error boundary (`error.tsx`), and empty state.
5. Working with mock data from `lib/mock/*` via the `MockApi`.

### 5.1 Home `/`

**Purpose:** make the visitor want to click into an event within 5 seconds.

Sections, top to bottom:

1. **Header** (see §6.1).
2. **Hero**: large headline ("Практики, разговоры и встречи в Москве" — or similar curatorial register), one-line subtitle ("Медиации, мастерские, обсуждения и встречи с художниками"), search input that opens command-style search, three suggested chips ("на эти выходные", "медиации", "встречи с художниками"). No carousel.
3. **Curated rows** (each a horizontal scroll on mobile, 3-up grid on desktop):
   - "Скоро" — events starting in the next 72 hours.
   - "На выходные" — Sat-Sun.
   - "Медиации по выставкам" — events of type `mediation`.
   - "Мастерские и практики" — `workshop`.
   - "Разговоры и обсуждения" — `discussion` + `open_lecture`.
   - "Встречи с художниками" — `artist_talk` + `studio_visit`.
   - "Свободный вход" — free or by donation.
   - "По интересам" — personalized once the participant has interests; hidden otherwise.
4. **AI helper card**: a single wide card. Copy: "Опишите, какой опыт сейчас нужен — подберём практику." CTA opens `/events/ai`.
5. **Editorial strip** (optional, P1): a single line linking to a curator's pick or a partner institution.
6. **Footer** (see §6.2).

Empty/zero state: if there are no events for a row, hide the row. Never render an empty rail.

### 5.2 Browse `/events`

**Purpose:** filter and find events.

Layout: two-pane on desktop (filters left 280px, results right), single-pane on mobile with a "Фильтры" bottom sheet.

Filters (in this order — order matters for a curatorial product):

- **Тип практики** (multi-select chips, most important filter, shown first):
  `Медиация`, `Мастер-класс / мастерская`, `Обсуждение`, `Открытая лекция`, `Встреча с художником`, `Студийный визит`, `Перформативная практика`, `Читательская группа`.
- **Дата**: chips for "Сегодня", "Завтра", "Выходные", "Эта неделя", "В этом месяце"; custom range via Calendar.
- **Темы / интересы** (multi-select tags): современное искусство, кино, литература, тело и движение, память и архив, феминизм, экология, звук, керамика, текст, и т.д. — from `/lib/mock/topics.ts`.
- **Тип площадки**: музей, галерея, мастерская, независимое пространство, парк/улица, онлайн.
- **Район / метро**: combobox with grouping.
- **Уровень участия**: для всех, требуется подготовка, требуется собеседование с куратором.
- **Размер группы**: «камерная (до 10)», «средняя (10–25)», «большая (25+)».
- **Язык**: русский, английский, без языка / невербальная практика.
- **Стоимость**: "Свободный вход", "Донат", "До 1000 ₽", "1000–3000 ₽", "Свыше 3000 ₽".
- **Доступность**: без ступеней, тифлокомментарий, сурдоперевод, сенсорно-дружественная среда — multi-select.
- **Сортировка**: "Скоро", "Только что добавили", "Малая группа", "Близко".

Do not surface "Популярные" as a sort on day one — it pushes a logic of attention metrics that contradicts the curatorial framing.

Results:

- Toggle: **List** (default) vs **Map**. Map shows pins with hover preview. Use `MapPlaceholder`.
- Result count and active filter pill row above the grid, with a "Сбросить" link.
- Cards (see §6.3) in a 1/2/3-column responsive grid.
- Infinite scroll with sentinel + skeleton; "Загрузить ещё" button as fallback.

URL state: every filter and sort lives in the query string. Back/forward must restore exact view.

### 5.3 Event detail `/events/[slug]`

**Purpose:** convince + convert. Server-rendered for SEO.

Hierarchy:

1. **Cover** — full-bleed image, `aspect-[16/9]` desktop, `aspect-[4/5]` mobile. Above it, a back link. A small chip on the cover indicates the **practice type** ("Медиация", "Мастерская", "Беседа" и т.д.).
2. **Title block**: `H1` title (`text-4xl font-semibold`), one-line meta row: date · time · площадка · район. Just below, a single sentence subtitle ("О чём эта встреча") — pulled from a dedicated `lede` field, not from the description.
3. **Primary actions** (sticky on mobile bottom). The CTA copy adapts to the sign-up mode:
   - If `signup_mode = open`: **"Записаться"** — primary button. After confirmation, copy switches to "Вы записаны".
   - If `signup_mode = application`: **"Подать заявку"** — opens a sheet with the curator's question.
   - If `signup_mode = external`: **"Записаться на сайте организатора"** — opens `external_registration_url` in a new tab with a small caption "Запись ведёт организатор".
   - **Сохранить** — bookmark toggle.
   - **Поделиться** — share sheet.
   - **В календарь** — `.ics` download.
   - When `seats_remaining = 0` and waitlist is on: CTA switches to **"В лист ожидания"**.
4. **Two-column body** on desktop, stacked on mobile:
   - **Left** — narrative column (`max-w-prose`):
     - **Описание практики** (rich text). Encourage organizers to write in second person ("вы будете…").
     - **Что мы будем делать** (optional structured block / agenda).
     - **Ведущий / медиатор / художник** — a card with portrait, name, role chip ("медиатор", "художник", "куратор", "ведущий"), short bio, and link to their other practices.
     - **Кому подойдёт** (optional) — short list: "людям без художественного образования", "тем, кто работает с архивами", и т.д. Sets expectations.
     - **Что взять с собой** (optional) — for workshops: материалы, удобная одежда, ноутбук.
     - **Подготовка** (optional) — recommended reading / viewing before the session.
     - **Доступность** — accessibility notes, written explicitly (это важно для аудитории).
     - **Темы и теги** — chips linking to filtered browse.
     - **Пожаловаться** — discreet link at the bottom.
   - **Right column, sticky**:
     - **Когда** — date, time, продолжительность (e.g., "2 часа").
     - **Где** — venue name, address, район, метро, map preview. If online: link with "присоединиться придёт письмом".
     - **Группа** — capacity progress bar: "осталось 7 из 15 мест". When small group: caption "Камерный формат".
     - **Язык** practice — RU / EN / без языка.
     - **Стоимость** — "Свободный вход" / "Донат" / точная цена. If donation: short "сколько считаете возможным" note.
     - **Возраст** — only if relevant (18+, 12+).
     - **Организатор** — name, avatar, link to organizer page, "ещё практики".
5. **Similar practices** rail at the bottom — same type or same topics, never just "popular nearby".

States:

- `pending_review` / `draft`: a top banner "Это превью. Событие ещё не опубликовано." Only visible to the organizer/admin.
- `cancelled`: striked title, red banner "Событие отменено".
- Past event: muted, no RSVP button, "Событие уже прошло".

### 5.4 AI conversational search `/events/ai`

**Purpose:** natural-language discovery.

Layout: full-height single column, `max-w-3xl mx-auto`.

- Prominent input at top initially; after first message, input moves to bottom (Claude/ChatGPT pattern).
- Each AI turn renders:
  - Short natural-language reply ("Нашёл 4 варианта на субботу").
  - A **horizontal row of event cards** (compact variant) — these are real events fetched from backend by ID.
  - Optional follow-up chips ("показать ещё", "дешевле", "ближе к центру").
- Below the input, a tiny notice: "ИИ-помощник ищет только по реальным событиям и может ошибаться." Required for trust.

Rate-limit and empty-result UX:

- If the limit is hit: friendly card "Попробуйте через минуту".
- If no results: "Ничего не нашёл по этому запросу. Попробуйте смягчить фильтры." with two recovery chips.

Hard rule: the UI must render zero free-text from the model that looks like an event description. Only real cards.

### 5.5 Auth — `/auth/sign-in`, `/auth/verify`, `/auth/callback`

**Sign-in**: single centered card, `max-w-md`. Email input, primary button "Получить ссылку". Below: "Продолжая, вы соглашаетесь с условиями и политикой ПДн." with links.

**Verify**: if magic link was sent, show "Проверьте почту" with the masked email and a "Отправить снова" link (60-second countdown). If OTP, show a 6-digit input.

**Callback**: minimal loading state; on success redirect to `next` query param or `/`.

No social login on day one. No password. No phone.

### 5.6 Personal area — `/me` and its children

The personal area lives under `/me/*` (see §5.14.8 for the full per-page spec of `/me`, `/me/practices`, `/me/applications`, `/me/saved`, `/me/follows`, `/me/notifications`, `/me/interests`, `/me/settings`, `/me/profile`). All routes are auth-only and redirect to sign-in with the original path preserved as `?next=`.

This section is intentionally short — the substantive design lives in §5.14, because the personal area is the embodiment of the post-login experience and must be designed alongside the anonymous/authenticated state model.

### 5.7 Organizer dashboard `/o`

**Purpose:** focused operations console for an organizer.

Top of page: organization switcher (if user belongs to multiple) and a "Создать событие" primary button.

Cards row:

- Active events count.
- Pending moderation count.
- This-month views.
- This-month RSVPs.

Sections:

- **Требует внимания**: events rejected with a reason, or with new complaints.
- **Ближайшие**: events in the next 14 days, with quick links to stats.
- **Черновики**: drafts.

### 5.8 Organizer event create `/o/events/new`

**Purpose:** make event creation feel light but complete.

Multi-step form, single page with a sticky progress rail on the left:

1. **Тип практики**: required first. Radio cards for `Медиация / Мастер-класс / Обсуждение / Открытая лекция / Встреча с художником / Студийный визит / Перформативная практика / Читательская группа`. Each card has a one-line hint explaining what it implies for the participant. Choosing a type can subtly adjust later steps (e.g., "Мастер-класс" shows the "Что взять с собой" field; "Медиация" shows "Какая выставка").
2. **Основное**: title, short subtitle ("О чём встреча в одном предложении" — used as `lede`), full description (rich text, mock toolbar). Encouraging placeholder copy that nudges second-person, participatory framing.
3. **Ведущий**: select existing person from organizer's roster or add new. Fields: имя, роль (медиатор / художник / куратор / ведущий), короткая био, фото, ссылки. Multiple hosts supported.
4. **Когда**: date, start time, duration (in hours/minutes, not end time — durations feel more honest for practices), timezone (default `Europe/Moscow`), recurring stub ("позже").
5. **Где**: venue picker — combobox over mock venues; if no match, "Добавить новое место" inline form (name, address with `AddressInput` stub for DaData, район, метро, lat/lon, тип площадки). Online / hybrid toggle reveals a URL field plus the note "ссылка придёт записавшимся по почте".
6. **Группа и запись**:
   - **Размер группы** (capacity): integer, with a soft warning if > 30 ("камерный формат — одна из ценностей платформы").
   - **Формат записи**: радио "Открытая запись / По заявке с вопросом куратора / Запись на сайте организатора".
   - If "По заявке": text field for the curator's question ("Расскажите, почему вам интересна эта практика").
   - If "Запись на сайте организатора": `external_registration_url`.
   - **Лист ожидания**: toggle (default on).
7. **Стоимость**: радио "Свободный вход / Донат / Платно". If "Донат": optional suggested amount. If "Платно": price (one number, not a range; ranges add cognitive load to this product) + `external_ticket_url` if applicable.
8. **Для кого и что взять**: optional fields — "Кому подойдёт", "Что взять с собой", "Подготовка (что прочитать/посмотреть)".
9. **Язык и доступность**: язык практики (RU/EN/без языка), возрастное ограничение (только если нужно), доступность (multi-select: без ступеней, тифлокомментарий, сурдоперевод, сенсорно-дружественная среда + free text).
10. **Обложка и медиа**: dropzone (mock); recommended ratio 16:9; max 5 MB; suggestion: "лучше работают спокойные изображения процесса, чем рекламные кадры".
11. **Темы и теги**: пресет тем + свободные теги.
12. **Публикация**: summary card + two buttons "Сохранить в черновик" / "Отправить на модерацию". Submit copy explains the curatorial bar in one sentence.

Validation with Zod, inline errors, autosave to draft every 10 s with a small "Сохранено" toast.

### 5.9 Organizer event edit and stats

**Edit `/o/events/[id]/edit`**: same form, pre-filled. Status banner at top showing current state and what edits do (e.g., "Изменения в опубликованном событии отправят его повторно на модерацию").

**Stats `/o/events/[id]/stats`**: four KPI tiles (views, saves, RSVPs, conversion), a 30-day line chart (`<MockChart>` placeholder), traffic source table, and an RSVP list with CSV export.

### 5.10 Organizer profile, members, settings

**Profile `/o/profile`**: split view — left edit form (name, description, website, social links, avatar, cover), right live preview of public organizer page.

**Members `/o/members`**: table of members with roles (`organizer_admin`, `organizer_member`), invite by email modal.

**Settings `/o/settings`**: notification preferences (transactional always on, marketing toggle), legal entity info stub, time zone.

### 5.11 Admin home `/admin`

**Purpose:** triage. The admin should know within 3 seconds where to click first.

Top: a row of queue tiles with counts:

- События на модерации
- Организаторы на проверке
- Жалобы
- Возможный спам

Below: a feed of recent admin actions (audit log), and quick filters by city and category.

### 5.12 Admin event moderation `/admin/moderation/events`

Layout: left column is the queue (compact list with title, organizer, submitted_at, category), right column is the selected event preview with action panel:

- **Одобрить** (primary)
- **Отклонить** (opens modal with reason templates + custom text)
- **Запросить правки** (opens modal with editable note)
- **Эскалировать**
- Keyboard shortcuts: `J/K` to navigate, `A` approve, `R` reject. Show a small "?" with shortcuts.

Reasons for rejection should be a curated list (templates) plus free text, so audit and analytics work.

### 5.13 Admin organizer moderation, complaints, categories, interests, featured, users, organizers

For each, build the same general shell: searchable, sortable, paginated table on the left; detail panel on the right; primary actions surfaced. Don't reinvent layout per page.

- **Complaints**: each row links to the offending event/organizer/user with quick actions: dismiss, warn, suspend.
- **Categories** and **Interests**: simple CRUD with slug, name (RU + EN), icon picker (lucide names).
- **Featured**: drag-handle list of currently featured events with start/end dates; an "Добавить" combobox.
- **Users / Organizers**: search, profile detail, suspension reason, audit log.

### 5.14 Anonymous and authenticated user — cross-cutting behavior

This is the most important UX layer. **Login is a moment of intent, not a gate.** The product must be deeply browsable without an account; sign-in happens only when the participant wants to do something personal (sign up to a practice, save, apply, subscribe, get reminders). Build every page with both states in mind from day one.

#### 5.14.1 What anonymous visitors can do — full list

- Browse the entire catalog with all filters and map view.
- Read every event/practice page in full (cover, description, host card, when/where, capacity, accessibility, organizer link).
- Read every organizer and host page.
- Use the global search.
- Use the AI search assistant, **rate-limited to 3 queries per device per day** (cookie + IP). After that, soft prompt to sign in.
- Add a practice to calendar via `.ics` download — no login required.
- Subscribe to **a single event's email reminder** by entering email inline (no account created; one-shot transactional with explicit consent).
- Share an event via the share sheet.
- Read all static pages (about, legal).

Anonymous users **cannot**: sign up to a practice, submit an application, save, follow an organizer or host, receive ongoing notifications, see personalized rows, see attendance history, edit a profile.

#### 5.14.2 Login trigger pattern — intent-preserving

When an anonymous visitor clicks any gated action, **never just redirect to `/auth/sign-in` and lose intent**. Instead:

1. A `Sheet` (bottom on mobile, side on desktop) opens with: a short headline scoped to the action — e.g. "Войдите, чтобы записаться", "Войдите, чтобы сохранить" — and a single email input + "Получить ссылку".
2. Beneath the button: "Письмо со ссылкой придёт за минуту. Маркетинговых писем не будет — только то, на что вы согласитесь."
3. The intent is encoded in the magic link callback URL as `?next=<path>&intent=<rsvp|apply|save|follow|waitlist>&entity_id=<id>`.
4. After verification, the user lands back on the originating page and the **intended action is auto-executed** (RSVP confirmed, application sheet pre-opened with cursor in the answer field, save toggled on, follow active).
5. A small toast confirms the result.

This pattern is non-negotiable. Apply it to every gated CTA on every page.

#### 5.14.3 What anonymous users see on each public page

**Home `/`:**
- Editor-curated practice rows visible (see §5.1).
- The "По интересам" rail is **hidden**.
- Hero CTA invites a search query or AI prompt, never a login.
- A single low-key bar after the third row (dismissable): "Сохраняйте практики, чтобы вернуться к ним — войдите за минуту." Bar is hidden permanently once dismissed.

**Browse `/events`:**
- All filters work, except "По интересам" filter chip is hidden.
- Save icon on cards triggers the login sheet with `intent=save`.
- No personalization in sort order; default sort is "Скоро".

**Event detail `/events/[slug]`:**
- Full content visible.
- Primary CTA ("Записаться" / "Подать заявку" / "В лист ожидания") triggers the login sheet with the corresponding intent.
- "В календарь" works without login (direct `.ics` download).
- **Subtle alternative for reminder-only intent**: below the calendar button, a thin link "Прислать напоминание на почту" opens a tiny inline form (email + consent checkbox) that subscribes to *this single event's* reminders without creating an account. Useful for low-commitment intent.
- The right column shows capacity as informational, no "вы записаны" block (no account yet).

**AI search `/events/ai`:**
- First 3 queries per device per day are free.
- 4th query: input is disabled and a soft card replaces it — "Войдите, чтобы продолжить разговор. Мы запомним ваши предпочтения и не покажем те же практики дважды." Sign-in sheet opens on click.
- Anonymous queries do not persist across sessions; conversation resets on page reload.

**Organizer and host pages:**
- Fully readable.
- "Подписаться" triggers login sheet with `intent=follow`.

#### 5.14.4 First-time onboarding `/auth/onboarding`

Triggered automatically after magic link verification for a new account. Three short steps, all skippable, with a progress dot row at the top:

1. **Имя** — display name, optional avatar upload. Pre-filled from email local-part as a soft suggestion.
2. **Темы, которые сейчас интересны** — multi-select from 12–15 curated topics (современное искусство, кино, литература, тело и движение, память и архив, феминизм, экология, звук, керамика, текст, городская среда, перформанс, кураторство, философия). Minimum 0, recommended 3–5. Used for the "По интересам" rail and AI personalization.
3. **Где удобно встречаться** — multi-select districts and venue types (опционально). Bias for discovery, not a filter.

Final screen: "Готово. Загляните в подборки, собранные под ваши интересы." with a "Перейти на главную" CTA. The user can re-open or change all of this later under `/me/interests`.

A user who skips everything still gets a fully functional account; onboarding is a recommendation, not a wall.

#### 5.14.5 What changes for authenticated users — header

- "Войти" replaced by avatar dropdown with: Моё, Мои практики, Сохранённое, Подписки, Уведомления, Настройки, Выйти. If the user has organizer role: "Кабинет организатора". If admin: "Админ-панель".
- A **notifications bell** appears to the left of the avatar with an unread count badge. Click opens a popover with the 5 most recent items + "Все уведомления →".

#### 5.14.6 Authenticated home `/` — personalized order

Above the editor-curated rows from §5.1, signed-in users see (each section hidden if empty — no zero-state placeholders):

1. **"Ваше следующее"** — single wide hero card with the user's next confirmed practice. Shows time-until ("через 3 дня"), venue, host. Two CTAs: "Подробнее" and "В календарь".
2. **"Заявки и ожидание"** — chip row of pending applications and waitlist positions, one chip per item, with state ("ждёт ответа", "вы 3-й в листе"). Click → `/me/applications` filtered.
3. **"По интересам"** — practices matching topics from onboarding.
4. **"От ваших организаторов"** — new practices from followed orgs/hosts since last visit.
5. **"Похожее на сохранённое"** — content-based similarity over saved practices (same type, overlapping topics). Never opaque ML; the "почему" must be explainable in one sentence on hover.

Below personalized rows, the standard editor-curated rows from §5.1 follow.

#### 5.14.7 Event detail — authenticated state additions

The primary CTA reflects the participant's current relation to the practice:

- Not yet signed up: **Записаться** / **Подать заявку** / **В лист ожидания** as defined in §5.3.
- Signed up (open mode): the CTA area becomes a confirmation block: "Вы записаны" with a secondary link "Отменить запись" (opens confirm Dialog).
- Application sent: "Заявка отправлена 12 мая. Куратор ответит в течение 3 дней." Secondary: "Изменить ответ" (if still pending) or "Отозвать заявку".
- Application accepted: "Заявка принята. До встречи 12 июня." + "Снять запись" link.
- Application declined: "К сожалению, в этот раз не получилось. Часто открываются похожие практики — посмотрите →".
- On waitlist: "Вы 3-й в листе ожидания. Если место освободится, мы напишем." + "Покинуть лист".

The save icon reflects state (filled vs. outline). On organizer pages there is also a follow toggle.

#### 5.14.8 New auth-only pages

**`/me` — личная главная.** A focused dashboard: hero card with next practice, three tiles (Заявки / Сохранённое / Подписки) with counts, a compact list of upcoming practices, and a link to the full catalog. No marketing copy.

**`/me/practices` — мои практики.** Tabs: "Предстоящие", "Прошедшие". Each row shows date, title, venue, status (записан / в листе ожидания), and a contextual action. Past tab adds an optional one-tap qualitative reflection: "Как прошло?" with three buttons (резонансно / нейтрально / не пошло). The signal is private and only feeds future recommendations — never shown publicly, never as a 5-star rating.

**`/me/applications` — мои заявки.** Tabs: "В ожидании", "Принятые", "Отклонённые", "Отозванные". Each card shows the curator's question, the user's answer (expandable), and the status with timestamp. From "В ожидании" the user can edit the answer or withdraw.

**`/me/saved` — сохранённое.** Grid identical to `/events`. Default sort by date saved, descending. Bulk-unsave from a long-press / select mode (P1).

**`/me/follows` — подписки.** Two tabs: "Организаторы", "Ведущие и художники". Each card shows the entity, last new practice date, and unsubscribe action. Empty state suggests three highlighted organizers/hosts.

**`/me/notifications` — уведомления.** Single chronological feed. Filterable by type: "Записи", "Заявки", "Напоминания", "Подписки", "Системные". Each notification is one short line + secondary line, with a direct link. Bulk "отметить всё прочитанным". Notifications are also delivered by email per the user's settings.

Notification types:

```
application_accepted        — ваша заявка принята
application_declined        — ваша заявка отклонена
application_needs_info      — куратор просит уточнить
waitlist_promoted           — вы прошли из листа ожидания
event_reminder_24h          — напоминание за сутки
event_reminder_2h           — напоминание за 2 часа
event_updated               — изменились детали (время / место / онлайн-ссылка)
event_cancelled             — событие отменено
followed_org_new_practice   — новая практика у организатора, на которого вы подписаны (digest, not per-event)
followed_host_new_practice  — новая практика у ведущего (digest)
system                      — об аккаунте, безопасности, изменениях в политике
```

**`/me/interests` — темы и районы.** Same controls as onboarding step 2 and 3, editable anytime. A small note: "Эти настройки влияют только на подборки. Мы не показываем их организаторам."

**`/me/settings` — настройки.** Sections:

- **Аккаунт**: email (с возможностью смены через подтверждение), display name.
- **Уведомления**: matrix of notification type × channel (in-app / email). Transactional rows (reminders, application_*, event_updated/cancelled, waitlist_promoted) cannot be turned off entirely — only channel. Digest rows can be off. Marketing digest is off by default and requires explicit opt-in.
- **Приватность**: видимость моего участия для организатора (нужна некоторым практикам по умолчанию), хранение истории поиска ИИ (вкл/выкл).
- **Данные**: "Скачать мои данные" (stub), "Удалить аккаунт" (Dialog explains what is erased and what is retained for legal reasons).
- **Безопасность**: список активных сессий, "выйти со всех устройств".

**`/me/profile` — публичный профиль.** On day one this is a minimal public preview: display name, avatar, optional one-line bio, topics (if user opted to make them public — default off). Used mainly as the canonical "you" page in case the product later adds social features. Most participants will never need to look at it.

#### 5.14.9 Soft prompts and never-prompts

Acceptable soft prompts for anonymous users (each dismissable, each shown at most once per week per device):

- After viewing 5+ event pages in a session: bar "Сохраняйте практики, чтобы вернуться".
- After the 3rd AI query: card "Войдите, чтобы продолжить разговор".

Never do:

- Modal walls of any kind on first visit.
- "Sign in to read more" gates on event content.
- Banner ads of any kind.
- Email-collection popups disconnected from a specific intent.
- "Hot now" / urgency timers / "X people are looking at this".

#### 5.14.10 Sign-out

From avatar menu → "Выйти". Confirms in a `Popover`, not a `Dialog` — it's not destructive. Clears tokens, redirects to `/`. On organizer routes with unsaved drafts, intercept with a guard.

#### 5.14.11 Implementation hints

- Authentication state lives in a server-readable cookie + a thin client store. Server components render the correct state without flicker — no client-only "loading user…" placeholders for the header.
- Every gated CTA is a `<SignInGate intent="…" entityId="…">` wrapper that renders the action button and handles the sheet flow.
- Personalized rails on `/` are server components that take `userId` as input and call `MockApi.getPersonalRails(userId)`. For anonymous, they return `[]` and the component renders nothing.
- The notification bell polls `MockApi.getUnreadNotifications()` every 60 s when the tab is visible.

---

## 6. Shared components and system

### 6.1 Header

- Sticky, transparent on home hero (turns solid on scroll), solid elsewhere.
- Left: word-mark only, no icon mark.
- Middle (desktop): search input that opens a `Command` palette with recent searches and practice types.
- Right, anonymous: city picker (default Москва), "Войти" button. Clicking "Войти" opens the inline sign-in sheet (no full-page redirect unless the user explicitly chose `/auth/sign-in`).
- Right, authenticated: city picker, **notifications bell with unread count badge** (popover shows 5 latest + link to `/me/notifications`), avatar dropdown with: **Моё** (`/me`), **Мои практики** (`/me/practices`), **Сохранённое** (`/me/saved`), **Подписки** (`/me/follows`), **Настройки** (`/me/settings`), **Выйти**. If user has organizer role, dropdown also shows "Кабинет организатора" → `/o`. If admin, "Админ-панель" → `/admin`.
- Mobile: hamburger opens a `Sheet` with the same items, in the same order, stacked vertically.
- The header must render the correct state on first paint (no avatar flicker). Use a server-readable auth cookie.

### 6.2 Footer

Three columns on desktop, stacked on mobile: product links, legal links, contacts and social. Bottom row: copyright, language switcher (RU/EN stub), small "ИИ-функции — beta" note.

### 6.3 Event card

Two variants:

- **Default** (used in grids): cover 16:9, **practice-type chip** in the top-left of the cover ("Медиация", "Мастерская", и т.д.), title (clamp 2 lines), one-line meta: дата · площадка · район. A second meta line is reserved for **seats remaining** when capacity is small: "осталось 3 из 12 мест". Save icon button at top-right of cover. Entire card is a link. Hover lifts background to `surface`.
- **Compact** (used in AI chat, rails): square or 4:3 cover, title clamp 1 line, single meta line, smaller padding. Practice-type chip still present.

Card includes (subtle, not loud):

- Status badge if not `published` (visible only to organizer/admin).
- "Свободный вход" or "Донат" label if applicable.
- "Камерный формат" caption when capacity ≤ 12.
- "Осталось мало мест" caption when ≤ 30% capacity remains.
- "По заявке" caption when `signup_mode = application`.
- "Скоро начнётся" badge when event starts within 6 hours.

Never combine more than two of these on a single card — pick the most informative.

### 6.4 Form primitives

Use shadcn `Form` + RHF + Zod. Show inline errors below fields. Long forms (`/o/events/new`, `/o/events/[id]/edit`) use a left progress rail.

### 6.5 Empty, loading, error states

Every list and detail page has all three:

- **Loading**: `Skeleton` blocks matching final layout.
- **Empty**: one-liner explanation + one CTA. No illustrations on MVP.
- **Error**: short copy + "Попробовать снова" button. Log to console for now.

### 6.6 Toasts and dialogs

- Use `Toast` for non-blocking confirmations ("Событие сохранено в избранное").
- Use `Dialog` for destructive actions (delete event, cancel RSVP).
- Use `Sheet` on mobile for filters and menus.

### 6.7 Date and price formatting

- Dates: `date-fns/format` with `ru` locale. Long: `5 июня, пт · 19:00`. Short: `5 июн`.
- Time ranges: `19:00 – 22:00`.
- Prices: `Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', maximumFractionDigits: 0 })`. Free → "Бесплатно". Range → "от 500 ₽" or "500 – 2 000 ₽".

---

## 7. Responsive rules

- Breakpoints: Tailwind defaults. Design for `sm` (≥640), `md` (≥768), `lg` (≥1024), `xl` (≥1280).
- Mobile first. Every page must be usable at 360 × 640.
- Primary CTAs on key conversion pages (event detail, RSVP, sign-in) are sticky on mobile.
- Filters are inline on desktop, bottom-sheet on mobile.
- Tables on admin pages must degrade gracefully: convert to stacked cards below `md`.

---

## 8. Accessibility (WCAG 2.1 AA)

- All interactive elements have visible focus rings (`focus-visible:ring-2 ring-ink/20`).
- Color contrast: body text `ink` on `bg` is fine; never use `muted` for body text, only meta.
- Buttons and links have accessible names; icon-only buttons have `aria-label`.
- Forms use `Label` + `aria-describedby` for help text and errors.
- Toasts are announced via `aria-live="polite"`.
- Keyboard navigation works fully on the moderation queue (J/K/A/R) and the AI search.
- Skip-to-content link in the header.
- Tap targets ≥ 44 × 44 px on mobile.

---

## 9. SEO and metadata

- `app/events/[slug]/page.tsx` exports `generateMetadata` with title, description, OG image (event cover), `og:type=event`, JSON-LD `Event` schema.
- Home and category pages export static metadata.
- Sitemap stub at `/sitemap.xml`, robots at `/robots.txt`.
- Use `next/link` everywhere internal. No client-side `window.location` navigation.

---

## 10. Deliverables

Create a single repo with this structure:

```
event-discovery-web/
├─ app/
│  ├─ (public)/                # public + user routes (group)
│  ├─ o/                       # organizer cabinet
│  ├─ admin/                   # admin
│  ├─ api/mock/                # mock API route handlers (optional)
│  ├─ layout.tsx
│  ├─ globals.css
│  └─ not-found.tsx
├─ components/                 # shared components
│  ├─ ui/                      # shadcn primitives
│  ├─ event/                   # EventCard, EventList, EventFilters
│  ├─ organizer/
│  ├─ admin/
│  ├─ map/MapPlaceholder.tsx
│  └─ layout/Header.tsx, Footer.tsx
├─ lib/
│  ├─ mock/                    # typed mock data
│  ├─ api/MockApi.ts           # the seam to swap with real backend
│  ├─ format/date.ts, price.ts
│  ├─ schemas/                 # Zod schemas
│  └─ utils.ts
├─ types/                      # shared TS types matching the domain model
├─ public/
├─ tailwind.config.ts
├─ tsconfig.json
├─ package.json
├─ README.md
└─ DESIGN_NOTES.md             # design decisions, open questions, what was assumed
```

`README.md` must include: how to run, what's mocked, where to plug the real backend, and a map of routes → page files.

`DESIGN_NOTES.md` must include: any product decisions you made beyond this brief, visual rationale, accessibility checklist, known gaps.

---

## 11. Implementation checklist (the agent must self-verify before declaring done)

1. `pnpm dev` (or npm/yarn) starts the app with no console errors.
2. Every route listed in §4 renders with mock data.
3. Every page has loading, empty, and error states.
4. Lighthouse on the home page scores ≥ 90 on Performance, Accessibility, Best Practices, SEO.
5. All UI copy is in Russian. No English strings leak into the UI (placeholders included).
6. All interactive elements are keyboard-reachable with a visible focus ring.
7. URL state for filters survives refresh and back/forward.
8. Event detail page renders valid JSON-LD `Event`.
9. The AI search page renders only real event cards from mock data; no free-text "event-like" output from the assistant.
10. No real API keys, tokens, or secrets in the repo. Map and AI calls are stubbed.

---

## 12. Anti-patterns to avoid

- Do not add carousels with autoplay on the home page.
- Do not use modals for primary navigation.
- Do not inline payment forms — paid events link out only.
- Do not invite users to enter a phone number on day one.
- Do not surface admin-only fields (status, moderation reason) in user-facing views.
- Do not use stock illustrations or 3D blobs. Stay editorial and restrained.
- Do not use color to convey state without an accompanying icon or label.
- Do not implement Google/Apple calendar OAuth; `.ics` only.
- Do not paint dark mode.

**Anti-patterns specific to this product's curatorial framing:**

- Do not use commercial-marketplace language: never "купить билет", "лучшее предложение", "успей записаться", countdowns, "осталось всего 2 места!!" with red urgency. Capacity is informational, not a pressure tactic.
- Do not show attention metrics on user-facing surfaces — no view counts, no "X посмотрели сегодня", no "горячее", no "трендовое". Save these for organizer stats.
- Do not surface "Популярные" as a sort or filter on browse — popularity contradicts the curatorial premise.
- Do not call hosts "спикерами" or "артистами". The role chip must reflect the actual function: медиатор, художник, куратор, ведущий, фасилитатор.
- Do not refer to events as "товар", "продукт", "оффер". They are practices, встречи, разговоры.
- Do not put star ratings or reviews on events. If feedback ships later, it will be qualitative and curated, not 1–5 stars.
- Do not auto-suggest "events you might like" using opaque ML on day one — surface curatorial logic instead (same medium, same topic, same organizer).
- Do not let the AI assistant respond in marketplace tone ("вот топ-5 крутых событий!"). Its register must be calm and curatorial.

---

## 13. Open product questions the agent may resolve and document

The agent is permitted to make reasonable decisions on these, and must record them in `DESIGN_NOTES.md`:

- Whether sign-up requires authentication or supports a guest flow with email + `.ics` (for open events).
- For application-mode events: whether the curator's question is required or optional, and how the participant sees their application status (pending / accepted / declined).
- How "Сохранить" behaves before sign-in (anonymous save with upgrade banner vs. forced sign-in).
- How recurring practices (e.g., weekly reading group) are displayed on cards — as one card with "следующая встреча" or as multiple.
- How to visually distinguish curator-featured practices on the home — must be a subtle label, never a different card shape.
- Whether the city picker affects only the catalog or also the AI assistant context.
- Whether participant-facing pages show the host's other practices inline or only through a link.
- How accessibility tags are surfaced on cards (icon row vs. caption vs. hidden until detail).

---

## 14. Done means

A teammate can clone the repo, run it, click through every route in §4 with mock data, walk a stakeholder through the product flow end-to-end, and hand the codebase to backend engineers who can replace `MockApi` with real endpoints without touching component code.

Build it.
