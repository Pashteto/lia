import type { LiaEvent } from "./types";

// Pin the timezone: without it, formatting uses the runtime zone, so the SSR
// container (UTC) and the visitor's browser (local zone) produce different text
// and React throws a hydration mismatch (#418). Moscow events → Europe/Moscow.
const dateFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  timeZone: "Europe/Moscow",
});

const timeFmt = new Intl.DateTimeFormat("ru-RU", {
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "Europe/Moscow",
});

/** "13 июня, 19:00" */
export function formatEventDate(iso: string): string {
  const date = new Date(iso);
  return `${dateFmt.format(date)}, ${timeFmt.format(date)}`;
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
// The Europe/Moscow civil day ("YYYY-MM-DD") and month of an instant.
function moscowParts(iso: string): { day: string; month: string } {
  // en-CA yields ISO "YYYY-MM-DD"; split off day/month for cheap comparison.
  const [y, m, d] = new Intl.DateTimeFormat("en-CA", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: "Europe/Moscow",
  })
    .format(new Date(iso))
    .split("-");
  return { day: `${y}-${m}-${d}`, month: m };
}
// A backend zero-time ("0001-01-01…") means "no end set" (see the ends_at hotfix).
function hasRealEnd(endsAt?: string): boolean {
  return !!endsAt && new Date(endsAt).getUTCFullYear() > 1;
}

/**
 * Human date for a card/detail. Single-day events keep the familiar
 * "13 июня, 19:00" form; multi-day events render a civil-day range
 * ("15–17 августа" within a month, "31 июля – 2 августа" across months).
 */
export function formatEventRange(
  event: Pick<LiaEvent, "startsAt" | "endsAt">,
): string {
  if (!hasRealEnd(event.endsAt)) return formatEventDate(event.startsAt);
  const start = moscowParts(event.startsAt);
  const end = moscowParts(event.endsAt as string);
  if (start.day === end.day) return formatEventDate(event.startsAt);
  if (start.month === end.month) {
    // Same month → "15–17 августа" (end carries the month word).
    return `${dayFmt.format(new Date(event.startsAt))}–${dayMonthFmt.format(
      new Date(event.endsAt as string),
    )}`;
  }
  // Cross-month → "31 июля – 2 августа" (spaced en-dash, both months).
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
