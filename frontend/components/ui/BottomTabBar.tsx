"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/cn";

const TABS = [
  { href: "/", label: "Лента" },
  { href: "/search", label: "Подбор" },
  { href: "/map", label: "Карта" },
  { href: "/me", label: "Я" },
] as const;

function isActive(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

/** Mobile-only 4-tab bar. Icons are plain squares by design — the system has
 * no icon library. Active tab: filled square + bold ink label. */
export function BottomTabBar() {
  const pathname = usePathname();
  return (
    <nav aria-label="Вкладки" className="fixed inset-x-0 bottom-0 z-40 flex border-t border-ink bg-paper sm:hidden">
      {TABS.map((tab) => {
        const active = isActive(pathname, tab.href);
        return (
          <Link
            key={tab.href}
            href={tab.href}
            aria-current={active ? "page" : undefined}
            className={cn(
              "flex min-h-[48px] flex-1 flex-col items-center justify-center gap-[4px] py-[6px] swiss-focus",
              active ? "font-bold text-ink" : "text-muted-2",
            )}
          >
            <span
              aria-hidden
              className={cn("h-[16px] w-[16px] border-[1.5px] border-current", active && "bg-current")}
            />
            <span className="font-alt text-[8px] uppercase tracking-[0.1em]">{tab.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
