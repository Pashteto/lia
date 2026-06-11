# Lia — Event Discovery MVP

A web-first discovery platform for **participatory cultural practices** in a single Russian city at launch (Moscow), with an iOS app to follow. It is deliberately *not* a generic city afisha and *not* a ticketing marketplace — it is a curated home for events where the visitor is a **participant**, not a spectator (mediations, workshops, open lectures, reading groups, artist talks, performative practices).

## Audiences

- **Visitors & participants** — discover, filter, save, sign up, add events to calendar, and use an AI search assistant tuned for curatorial language.
- **Organizers** — museums, galleries, independent curators and artists manage profiles, create/edit events, and view basic stats.
- **Admins & moderators** — uphold the curatorial bar, review accounts and events, manage categories and tags.

## Repository contents

| Path | Description |
|------|-------------|
| `event_discovery_mvp_technical_stack.md` | Technical architecture for the first version — recommended stack (Go modular monolith, PostgreSQL + PostGIS, Redis, S3), MVP requirements, scaling path. |
| `design_agent_prompt.md` | Master design brief for an AI agent to design and implement the web app pages (Next.js + TypeScript + Tailwind, Luma-like visual direction). |
| `stitch_/` | Design explorations and mockups — desktop variants, mobile, and a curatorial-minimalist direction (screenshots + generated HTML). |

## Notes

- Product UI copy is in **Russian**; code, comments, and file/component names stay in **English**.
- Documentation last actualized: 2026-05-24.
