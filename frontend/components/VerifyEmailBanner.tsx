"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth-context";

/**
 * Persistent, non-dismissible notice shown while a signed-in user's email is
 * unverified. This is the only proactive signal that verification exists —
 * without it, users discover it only by being blocked mid-action.
 *
 * Unmounts itself: /auth/verify calls refresh() on success, flipping
 * emailVerified in the auth context.
 */
export function VerifyEmailBanner() {
  const { isAuthed, ready, emailVerified } = useAuth();

  // `ready` gates the first paint: without it the banner flashes for
  // already-verified users while /auth/me is still in flight.
  if (!ready || !isAuthed || emailVerified) return null;

  return (
    <div className="flex items-center justify-between gap-3 bg-amber-50 px-4 py-2 text-[13px] text-amber-900 dark:bg-amber-950 dark:text-amber-100">
      <span>Почта не подтверждена — часть действий недоступна.</span>
      <Link
        href="/auth/verify"
        className="shrink-0 rounded-capsule bg-accent px-3 py-1 text-white"
      >
        Подтвердить
      </Link>
    </div>
  );
}
