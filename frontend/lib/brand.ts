/** Brand identity (handoff tokens.ts → brand). The only place brand strings
 * live — headers, titles, icons and the intro splash all read from here.
 * `domain` is pending a registrar decision (soobscha.ru is third-party owned
 * as of 2026-09-24); change it here and nowhere else. */
export const BRAND = {
  name: "Сообща",
  wordmark: "сообща",
  domain: "soobscha.ru",
  monogram: "сб",
} as const;

/** Page <title>: «Вход — Сообща»; the root is «Сообща — События». */
export function pageTitle(section?: string): string {
  return section ? `${section} — ${BRAND.name}` : `${BRAND.name} — События`;
}
