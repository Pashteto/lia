"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
} from "react";

import { demoLogin, getMe, loginWithPassword, registerWithPassword } from "./api";
import { clearSession, getStoredEmail, getToken, isTokenExpired, setSession } from "./auth";
import { shouldRevalidateVerification } from "./verify-revalidate";

interface AuthState {
  /** Email of the signed-in user, or null when signed out. */
  email: string | null;
  /** True once a token is present. */
  isAuthed: boolean;
  /** True until the stored session has been read on mount (avoids UI flicker). */
  ready: boolean;
  /** Role from the server (e.g. "admin"), or null. */
  role: string | null;
  /**
   * True once the role lifecycle has settled for the current session — i.e.
   * getMe() has resolved (success or error) after mount or after a login.
   * False on mount until the fetch settles, and reset to false on logout.
   * When there is no token on mount, stays false; the gate uses isAuthed to
   * short-circuit before checking roleResolved.
   */
  roleResolved: boolean;
  /** Whether the signed-in user's email is verified, per the server. False when unknown/signed out. */
  emailVerified: boolean;
  /**
   * True when the stored session was found past its own expiry and torn down.
   * Drives the notice that tells the user — of any role — to sign in again;
   * before it existed an expired token left the admin gate on its skeleton
   * forever, with nothing on screen to explain why.
   */
  sessionExpired: boolean;
  /** Demo-login with an email; persists the session. Throws on failure. */
  login: (email: string, name?: string) => Promise<void>;
  /** Register with email + password; persists the session. Throws on failure. */
  register: (email: string, name: string, password: string) => Promise<void>;
  /** Log in with email + password; persists the session. Throws on failure. */
  loginPassword: (email: string, password: string) => Promise<void>;
  /** Clears the session. */
  logout: () => void;
  /** Re-fetches /auth/me and updates role/emailVerified (e.g. after verifying email). */
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

// ---------------------------------------------------------------------------
// useSyncExternalStore wiring — mirrors the ThemeToggle pattern so SSR markup
// and the client's first render are identical (signed-out / not-ready).
// ---------------------------------------------------------------------------

const authListeners = new Set<() => void>();

function notifyAuthListeners() {
  authListeners.forEach((cb) => cb());
}

function subscribeAuth(callback: () => void) {
  authListeners.add(callback);
  return () => {
    authListeners.delete(callback);
  };
}

// `ready` flips false→true once on mount via the server/client snapshot
// difference; it is not driven by auth events, so it needs no live subscription.
function subscribeReady(): () => void {
  return () => {};
}

/** Client snapshot: read the live localStorage state. */
function getAuthSnapshot(): string | null {
  return getToken() ? getStoredEmail() : null;
}

/** Server snapshot: always signed-out so SSR markup matches first client render. */
function getAuthServerSnapshot(): string | null {
  return null;
}

/** Client snapshot: true after hydration (window exists). */
function getReadySnapshot(): boolean {
  return typeof window !== "undefined";
}

/** Server snapshot: false — not-ready matches the server render. */
function getReadyServerSnapshot(): boolean {
  return false;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  // useSyncExternalStore guarantees SSR markup === first client render:
  //   server / first-client: email=null, ready=false  (server snapshots)
  //   after hydration:        email=<stored>, ready=true (client snapshots)
  const email = useSyncExternalStore(
    subscribeAuth,
    getAuthSnapshot,
    getAuthServerSnapshot,
  );
  const ready = useSyncExternalStore(
    subscribeReady,
    getReadySnapshot,
    getReadyServerSnapshot,
  );

  const [role, setRole] = useState<string | null>(null);
  const [roleResolved, setRoleResolved] = useState(false);
  const [emailVerified, setEmailVerified] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);

  /**
   * Tears down a session whose token is past its own `exp` and raises the
   * notice. Returns whether it acted, so callers can skip the request that
   * would only come back 401.
   */
  const dropIfExpired = useCallback((): boolean => {
    if (!isTokenExpired(getToken())) return false;
    clearSession();
    setRole(null);
    setEmailVerified(false);
    setRoleResolved(false);
    setSessionExpired(true);
    notifyAuthListeners();
    return true;
  }, []);

  // Shared getMe result handlers. Success resolves the role; failure leaves
  // roleResolved=false so verification-gated UI (the banner) stays hidden
  // rather than showing a false "unverified" state off a failed request.
  const applyMe = useCallback(
    (me: Awaited<ReturnType<typeof getMe>>) => {
      setRole(me?.role ?? null);
      setEmailVerified(me?.emailVerified ?? false);
      setRoleResolved(true);
    },
    [],
  );
  // NB: no clearSession on 401 — the backend answers 401 on /auth/me for a
  // freshly registered (still unverified) session, and tearing it down there
  // would log the user out mid-verification (found on prod re-test).
  const failMe = useCallback((_err: unknown) => {
    // An expired token is the one 401 we CAN identify, and it is a dead end:
    // leave it in place and the gate waits on roleResolved forever.
    if (dropIfExpired()) return;
    setRole(null);
    setEmailVerified(false);
    setRoleResolved(false);
  }, [dropIfExpired]);

  // Populate role from the server on mount when a session already exists.
  useEffect(() => {
    const token = getToken();
    if (!token) return; // the gate uses isAuthed first; roleResolved stays false
    // Checked before the request: a token we already know is stale would only
    // earn a 401, and the user is better told than made to wait for it. The
    // teardown is deferred a tick because setting state synchronously inside an
    // effect cascades renders (and eslint rightly refuses it).
    if (isTokenExpired(token)) {
      queueMicrotask(dropIfExpired);
      return;
    }
    getMe().then(applyMe).catch(failMe);
  }, [applyMe, failMe, dropIfExpired]);

  const login = useCallback(async (loginEmail: string, name?: string) => {
    const token = await demoLogin(loginEmail, name);
    setSession(token, loginEmail);
    setSessionExpired(false);
    notifyAuthListeners();
    getMe().then(applyMe).catch(failMe);
  }, [applyMe, failMe]);

  const register = useCallback(
    async (regEmail: string, name: string, password: string) => {
      const token = await registerWithPassword(regEmail, name, password);
      setSession(token, regEmail);
      setSessionExpired(false);
      notifyAuthListeners();
      getMe().then(applyMe).catch(failMe);
    },
    [applyMe, failMe],
  );

  const loginPassword = useCallback(
    async (loginEmail: string, password: string) => {
      const token = await loginWithPassword(loginEmail, password);
      setSession(token, loginEmail);
      setSessionExpired(false);
      notifyAuthListeners();
      getMe().then(applyMe).catch(failMe);
    },
    [applyMe, failMe],
  );

  const refresh = useCallback(async () => {
    if (!getToken()) return;
    try {
      applyMe(await getMe());
    } catch (err) {
      failMe(err);
    }
  }, [applyMe, failMe]);

  // Verification can be completed anywhere — a second tab, another device, the
  // link in the mail client — and this tab would keep showing the banner until
  // the session was torn down and rebuilt. Ask again whenever an unverified tab
  // is brought back to the front.
  useEffect(() => {
    const recheck = () => {
      if (
        shouldRevalidateVerification({
          hasToken: getToken() !== null,
          emailVerified,
          visibility: document.visibilityState,
        })
      ) {
        void refresh();
      }
    };
    document.addEventListener("visibilitychange", recheck);
    window.addEventListener("focus", recheck);
    return () => {
      document.removeEventListener("visibilitychange", recheck);
      window.removeEventListener("focus", recheck);
    };
  }, [emailVerified, refresh]);

  const logout = useCallback(() => {
    clearSession();
    setRole(null);
    setRoleResolved(false);
    setEmailVerified(false);
    // Signing out deliberately is not an expiry — clear the notice so it does
    // not follow the user onto the login screen.
    setSessionExpired(false);
    notifyAuthListeners();
  }, []);

  const value = useMemo<AuthState>(
    () => ({
      email,
      isAuthed: email !== null,
      ready,
      role,
      roleResolved,
      emailVerified,
      sessionExpired,
      login,
      register,
      loginPassword,
      logout,
      refresh,
    }),
    [email, ready, role, roleResolved, emailVerified, sessionExpired, login, register, loginPassword, logout, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/** Access the auth state. Must be used under <AuthProvider>. */
export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
}
