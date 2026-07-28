import type { DiscoverIntent } from "./discover-intent";
import { haversineKm, type LatLon } from "./geo";
import { weekendRange } from "./mock-events";
import type { LiaEvent } from "./types";

export interface DiscoverHit {
  event: LiaEvent;
  matchReason: string;
  score: number;
}

export interface DiscoverRanking {
  answer: string;
  hits: DiscoverHit[];
  usedKidsFallback: boolean;
  omittedDistanceClaim: boolean;
}

const QUIET_SLUGS = new Set(["mediation", "reading-group", "exhibition"]);
const NEAR_KM = 5;
const KIDS_RE = /дет|семь|юн(ый|ая|ое|ые)|школ/i;
const DAY_MS = 24 * 60 * 60 * 1000;

const hourFmt = new Intl.DateTimeFormat("en-GB", {
  hour: "numeric",
  hourCycle: "h23",
  timeZone: "Europe/Moscow",
});
const weekdayFmt = new Intl.DateTimeFormat("en-US", {
  weekday: "short",
  timeZone: "Europe/Moscow",
});

export function moscowHour(iso: string): number {
  return Number(hourFmt.format(new Date(iso)));
}

function isMoscowWeekday(iso: string): boolean {
  const w = weekdayFmt.format(new Date(iso));
  return w !== "Sat" && w !== "Sun";
}

function primarySlug(e: LiaEvent): string | undefined {
  return e.categories[0]?.slug;
}

function sortUpcomingFirst(a: LiaEvent, b: LiaEvent, nowMs: number): number {
  const ta = new Date(a.startsAt).getTime();
  const tb = new Date(b.startsAt).getTime();
  const aUp = ta >= nowMs;
  const bUp = tb >= nowMs;
  if (aUp !== bUp) return aUp ? -1 : 1;
  return aUp ? ta - tb : tb - ta;
}

function takeTop3(scored: DiscoverHit[], nowMs: number): DiscoverHit[] {
  return [...scored]
    .sort((a, b) => {
      if (b.score !== a.score) return b.score - a.score;
      return sortUpcomingFirst(a.event, b.event, nowMs);
    })
    .slice(0, 3);
}

function quietReason(e: LiaEvent): string {
  if (e.capacity != null && e.capacity <= 12) return `группа ${e.capacity} человек`;
  if (e.categories.some((c) => QUIET_SLUGS.has(c.slug))) return "тихий формат";
  return "выходные";
}

function eveningReason(e: LiaEvent): string {
  return isMoscowWeekday(e.startsAt) ? "вечер, будний день" : "вечер";
}

function withinKm(e: LiaEvent, at: LatLon): boolean {
  const lat = e.venue?.lat;
  const lon = e.venue?.lon;
  if (lat == null || lon == null) return false;
  return haversineKm(at, [lat, lon]) <= NEAR_KM;
}

function textBlob(e: LiaEvent): string {
  return [e.title, e.description ?? "", e.organizer?.name ?? "", e.venue?.name ?? ""]
    .join(" ")
    .toLowerCase();
}

export function rankDiscover(
  events: readonly LiaEvent[],
  intent: DiscoverIntent,
  opts: { now: Date; userLatLon?: LatLon | null },
): DiscoverRanking {
  const nowMs = opts.now.getTime();
  const geo = opts.userLatLon ?? null;
  let usedKidsFallback = false;
  let omittedDistanceClaim = false;
  let scored: DiscoverHit[] = [];

  if (intent.id === "quiet_weekend") {
    const { from, to } = weekendRange(opts.now);
    scored = events
      .filter((e) => {
        const t = new Date(e.startsAt).getTime();
        return t >= from.getTime() && t < to.getTime();
      })
      .map((e) => {
        let score = 1;
        if (e.categories.some((c) => QUIET_SLUGS.has(c.slug))) score += 3;
        if (e.capacity != null && e.capacity <= 12) score += 2;
        return { event: e, score, matchReason: quietReason(e) };
      });
  } else if (intent.id === "quiet_any") {
    scored = events.map((e) => {
      let score = 1;
      if (e.categories.some((c) => QUIET_SLUGS.has(c.slug))) score += 3;
      if (e.capacity != null && e.capacity <= 12) score += 2;
      const reason =
        e.capacity != null && e.capacity <= 12
          ? `группа ${e.capacity} человек`
          : e.categories.some((c) => QUIET_SLUGS.has(c.slug))
            ? "тихий формат"
            : "ближайшие события";
      return { event: e, score, matchReason: reason };
    });
  } else if (intent.id === "free_nearby") {
    const free = events.filter((e) => e.priceType === "free");
    if (geo) {
      scored = free
        .filter((e) => withinKm(e, geo))
        .map((e) => ({ event: e, score: 2, matchReason: "бесплатно" }));
      omittedDistanceClaim = false;
    } else {
      scored = free.map((e) => ({ event: e, score: 1, matchReason: "бесплатно" }));
      omittedDistanceClaim = true;
    }
  } else if (intent.id === "evening_pair") {
    const until = nowMs + 7 * DAY_MS;
    scored = events
      .filter((e) => {
        const t = new Date(e.startsAt).getTime();
        return t >= nowMs && t < until && moscowHour(e.startsAt) >= 18;
      })
      .map((e) => {
        let score = 1;
        if (e.format === "offline") score += 1;
        if (e.capacity != null && e.capacity <= 40) score += 1;
        return { event: e, score, matchReason: eveningReason(e) };
      });
  } else if (intent.id === "with_kids") {
    const hard = events.filter((e) => KIDS_RE.test(textBlob(e)));
    if (hard.length > 0) {
      scored = hard.map((e) => ({
        event: e,
        score: 2,
        matchReason: "семейный формат",
      }));
    } else {
      usedKidsFallback = true;
      scored = events.map((e) => ({
        event: e,
        score: 1,
        matchReason: "ближайшие события",
      }));
    }
  } else {
    // keyword
    const needle = intent.keyword.toLowerCase();
    scored = events
      .filter((e) => textBlob(e).includes(needle))
      .map((e) => ({
        event: e,
        score: primarySlug(e) && needle.includes(primarySlug(e)!) ? 2 : 1,
        matchReason: "совпадение по тексту",
      }));
  }

  const hits = takeTop3(scored, nowMs);
  const n = hits.length;

  let answer: string;
  switch (intent.id) {
    case "quiet_weekend":
      answer =
        n === 0
          ? "Пока тихого на выходных не нашлось — соберите вручную ниже."
          : `Нашла ${n} ${n === 1 ? "событие" : "события"} на выходных — тихо и без спешки.`;
      break;
    case "quiet_any":
      answer =
        n === 0
          ? "Тихого формата пока не нашлось — соберите вручную ниже."
          : `Нашла ${n} ${n === 1 ? "событие" : "события"} в спокойном формате.`;
      break;
    case "free_nearby":
      answer = omittedDistanceClaim
        ? "Бесплатные события в городе."
        : "Бесплатные события рядом — до 5 км.";
      if (n === 0) {
        answer = omittedDistanceClaim
          ? "Бесплатных событий пока нет — соберите вручную ниже."
          : "Бесплатных рядом не нашлось — попробуйте без геолокации или ленту.";
      }
      break;
    case "evening_pair":
      answer =
        n === 0
          ? "Вечерних событий на ближайшие дни нет — соберите вручную ниже."
          : "Вечерние события на ближайшие дни.";
      break;
    case "with_kids":
      answer = usedKidsFallback
        ? "Ближайшие события — уточните на странице, подойдёт ли с детьми."
        : n === 0
          ? "Пока ничего не нашлось — соберите вручную ниже."
          : "События, где обычно бывает с детьми.";
      break;
    default:
      answer =
        n === 0
          ? `По запросу «${intent.sourceLabel}» ничего не нашлось.`
          : `По запросу «${intent.sourceLabel}» — вот что нашлось.`;
  }

  return { answer, hits, usedKidsFallback, omittedDistanceClaim };
}
