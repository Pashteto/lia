/**
 * Reasons a moderator can attach to a take-down, in one tap.
 *
 * The A2 handoff specified the first four; the rest were added once the queue
 * filled with imported editorial content, where the recurring problems are
 * different — a lecture that is simply not what this platform is for, a date
 * a parser guessed wrong, an announcement republished without the credit its
 * venue asked for.
 *
 * «Не наша тема», not «не подходит формату площадки»: «площадка» here means a
 * physical venue (there is a whole catalogue of them, and the card shows one
 * two rows above), so that wording would read as «wrong hall for this event».
 *
 * The four original chips keep their positions — moderators have tapped them
 * for months — and the new ones are appended as a group.
 *
 * None may contain "; ": that is the separator concatenateReasons() joins on,
 * and a reason carrying it would be indistinguishable from two reasons.
 */
export const REJECT_REASON_CHIPS = [
  "Тестовые данные",
  "Нет описания",
  "Обложка низкого качества",
  "Дубликат",
  "Не наша тема",
  "Неверная дата или время",
  "Нет источника",
  "Реклама или продажа",
] as const;
export type RejectReason = (typeof REJECT_REASON_CHIPS)[number];

/** Join selected reasons with "; ". Empty → "". */
export function concatenateReasons(reasons: readonly string[]): string {
  return reasons.join("; ");
}

/** Prefix for НА ДОРАБОТКУ takedown body. */
export function revisionReason(reasons: readonly string[]): string {
  return `На доработку: ${concatenateReasons(reasons)}`;
}
