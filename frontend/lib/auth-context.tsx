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
import { clearSession, getStoredEmail, getToken, setSession } from "./auth";

interface AuthState {
  /** Email of the signed-in user, or null when signed out. */
  email: string | null;
  /** True once a token is present. */
  isAuthed: boolean;
  /** True until the stored session has been read on mount (avoids UI flicker). */
  ready: boolean;
  /** Role from the server (e.g. "admin"), or null. */
  role: string | null;
  /** Demo-login with an email; persists the session. Throws on failure. */
  login: (email: string, name?: string) => Promise<void>;
  /** Register with email + password; persists the session. Throws on failure. */
  register: (email: string, name: string, password: string) => Promise<void>;
  /** Log in with email + password; persists the session. Throws on failure. */
  loginPassword: (email: string, password: string) => Promise<void>;
  /** Clears the session. */
  logout: () => void;
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

  // Populate role from the server on mount when a session already exists.
  useEffect(() => {
    if (getToken()) {
      getMe().then((me) => setRole(me?.role ?? null)).catch(() => setRole(null));
    }
  }, []);

  const login = useCallback(async (loginEmail: string, name?: string) => {
    const token = await demoLogin(loginEmail, name);
    setSession(token, loginEmail);
    notifyAuthListeners();
    getMe().then((me) => setRole(me?.role ?? null)).catch(() => setRole(null));
  }, []);

  const register = useCallback(
    async (regEmail: string, name: string, password: string) => {
      const token = await registerWithPassword(regEmail, name, password);
      setSession(token, regEmail);
      notifyAuthListeners();
      getMe().then((me) => setRole(me?.role ?? null)).catch(() => setRole(null));
    },
    [],
  );

  const loginPassword = useCallback(
    async (loginEmail: string, password: string) => {
      const token = await loginWithPassword(loginEmail, password);
      setSession(token, loginEmail);
      notifyAuthListeners();
      getMe().then((me) => setRole(me?.role ?? null)).catch(() => setRole(null));
    },
    [],
  );

  const logout = useCallback(() => {
    clearSession();
    setRole(null);
    notifyAuthListeners();
  }, []);

  const value = useMemo<AuthState>(
    () => ({
      email,
      isAuthed: email !== null,
      ready,
      role,
      login,
      register,
      loginPassword,
      logout,
    }),
    [email, ready, role, login, register, loginPassword, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/** Access the auth state. Must be used under <AuthProvider>. */
export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
}
