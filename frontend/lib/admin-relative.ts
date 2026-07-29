import { formatShortDate } from "./format";

function moscowDay(iso: string): string {
  return new Intl.DateTimeFormat("en-CA", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: "Europe/Moscow",
  }).format(new Date(iso));
}

function moscowDayDiff(iso: string, now: Date): number {
  const thenDay = moscowDay(iso);
  const nowDay = moscowDay(now.toISOString());
  const [y1, m1, d1] = thenDay.split("-").map(Number);
  const [y2, m2, d2] = nowDay.split("-").map(Number);
  const thenMs = Date.UTC(y1, m1 - 1, d1);
  const nowMs = Date.UTC(y2, m2 - 1, d2);
  return Math.floor((nowMs - thenMs) / (24 * 60 * 60 * 1000));
}

/** Compact RU relative for admin queues. Moscow day boundaries. */
export function formatRelativeCompactRu(iso: string, now: Date = new Date()): string {
  const then = new Date(iso);
  const diffMs = now.getTime() - then.getTime();

  if (diffMs < 0) return formatShortDate(iso);

  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return "сейчас";

  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin} мин`;

  const diffHours = Math.floor(diffMin / 60);
  if (diffHours < 24) return `${diffHours} ч`;

  const diffDays = moscowDayDiff(iso, now);
  if (diffDays < 7) return `${diffDays} д`;

  return formatShortDate(iso);
}
