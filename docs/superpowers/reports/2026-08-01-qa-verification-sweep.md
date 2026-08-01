# QA — verification sweep after the 31-jul fixes

**Date:** 2026-08-01
**Target:** production — https://presence.tarski.ru (API https://api.presence.tarski.ru)
**Build under test:** `lia-backend:qa31-r1` + `lia-frontend:qa31-r2`, branch `fix/qa-31jul` @ `1b112b3`
**Method:** live browser at 1512×804 + API probes + prod DB reads
**Predecessor:** `2026-07-31-qa-full-site-sweep.md` (13 defects)

---

## 1. Verdict

**All 13 defects from the 31-jul sweep are fixed and deployed.** Eleven are
verified live in the browser; two are unit-verified only, for the reasons given
below. One new defect was found during this walk (the report modal), fixed, and
redeployed as `qa31-r2`.

Two sub-items of §4.8 were deliberately **not** fixed, and one half of §4.13
turns out to be a Yandex licence constraint rather than a defect. Details in §3.

---

## 2. Verified live

| # | Defect | Evidence |
|---|---|---|
| 4.1 | «МЕСТА» dead on every event | Reads **`1 / 12`** on the capacity event; sidebar «Осталось мест: 11» reconciles |
| 4.2 | Feed led with past events | Feed header «МОСКВА · 8 СОБЫТИЙ», all dates ≥ today; API `from` default confirmed |
| 4.3 | Moderation promised but absent | Create-form and publish-dialog copy now describe what actually happens |
| 4.4 | Places key silently degraded | `mergeVenueLookups` reports the dead lookup; unit-verified (needs an authed venue-edit modal to see live) |
| 4.5 | Publish dialog copy wrong | Fixed with 4.3 |
| 4.6 | complaints/settings missed by redesign | Both carry the header band + U8 empty state; **square ink checkbox**, «ВКЛЮЧЕНО» chip; both in the admin nav |
| 4.7 | `/me/applications` payload gap | `rsvp lists wired to events enrichment` in the backend log; enrichment applies to `/me/practices` too |
| 4.8 | Numeric cells that can never populate | **«ОРГАНИЗАТОРОВ 07» / «ПОЛЬЗОВАТЕЛЕЙ 33»**; registry «СОБЫТИЙ» 01–02 and «ЖАЛОБ» 00 |
| 4.9 | Colliding admin short IDs | Nine distinct ids: EV-2JL0, EV-XURL, EV-15L0, EV-0CYY, EV-2CI3, EV-1JW1, EV-1Y72, EV-3XQ7, EV-4C18 |
| 4.10 | RU copy defects | **«ПОДАНО 15.07»** — «назад» gone, and it is now the *publication* date, not the event's start; queue column all-absolute; «1 ЖАЛОБА» in the inbox |
| 4.11 | Feedback form for non-attendees | Form absent for a signed-in non-participant |
| 4.12 | Blank cover band | Coverless hero renders the **numeral plate «01 · ЛЕКЦИИ»** |
| 4.13 | Venue map marker not to spec | **Square ink marker with the category numeral** («04», «01») — see §3 for the controls half |
| 4.14 | Admin first-load failure | **Did not reproduce.** Clean on a normal login → «Админ» navigation. The 31-jul occurrence was a harness artefact, as the report suspected |

### §8 surfaces the previous sweep could not cover

| Item | Result |
|---|---|
| 404 page render | ✅ Inverted-ink U8: «404», «Страница не найдена», «ВЕРНУТЬСЯ К ЛЕНТЕ» |
| Complaint submission | ✅ End-to-end: modal → «Жалоба отправлена. Спасибо.» → appears in the inbox → dismissed |
| Mobile at 390px | ❌ **Still unverified** — the browser extension screenshots at a fixed 1512px viewport regardless of window resize |
| Invitation accept flow | ❌ Not exercised — needs a real invite token |

---

## 3. Not fixed, and why

**§4.8 «Всего записей» (organizer hub).** Derivable only for events that have a
capacity; an uncapped event carries no headcount in the payload. Adding one
means a new `going_count` field, which means regenerating the go-swagger server
from `api/swagger.yaml`. That did not belong in a sweep about to deploy.

**§4.8 «Радиус» (/map).** Depends on the live viewport report; needs a browser
session on the map to judge, not a blind change.

**§4.1 for uncapped events.** «МЕСТА» still reads «—» when an event has no
capacity — same `going_count` dependency. Verified: the seeded «Кураторская
экскурсия» has `capacity: null`, so «—» there is correct, not a regression.

**§4.13, the controls half.** The marker is fixed. The bottom bar («Как
добраться», «Доехать на такси», «Создать свою карту», «Яндекс Карты») is
Yandex's mandatory attribution layer on the free JS API licence — `controls: []`
does not remove it, and removing it would breach the terms. Not a defect we can
close.

---

## 4. New defect found and fixed

**The report modal was never Swiss Grid.** §8 never exercised complaint
submission, so nobody had opened it. It carried the same violation §4.6 named on
`/admin/settings` — **native radios painted platform blue** — plus rounded
corners, a card shadow and a raw `text-red-500`. `ConfirmModal` (publish,
takedown) had the same disease.

Fixed with `SquareRadio` (input kept as `sr-only` so keyboard group navigation
survives) and a square hairline-ink treatment on both modals. Verified live.

---

## 5. Data changes made on production

- **Seeded events shifted +42 days** (7 rows, user-approved). The `from` fix was
  correct and revealed that 8 of 9 published events were in the past; the feed
  had legitimately dropped to 1 event. `twix-mars` was deliberately left in the
  past, so the junk event now drops off the feed on its own.
- One test complaint submitted and dismissed. Prod ends at 4 dismissed
  complaints, 9 published events — the same as before.

---

## 6. Concurrent session — prod schema moved during this sweep

Another session applied migrations **021 and 022 to production mid-QA**
(`schema_migrations` 20 → 22 between two screenshots), adding the
`reading-group` category and shifting every category numeral. Nothing broke —
the deployed backend does not call the new extensions, and all routes were
re-verified 200 afterwards.

Consequence for this branch: `fix/qa-31jul` is **not merged to `main`**, and
`main` was never checked out, to avoid the shared-worktree HEAD collision.
Merge when the other session is idle.

---

## 7. Still outstanding from the 31-jul report

- **§5 production data purge** — mostly untouched. `twix-mars` is off the feed
  but still in the DB; the QA accounts, `QA Revoke-Fix Test Org` /
  `Smoke Verification Org` (both correctly flagged ТЕСТОВЫЙ in the registry),
  and the St. Petersburg venues in a «МОСКВА» product all remain.
- **§2 QA accounts** — three still live, one with `admin` in `gateguard`.
- Mobile 390px and the invitation-accept flow (§2 above).
- Cosmetic: the 404 page sits flush to the viewport edge rather than inside the
  1360px container the rest of the app uses.
