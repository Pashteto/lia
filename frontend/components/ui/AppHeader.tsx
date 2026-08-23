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
  /** Mobile shows a context caption instead of nav. A string gets the cap
   * style; a ReactNode (e.g. the city control) renders as-is. */
  mobileCaption?: ReactNode;
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
      <Link
        href={admin ? "/admin" : "/"}
        className="swiss-focus shrink-0 pr-[10px] text-[13px] font-black tracking-[-0.01em] max-sm:text-[11px]"
      >
        {admin ? (
          <>
            <span className="sm:hidden">ADMIN</span>
            <span className="hidden sm:inline">PRESENCE / ADMIN</span>
          </>
        ) : (
          "PRESENCE"
        )}
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
      {(mobileCaption || actions) ? (
        <div className="flex min-w-0 items-baseline justify-end gap-[10px] sm:hidden">
          {typeof mobileCaption === "string" ? (
            <span className="cap whitespace-nowrap">{mobileCaption}</span>
          ) : (
            (mobileCaption ?? null)
          )}
          {actions}
        </div>
      ) : null}
    </header>
  );
}

/** Mobile escape hatch for organizer-area headers: the bottom tab bar is
 * user-layer only, so without this the only way back was the wordmark
 * (QA 14.08, finding 8). Renders in the header's actions slot. */
export function BackToFeedLink() {
  return (
    <Link href="/" className="cap swiss-focus hover-invert">
      ← Лента
    </Link>
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
  { href: "/admin/complaints", label: "Жалобы" },
  { href: "/admin/settings", label: "Настройки" },
];
