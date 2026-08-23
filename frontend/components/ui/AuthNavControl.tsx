"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth-context";

const NAV_ITEM = "swiss-focus font-alt text-[9px] uppercase tracking-[0.14em]";

/** AppHeader actions slot: «Войти» when signed out; email + «Выйти» (+Админ) when in. */
export function AuthNavControl() {
  const { email, isAuthed, ready, logout, role } = useAuth();
  if (!ready) return <span className={NAV_ITEM}>…</span>;
  if (!isAuthed) {
    return (
      <Link href="/login" className={NAV_ITEM}>
        Войти
      </Link>
    );
  }
  return (
    <span className="flex min-w-0 items-baseline gap-[14px] max-sm:gap-[10px]">
      {role === "admin" && (
        <Link href="/admin" className={NAV_ITEM}>
          Админ
        </Link>
      )}
      <span className="max-w-[10rem] truncate font-alt text-[9px] tracking-[0.14em] text-muted-2 max-sm:max-w-[88px]" title={email ?? undefined}>
        {email}
      </span>
      <button type="button" onClick={logout} className={`${NAV_ITEM} shrink-0 whitespace-nowrap`}>
        Выйти
      </button>
    </span>
  );
}
