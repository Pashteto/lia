# Archive — pre-Swiss-Grid design exploration

Historical design artefacts kept for provenance. **Not authoritative.**
Current spec lives in [`../design_handoff_presence_swiss_grid/README.md`](../design_handoff_presence_swiss_grid/README.md).

| File | What it is | Status |
|---|---|---|
| `Presence Design Review.dc.html` | Page-by-page critique of the pre-redesign site, mocked in the **Organic** system (rounded shapes, category colour ramps) | Superseded — Organic was dropped for Swiss Grid; its hygiene / card / AI-подбор findings shipped in Swiss Grid P1–P7 |
| `Presence Design Directions.dc.html` | Direction exploration, turns 1–4 (`1a`/`1b`/`2a`/`2b` Swiss variants of the event-detail screen, then photos + map) | Superseded — the chosen direction is in the canonical Full System mock |

`support.js`, `image-slot.js`, `map-embed.html` are symlinks into the canonical
handoff folder so these mocks stay runnable without duplicating assets:

```bash
cd docs/Redesign/5/archive && python3 -m http.server 8099
```
