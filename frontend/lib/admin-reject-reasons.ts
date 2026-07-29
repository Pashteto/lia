export const REJECT_REASON_CHIPS = [
  "Тестовые данные",
  "Нет описания",
  "Обложка низкого качества",
  "Дубликат",
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
