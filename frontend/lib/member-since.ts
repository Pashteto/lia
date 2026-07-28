// Genitive month names: Intl's ru-RU "long" month is nominative and appends
// "г." to the year ("март 2026 г."), which is not the handoff string.
const MONTHS_GENITIVE = [
  "января", "февраля", "марта", "апреля", "мая", "июня",
  "июля", "августа", "сентября", "октября", "ноября", "декабря",
] as const;

// en-CA yields "YYYY-MM-DD"; pinned to Moscow like every other formatter.
const moscowDayFmt = new Intl.DateTimeFormat("en-CA", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  timeZone: "Europe/Moscow",
});

/** U6 identity caption: "Участник с марта 2026". Null when unknown — the
 * caller drops the caption rather than printing a placeholder date. */
export function memberSince(iso: string | null | undefined): string | null {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  // The backend serializes an unset time.Time as year 0001 (same rule as
  // ends_at in lib/api.ts) — treat anything pre-1971 as "unknown".
  if (d.getUTCFullYear() <= 1970) return null;
  const [year, month] = moscowDayFmt.format(d).split("-").map(Number);
  return `Участник с ${MONTHS_GENITIVE[month - 1]} ${year}`;
}
