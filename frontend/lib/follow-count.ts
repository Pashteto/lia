/** Follower count with the optimistic follow/unfollow override applied
 * (QA-23-aug №9: the counter used to stay stale until a full reload). */
export function adjustedFollowers(
  base: number | undefined,
  serverFollowing: boolean,
  override: boolean | null,
): number {
  const b = base ?? 0;
  if (override === null || override === serverFollowing) return b;
  return Math.max(0, b + (override ? 1 : -1));
}
