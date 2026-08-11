// Human messages for RSVP CTA failures. api.ts throws raw "<action> failed: <status>"
// strings; a design review caught "sign up failed: 401" reaching the UI at the
// single most important conversion moment. Status codes must never leak to users.

/** Parses the trailing HTTP status of an api.ts raw-throw message, or null. */
export function extractHttpStatus(err: unknown): number | null {
  if (!(err instanceof Error)) return null;
  const m = /failed: (\d{3})$/.exec(err.message);
  return m ? Number(m[1]) : null;
}

/**
 * Maps an RSVP failure to a Russian message. 401 is intentionally absent —
 * callers route it to the login modal instead of showing text.
 */
export function rsvpErrorMessage(err: unknown, action: "signup" | "cancel"): string {
  const status = extractHttpStatus(err);
  if (status === 409) {
    return "Похоже, вы уже записаны. Обновите страницу, чтобы увидеть свой статус.";
  }
  if (status === 429) {
    return "Слишком много попыток. Подождите минуту и попробуйте ещё раз.";
  }
  return action === "cancel"
    ? "Не удалось отменить запись. Попробуйте ещё раз."
    : "Не удалось записаться. Попробуйте ещё раз.";
}
