/**
 * Whether an RSVP status means the user actually attended, and may therefore
 * leave feedback.
 *
 * Mirrors the server rule in `internal/feedback/repository.go` —
 * `status IN ('going','accepted')`. Anything else (applied, waitlist, declined,
 * withdrawn, cancelled, or no RSVP at all) gets 403 ErrNotParticipant, so the
 * form must not be offered.
 */
export function isActiveParticipant(status: string | undefined): boolean {
  return status === "going" || status === "accepted";
}
