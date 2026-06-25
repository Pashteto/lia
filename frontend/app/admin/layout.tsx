"use client";

import { useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { useAuth } from "@/lib/auth-context";
import { cn } from "@/lib/cn";

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const { ready, role } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (ready && role !== "admin") router.replace("/");
  }, [ready, role, router]);

  // Avoid flicker before session/role is known
  if (!ready) return null;
  // Redirecting — render nothing while navigation lands
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
          </div>
        </nav>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-8">{children}</main>
    </div>
  );
}
