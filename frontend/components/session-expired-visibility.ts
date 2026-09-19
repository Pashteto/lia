/**
 * Mirrors shouldShowVerifyBanner: the notice waits for hydration, because
 * before it there is no localStorage to judge the session from and the server
 * render would disagree with the client.
 */
export function shouldShowSessionExpired(s: {
  ready: boolean;
  sessionExpired: boolean;
}): boolean {
  return s.ready && s.sessionExpired;
}
