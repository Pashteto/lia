import { formatRelativeCompactRu } from "./admin-relative";
import { pluralRu } from "./plural";

const EVENT_FORMS: [string, string, string] = ["событие", "события", "событий"];

/** «3 события с тестовыми данными» — the A1 hygiene signal line. */
export function testDataSignalRu(n: number): string {
  return `${n} ${pluralRu(n, EVENT_FORMS)} с тестовыми данными`;
}

/** «3 события с жалобами» — the A1 complaints signal line. */
export function complaintsSignalRu(n: number): string {
  return `${n} ${pluralRu(n, EVENT_FORMS)} с жалобами`;
}

/**
 * «подано 2 д назад» | «подано сейчас» | «подано 05.07» | «—».
 *
 * The whole phrase, because «назад» is only grammatical after a relative
 * value. The caller used to append it to `formatRelativeCompactRu` output
 * unconditionally, which produced «подано 05.07 назад» once an item aged past
 * the formatter's 7-day relative window.
 */
export function submittedAgoRu(iso: string | undefined, now: Date = new Date()): string {
  if (!iso) return "—";
  const value = formatRelativeCompactRu(iso, now);
  const relative = /^\d+\s(мин|ч|д)$/.test(value);
  return relative ? `подано ${value} назад` : `подано ${value}`;
}
