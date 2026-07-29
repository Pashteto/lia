// en-CA yields "YYYY-MM-DD"; pinned to Moscow like every other formatter.
const moscowDayFmt = new Intl.DateTimeFormat("en-CA", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  timeZone: "Europe/Moscow",
});

/** A4 «Регистрация» column: mono MM.YYYY, or «—» when unknown. Go serializes
 * an unset time.Time as year 0001 — treat anything pre-1971 as unknown
 * (same rule as lib/member-since.ts). */
export function formatRegistrationMonth(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  if (d.getUTCFullYear() <= 1970) return "—";
  const [year, month] = moscowDayFmt.format(d).split("-");
  return `${month}.${year}`;
}
