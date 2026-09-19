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

/**
 * Whether a stored bearer token's own `exp` claim has already passed.
 *
 * Read locally, deliberately: the backend answers a bare 401 on /auth/me both
 * for an expired token and for a session it refuses for other reasons, so the
 * response cannot tell the two apart — but the JWT carries its own deadline.
 * Without this the admin gate sat on its loading skeleton forever (prod,
 * 2026-09-19: a 72h token 16h past expiry).
 *
 * Unreadable input is never reported as expired. A false positive here signs a
 * working user out, which is worse than the stall this exists to end.
 */
export function isTokenExpired(token: string | null): boolean {
  if (!token) return false;
  const parts = token.split(".");
  if (parts.length !== 3) return false;
  try {
    const base64 = parts[1]!.replace(/-/g, "+").replace(/_/g, "/");
    const payload: unknown = JSON.parse(
      typeof atob === "function"
        ? atob(base64)
        : Buffer.from(base64, "base64").toString("utf8"),
    );
    if (typeof payload !== "object" || payload === null) return false;
    const exp = (payload as { exp?: unknown }).exp;
    if (typeof exp !== "number" || !Number.isFinite(exp)) return false;
    return exp <= Math.floor(Date.now() / 1000);
  } catch {
    return false;
  }
}
