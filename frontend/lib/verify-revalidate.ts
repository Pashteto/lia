/**
 * Verification state is fetched once per session (mount, login, or the explicit
 * refresh() the /auth/verify page calls). That leaves every OTHER open tab
 * stale: a user who confirms the code in a second tab — or on their phone —
 * keeps seeing "почта не подтверждена" in the first one until they log out and
 * back in. Re-reading /auth/me when a tab is brought back to the front closes
 * that window without polling.
 *
 * The refetch is deliberately one-directional: only an unverified tab asks
 * again. Verification never flips back to false, so a verified tab has nothing
 * to learn, and every tab switch would otherwise cost a request.
 */
export function shouldRevalidateVerification(s: {
  /** A session token is present in this tab. */
  hasToken: boolean;
  /** Last known server answer for this session. */
  emailVerified: boolean;
  /** document.visibilityState at the moment of the event. */
  visibility: string;
}): boolean {
  return s.hasToken && !s.emailVerified && s.visibility === "visible";
}
