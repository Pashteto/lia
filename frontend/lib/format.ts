import type { LiaEvent } from "./types";

// Pin the timezone: without it, formatting uses the runtime zone, so the SSR
// container (UTC) and the visitor's browser (local zone) produce different text
// and React throws a hydration mismatch (#418). Moscow events → Europe/Moscow.
const dateFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "Europe/Moscow",
});

/** "13 июня в 19:00" */
export function formatEventDate(iso: string): string {
  return dateFmt.format(new Date(iso));
}

// Day + month only, e.g. "15 августа" — for the ends of a multi-day range.
const dayMonthFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  timeZone: "Europe/Moscow",
});
// Day only, e.g. "15" — used when both ends share a month.
const dayFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  timeZone: "Europe/Moscow",
});
// The Europe/Moscow civil day of an instant, as ISO "YYYY-MM-DD" (en-CA yields it).
function moscowDay(iso: string): string {
  return new Intl.DateTimeFormat("en-CA", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: "Europe/Moscow",
  }).format(new Date(iso));
}
// A backend zero-time ("0001-01-01…") means "no end set" (see the ends_at hotfix).
function hasRealEnd(endsAt?: string): boolean {
  return !!endsAt && new Date(endsAt).getUTCFullYear() > 1;
}

/**
 * Human date for a card/detail. Single-day events keep the familiar
 * "13 июня в 19:00" form; multi-day events render a civil-day range
 * ("15–17 августа" within one month, "31 июля – 2 августа" across months
 * or years).
 */
export function formatEventRange(
  event: Pick<LiaEvent, "startsAt" | "endsAt">,
): string {
  if (!hasRealEnd(event.endsAt)) return formatEventDate(event.startsAt);
  const startDay = moscowDay(event.startsAt);
  const endDay = moscowDay(event.endsAt as string);
  if (startDay === endDay) return formatEventDate(event.startsAt);
  // Same year AND month (compare "YYYY-MM") → "15–17 августа".
  if (startDay.slice(0, 7) === endDay.slice(0, 7)) {
    return `${dayFmt.format(new Date(event.startsAt))}–${dayMonthFmt.format(
      new Date(event.endsAt as string),
    )}`;
  }
  // Different month or year → "31 июля – 2 августа" (spaced en-dash, both months).
  return `${dayMonthFmt.format(new Date(event.startsAt))} – ${dayMonthFmt.format(
    new Date(event.endsAt as string),
  )}`;
}

/** "Бесплатно" | "2 500 ₽" | "от 500 ₽" */
export function formatPrice(event: LiaEvent): string {
  if (event.priceType === "free") return "Бесплатно";
  const amount = (event.priceMin ?? 0).toLocaleString("ru-RU");
  return event.priceType === "from" ? `от ${amount} ₽` : `${amount} ₽`;
}

/** "11 / 18 участников" | "64 участника" */
export function formatAttendance(event: LiaEvent): string | null {
  if (event.attendeeCount == null) return null;
  return event.capacity != null
    ? `${event.attendeeCount} / ${event.capacity} участников`
    : `${event.attendeeCount} участников`;
}

// Compact numeric forms for the Swiss Grid modules — all numerals render in
// JetBrains Mono at the call site. Moscow-pinned like everything above.
const shortDateFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "2-digit",
  month: "2-digit",
  timeZone: "Europe/Moscow",
});
const shortTimeFmt = new Intl.DateTimeFormat("ru-RU", {
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "Europe/Moscow",
});

/** "16:00" — Moscow wall clock. */
export function formatStartTime(iso: string): string {
  return shortTimeFmt.format(new Date(iso));
}

/** "12.07 · 16:00" single-day; "15.08 – 17.08" across civil days. */
export function formatModuleDate(startsAt: string, endsAt?: string): string {
  const single = `${shortDateFmt.format(new Date(startsAt))} · ${formatStartTime(startsAt)}`;
  if (!hasRealEnd(endsAt)) return single;
  if (moscowDay(startsAt) === moscowDay(endsAt as string)) return single;
  return `${shortDateFmt.format(new Date(startsAt))} – ${shortDateFmt.format(new Date(endsAt as string))}`;
}

/** "20:00" (Moscow) when a real end is set; null when the backend zero-time
 * placeholder means "no end". Detail-page «Окончание» cell. */
export function formatEndTime(endsAt?: string): string | null {
  return hasRealEnd(endsAt) ? shortTimeFmt.format(new Date(endsAt as string)) : null;
}

/** "12 / 40" | "64" | "Без ограничения" | "Ограничено · наличие на сайте
 * регистрации" — seat counter for cells and module footers. A bare "—" for an
 * unset limit read as "data failed to load" (design review P2), so the
 * no-capacity case is spelled out. `capacityLimited` marks an external event
 * whose real attendee count Lia doesn't track — showing a stale/zero count
 * there would misrepresent availability, so it's replaced with a pointer to
 * the registration site instead. */
export function attendanceShort(
  event: Pick<LiaEvent, "attendeeCount" | "capacity" | "capacityLimited">,
): string {
  if (event.capacityLimited) {
    return "Ограничено · наличие на сайте регистрации";
  }
  if (event.capacity == null) {
    return event.attendeeCount ? String(event.attendeeCount) : "Без ограничения";
  }
  return `${event.attendeeCount ?? 0} / ${event.capacity}`;
}

/** "12.07" — Moscow civil day, for the U6 date column. */
export function formatShortDate(iso: string): string {
  return shortDateFmt.format(new Date(iso));
}
