/** Brand identity (handoff tokens.ts → brand). The only place brand strings
 * live — headers, titles, icons and the intro splash all read from here.
 * `domain` is soobshcha.ru, not the handoff's soobscha.ru: that one belongs to
 * a third party (whois 2026-09-29). Change it here and nowhere else. */
export const BRAND = {
  name: "Сообща",
  wordmark: "сообща",
  domain: "soobshcha.ru",
  monogram: "сб",
} as const;

/** Page <title>: «Вход — Сообща»; the root is «Сообща — События». */
export function pageTitle(section?: string): string {
  return section ? `${section} — ${BRAND.name}` : `${BRAND.name} — События`;
}
