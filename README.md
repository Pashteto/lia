# Lia — Event Discovery MVP

A web-first discovery platform for **participatory cultural practices** in a single Russian city at launch (Moscow), with an iOS app to follow. It is deliberately *not* a generic city afisha and *not* a ticketing marketplace — it is a curated home for events where the visitor is a **participant**, not a spectator (mediations, workshops, open lectures, reading groups, artist talks, performative practices).

## Audiences

- **Visitors & participants** — discover, filter, save, sign up, add events to calendar, and use an AI search assistant tuned for curatorial language.
- **Organizers** — museums, galleries, independent curators and artists manage profiles, create/edit events, and view basic stats.
- **Admins & moderators** — uphold the curatorial bar, review accounts and events, manage categories and tags.

## Repository layout

| Path | Description |
|------|-------------|
| `frontend/` | Web client (Next.js App Router + TypeScript + Tailwind v4 + pnpm). Discovery, event-detail, and create-event screens built against the API; AI-search stub. See `frontend/README.md`. |
| `backend/` | API and services (Go modular monolith from `go-microservice-template`, PostgreSQL + PostGIS). `events`, `categories`, and `venues` domains wired end-to-end; remaining domains are documented skeletons. See `backend/README.md`. |
| `design/` | Design system + reference screens. Active direction: **Apple HIG-based** (web-first, light/dark) — see `design/DESIGN.md`. `stitch_/` kept as historical archive. |
| `docs/` | Project documentation — technical stack and the design agent brief. |

### Documentation (`docs/`)

- `event_discovery_mvp_technical_stack.md` — technical architecture for the first version: recommended stack, MVP requirements, scaling path.
- `design_agent_prompt.md` — master design brief for an AI agent to design and implement the web app pages (Luma-like visual direction).

## Status

Frontend + backend are scaffolded with a working create → list → detail loop over the real API, plus **category** and **venue** normalization (both merged to `main`). Current state, how to run both sides, what's next, and known gotchas live in [`docs/HANDOFF.md`](docs/HANDOFF.md). Specs + implementation plans: [`docs/superpowers/`](docs/superpowers/).

## Notes

- Product UI copy is in **Russian**; code, comments, and file/component names stay in **English**.
- Documentation last actualized: 2026-06-14.
