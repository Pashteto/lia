"use client";

import { useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { useAuth } from "@/lib/auth-context";
import { cn } from "@/lib/cn";

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const { ready, isAuthed, role, roleResolved } = useAuth();
  const router = useRouter();

  useEffect(() => {
    // State 2: no session at all — redirect immediately, no need to wait for role.
    if (ready && !isAuthed) {
      router.replace("/");
      return;
    }
    // State 4: session exists, role resolved, and it's not admin — redirect.
    if (ready && isAuthed && roleResolved && role !== "admin") {
      router.replace("/");
    }
  }, [ready, isAuthed, role, roleResolved, router]);

  // State 1: still hydrating — render nothing.
  if (!ready) return null;
  // State 2: no session — redirect in effect above; render nothing while it lands.
  if (!isAuthed) return null;
  // State 3: session exists but role fetch still in flight — show loading indicator.
  if (!roleResolved)
    return (
      <div className="flex min-h-screen items-center justify-center bg-bg-grouped">
        <p className="text-label-secondary">Загрузка…</p>
      </div>
    );
  // State 4 (non-admin): redirect in effect above; render nothing while it lands.
  if (role !== "admin") return null;

  return (
    <div className="min-h-screen bg-bg-grouped">
      {/* Floating glass nav bar */}
      <header className="sticky top-0 z-10 px-3 pt-3">
        <nav
          className={cn(
            "glass mx-auto flex max-w-5xl items-center gap-6 rounded-card px-5 py-3",
            "ring-1 ring-inset ring-black/5 dark:ring-white/10",
          )}
        >
          <span className="shrink-0 text-[17px] font-bold tracking-[-0.022em]">
            Lia Admin
          </span>
          <div className="flex items-center gap-4 text-[15px] font-medium">
            <Link
              href="/admin"
              className="text-accent transition-opacity hover:opacity-70"
            >
              Обзор
            </Link>
            <Link
              href="/admin/moderation/events"
              className="text-label-secondary transition-opacity hover:opacity-70"
            >
              Модерация событий
            </Link>
            <Link
              href="/admin/moderation/organizers"
              className="text-label-secondary transition-opacity hover:opacity-70"
            >
              Модерация организаторов
            </Link>
            <Link
              href="/admin/organizers"
              className="text-label-secondary transition-opacity hover:opacity-70"
            >
              Организаторы
            </Link>
            <Link
              href="/admin/settings"
              className="text-label-secondary transition-opacity hover:opacity-70"
            >
              Настройки
            </Link>
          </div>
        </nav>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-8">{children}</main>
    </div>
  );
}
