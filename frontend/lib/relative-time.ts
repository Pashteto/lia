import { formatShortDate } from "./format";

function moscowDay(iso: string): string {
  return new Intl.DateTimeFormat("en-CA", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: "Europe/Moscow",
  }).format(new Date(iso));
}

function moscowDayBefore(day: string): string {
  const [y, m, d] = day.split("-").map(Number);
  const dt = new Date(Date.UTC(y, m - 1, d));
  dt.setUTCDate(dt.getUTCDate() - 1);
  return dt.toISOString().slice(0, 10);
}

/** Compact RU relative for O4 / activity. Pins Europe/Moscow for calendar-day checks. */
export function formatRelativeRu(iso: string, now: Date = new Date()): string {
  const then = new Date(iso);
  const diffMs = now.getTime() - then.getTime();

  if (diffMs < 0) return formatShortDate(iso);

  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return "только что";

  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin} мин`;

  const nowDay = moscowDay(now.toISOString());
  const thenDay = moscowDay(iso);
  if (thenDay === moscowDayBefore(nowDay)) return "вчера";

  const diffHours = Math.floor(diffMin / 60);
  if (diffHours < 24) return `${diffHours} ч назад`;

  return formatShortDate(iso);
}
