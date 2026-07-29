"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";

import { AppHeader, ADMIN_NAV } from "@/components/ui/AppHeader";
import { Skeleton } from "@/components/ui/Skeleton";
import { useAuth } from "@/lib/auth-context";

function adminMobileCaption(pathname: string): string {
  if (pathname === "/admin") return "ОБЗОР";
  if (pathname.startsWith("/admin/moderation")) return "МОДЕРАЦИЯ";
  if (pathname.startsWith("/admin/organizers")) return "ОРГАНИЗАТОРЫ";
  if (pathname.startsWith("/admin/users")) return "ПОЛЬЗОВАТЕЛИ";
  if (pathname.startsWith("/admin/settings")) return "НАСТРОЙКИ";
  if (pathname.startsWith("/admin/complaints")) return "ЖАЛОБЫ";
  return "ОБЗОР";
}

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const { ready, isAuthed, role, roleResolved } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

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
  // State 3: session exists but role fetch still in flight — ink Skeleton gate.
  if (!roleResolved)
    return (
      <div data-surface="ink" className="min-h-screen bg-surface text-on-surface">
        <div className="mx-auto flex max-w-[1360px] flex-col gap-[12px] px-[20px] py-[26px]">
          <Skeleton className="h-[40px] w-full" />
          <Skeleton className="h-[120px] w-full" />
          <Skeleton className="h-[120px] w-full" />
        </div>
      </div>
    );
  // State 4 (non-admin): redirect in effect above; render nothing while it lands.
  if (role !== "admin") return null;

  return (
    <div data-surface="ink" className="min-h-screen bg-surface text-on-surface">
      <div className="mx-auto max-w-[1360px]">
        <AppHeader admin nav={ADMIN_NAV} mobileCaption={adminMobileCaption(pathname)} />
        <main>{children}</main>
      </div>
    </div>
  );
}
