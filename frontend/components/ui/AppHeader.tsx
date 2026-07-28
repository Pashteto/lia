"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/cn";
import type { ReactNode } from "react";

export interface NavItem {
  href: string;
  label: string;
}

export interface AppHeaderProps {
  nav?: NavItem[];
  /** Admin variant: wordmark PRESENCE / ADMIN, paper bottom rule (inside data-surface="ink"). */
  admin?: boolean;
  /** Mobile shows a context caption instead of nav. */
  mobileCaption?: string;
  /** Optional right-side extras (auth control) appended after nav. */
  actions?: ReactNode;
}

function matches(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

/** Swiss Grid header: wordmark left, 9px tracked uppercase nav right,
 * active item = 2px bottom rule in currentColor. */
export function AppHeader({ nav = [], admin, mobileCaption, actions }: AppHeaderProps) {
  const pathname = usePathname();
  const activeHref = nav
    .filter((item) => matches(pathname, item.href))
    .reduce<string | null>((best, item) => (best === null || item.href.length > best.length ? item.href : best), null);

  return (
    <header
      className={cn(
        "flex items-baseline justify-between border-b px-[20px] py-[13px] max-sm:px-[14px] max-sm:py-[11px]",
        admin ? "border-paper" : "border-ink",
      )}
    >
      <Link href="/" className="swiss-focus text-[13px] font-black tracking-[-0.01em] max-sm:text-[11px]">
        PRESENCE{admin ? " / ADMIN" : ""}
      </Link>
      <nav aria-label="Основная навигация" className="flex items-baseline gap-[14px] max-sm:hidden">
        {nav.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            aria-current={item.href === activeHref ? "page" : undefined}
            className={cn(
              "swiss-focus font-alt text-[9px] uppercase tracking-[0.14em]",
              item.href === activeHref && "border-b-2 border-current pb-[2px] font-bold",
            )}
          >
            {item.label}
          </Link>
        ))}
        {actions}
      </nav>
      {mobileCaption ? <span className="cap sm:hidden">{mobileCaption}</span> : null}
    </header>
  );
}

/** Canonical nav sets (handoff → Interactions → Navigation). */
export const USER_NAV: NavItem[] = [
  { href: "/", label: "События" },
  { href: "/search", label: "Подбор" },
  { href: "/me/calendar", label: "Календарь" },
  { href: "/map", label: "Карта" },
  { href: "/me", label: "Профиль" },
  { href: "/organizer", label: "Организаторам" },
];
export const ORG_NAV: NavItem[] = [
  { href: "/organizer", label: "Кабинет" },
  { href: "/events/mine", label: "Мои события" },
  { href: "/organizer/applications", label: "Заявки" },
  { href: "/me/organizer", label: "Профиль" },
];
export const ADMIN_NAV: NavItem[] = [
  { href: "/admin", label: "Обзор" },
  { href: "/admin/moderation/events", label: "Модерация" },
  { href: "/admin/organizers", label: "Организаторы" },
  { href: "/admin/users", label: "Пользователи" },
];
