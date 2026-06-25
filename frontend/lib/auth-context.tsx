"use client";

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";

import { demoLogin } from "./api";
import { clearSession, getStoredEmail, getToken, setSession } from "./auth";

interface AuthState {
  /** Email of the signed-in user, or null when signed out. */
  email: string | null;
  /** True once a token is present. */
  isAuthed: boolean;
  /** True until the stored session has been read on mount (avoids UI flicker). */
  ready: boolean;
  /** Demo-login with an email; persists the session. Throws on failure. */
  login: (email: string, name?: string) => Promise<void>;
  /** Clears the session. */
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  // Use lazy initialisers so localStorage is read once on the client without
  // triggering a second render via setState-in-effect. getToken() / getStoredEmail()
  // return null on the server (typeof window === "undefined"), so the initial
  // server-render and client hydration both produce null/false — no mismatch.
  const [email, setEmail] = useState<string | null>(() =>
    getToken() ? getStoredEmail() : null,
  );
  // ready: false on the server; true immediately on the client so the UI never
  // blocks behind an effect tick.
  const [ready] = useState(() => typeof window !== "undefined");

  const login = useCallback(async (loginEmail: string, name?: string) => {
    const token = await demoLogin(loginEmail, name);
    setSession(token, loginEmail);
    setEmail(loginEmail);
  }, []);

  const logout = useCallback(() => {
    clearSession();
    setEmail(null);
  }, []);

  const value = useMemo<AuthState>(
    () => ({ email, isAuthed: email !== null, ready, login, logout }),
    [email, ready, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/** Access the auth state. Must be used under <AuthProvider>. */
export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
}
