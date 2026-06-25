// Client-side session store for the demo-login flow.
//
// DEMO-ONLY auth: the backend's POST /auth/demo-login mints a GateGuard JWT for
// any email (no password, no Google) and returns it in the response body. We
// keep that token in localStorage and attach it as `Authorization: Bearer` on
// write requests. This is a demo control — a real deployment would use an
// httpOnly cookie set by a server-side OIDC callback (see auth spec §6.3).

const TOKEN_KEY = "lia.auth.token";
const EMAIL_KEY = "lia.auth.email";

/** Returns the stored bearer token, or null on the server / when signed out. */
export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(TOKEN_KEY);
}

/** Returns the email of the signed-in user, or null. */
export function getStoredEmail(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(EMAIL_KEY);
}

/** Persists a session after a successful login. */
export function setSession(token: string, email: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TOKEN_KEY, token);
  window.localStorage.setItem(EMAIL_KEY, email);
}

/** Clears the session on sign-out. */
export function clearSession(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TOKEN_KEY);
  window.localStorage.removeItem(EMAIL_KEY);
}
