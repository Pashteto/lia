// Date math for the calendar grid. A calendar day is represented as a Date at
// UTC midnight ("civil date") and all arithmetic uses UTC methods, so there's no
// timezone/DST drift. Events are bucketed by their Europe/Moscow civil day so the
// grid matches the dates shown elsewhere in the app (see lib/format.ts).

const MOSCOW = "Europe/Moscow";

// en-CA formats as YYYY-MM-DD, which sorts and compares lexicographically.
const moscowDayFmt = new Intl.DateTimeFormat("en-CA", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  timeZone: MOSCOW,
});
const moscowTimeFmt = new Intl.DateTimeFormat("ru-RU", {
  hour: "2-digit",
  minute: "2-digit",
  timeZone: MOSCOW,
});
const monthYearFmt = new Intl.DateTimeFormat("ru-RU", {
  month: "long",
  year: "numeric",
  timeZone: "UTC", // civil dates are UTC-midnight; format in UTC to avoid shift
});
const dayLabelFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  weekday: "long",
  timeZone: "UTC",
});

/** Europe/Moscow civil day of an instant, as "YYYY-MM-DD". */
export function moscowDayKey(d: Date): string {
  return moscowDayFmt.format(d);
}

/** "19:00" in Europe/Moscow. */
export function moscowTime(d: Date): string {
  return moscowTimeFmt.format(d);
}

/** A civil date (UTC midnight) for the given year / month (0-based) / day. */
export function civil(year: number, month0: number, day: number): Date {
  return new Date(Date.UTC(year, month0, day));
}

/** "YYYY-MM-DD" key of a civil date. */
export function civilKey(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export function addDays(d: Date, n: number): Date {
  return new Date(d.getTime() + n * 86_400_000);
}

/** Today's Europe/Moscow civil date as a UTC-midnight Date. */
export function todayCivil(): Date {
  const [y, m, day] = moscowDayKey(new Date()).split("-").map(Number);
  return civil(y, m - 1, day);
}

/** Monday-based weekday index (Mon=0 … Sun=6) of a civil date. */
function mondayIndex(d: Date): number {
  return (d.getUTCDay() + 6) % 7;
}

/** The 42 civil dates (6 weeks) of the month grid containing `anchor`. */
export function monthGrid(anchor: Date): Date[] {
  const first = civil(anchor.getUTCFullYear(), anchor.getUTCMonth(), 1);
  const start = addDays(first, -mondayIndex(first));
  return Array.from({ length: 42 }, (_, i) => addDays(start, i));
}

/** The 7 civil dates (Mon–Sun) of the week containing `anchor`. */
export function weekGrid(anchor: Date): Date[] {
  const start = addDays(anchor, -mondayIndex(anchor));
  return Array.from({ length: 7 }, (_, i) => addDays(start, i));
}

export function sameMonth(a: Date, b: Date): boolean {
  return a.getUTCFullYear() === b.getUTCFullYear() && a.getUTCMonth() === b.getUTCMonth();
}

export function monthYearLabel(anchor: Date): string {
  return monthYearFmt.format(anchor);
}

export function dayLabel(anchor: Date): string {
  return dayLabelFmt.format(anchor);
}

/** Civil month start, shifted by `delta` months. */
export function shiftMonth(anchor: Date, delta: number): Date {
  return civil(anchor.getUTCFullYear(), anchor.getUTCMonth() + delta, 1);
}

/** Russian short weekday headers, Monday first. */
export const WEEKDAY_LABELS = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];

/**
 * The Europe/Moscow civil day keys ("YYYY-MM-DD") an event occupies, inclusive
 * of start and end. Single-day (or open-ended, i.e. zero-time end) → one key.
 * Used to span multi-day events across every calendar cell they cover.
 */
export function eventDayKeys(startsAt: string, endsAt: string | undefined): string[] {
  const startKey = moscowDayKey(new Date(startsAt));
  const realEnd = endsAt && new Date(endsAt).getUTCFullYear() > 1 ? endsAt : undefined;
  if (!realEnd) return [startKey];
  const endKey = moscowDayKey(new Date(realEnd));
  if (endKey <= startKey) return [startKey];
  const [sy, sm, sd] = startKey.split("-").map(Number);
  const keys: string[] = [];
  let cursor = civil(sy, sm - 1, sd);
  while (civilKey(cursor) <= endKey) {
    keys.push(civilKey(cursor));
    cursor = addDays(cursor, 1);
  }
  return keys;
}
