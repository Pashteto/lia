/** Index to select after acting on `current`. If last, clamp to newLast; if empty → -1. */
export function nextQueueIndex(
  current: number,
  lengthAfterRemoval: number,
): number {
  if (lengthAfterRemoval === 0) return -1;
  if (current >= lengthAfterRemoval) return lengthAfterRemoval - 1;
  return current;
}

/** Which list of events the admin is looking at. */
export type ModerationFilter = "waiting" | "all" | "links";

/** What an action left the event as, in the terms the queue is built from. */
export type ModerationOutcome =
  /** «Одобрить» on a live event: still published, now stamped reviewed_at. */
  | "reviewed"
  /** Taken down or sent back for revision. */
  | "rejected"
  /** Reinstated or published from pending_review (and stamped reviewed). */
  | "published";

/**
 * Whether the acted-on row leaves the list the admin is currently looking at.
 *
 * «Все» lists published + rejected, so a taken-down event still belongs there
 * and must stay, marked «снято». Dropping it locally made it vanish until the
 * next reload, which read as «действие не сохранилось» (prod, 2026-09-18).
 * Every other combination does leave: «Ждут» holds only unreviewed published
 * events, «На проверке» only pending_review ones, and a reviewed event is out
 * of both — including out of «Все», whose published half is the queue itself.
 */
export function queueEffect(
  filter: ModerationFilter,
  outcome: ModerationOutcome,
): "remove" | "restatus" {
  if (filter === "all" && outcome === "rejected") return "restatus";
  return "remove";
}

/**
 * Which queue row an arrow key should move to, or null when the key is not
 * ours or the move would leave the list.
 *
 * Clamps at both ends rather than wrapping: moderation is a conveyor worked
 * top to bottom, and a silent jump from the last event back to the first reads
 * as "the queue ended" — the moderator loses their place.
 *
 * `index` is -1 when nothing is selected yet; ArrowDown then opens the first
 * row, which is how a keyboard user enters the list at all.
 */
export function queueKeyTarget(
  key: string,
  index: number,
  length: number,
): number | null {
  if (length === 0) return null;
  if (key === "ArrowDown") {
    const next = index + 1;
    return next < length ? next : null;
  }
  if (key === "ArrowUp") {
    const prev = index - 1;
    return prev >= 0 ? prev : null;
  }
  return null;
}
