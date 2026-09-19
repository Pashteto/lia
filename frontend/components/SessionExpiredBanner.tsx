"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { shouldShowSessionExpired } from "@/components/session-expired-visibility";

/**
 * Tells the user their session ran out and they need to sign in again.
 *
 * Mounted in the root layout, so it reaches every role — visitor, organizer and
 * admin alike. Until it existed an expired token was invisible: the feed simply
 * behaved as if signed out, and the admin gate sat on its loading skeleton
 * forever with nothing on screen to explain why (prod, 2026-09-19).
 */
export function SessionExpiredBanner() {
  const { ready, sessionExpired } = useAuth();

  if (!shouldShowSessionExpired({ ready, sessionExpired })) return null;

  return (
    <div
      role="status"
      className="flex items-center justify-between gap-3 bg-amber-50 px-4 py-2 text-[13px] text-amber-900 dark:bg-amber-950 dark:text-amber-100"
    >
      <span>Сессия истекла — войдите заново, чтобы продолжить.</span>
      <Link href="/login" className="shrink-0 rounded-capsule bg-accent px-3 py-1 text-white">
        Войти
      </Link>
    </div>
  );
}
