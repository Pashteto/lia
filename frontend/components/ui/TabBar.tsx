"use client";

import { cn } from "@/lib/cn";
import Link from "next/link";
import { usePathname } from "next/navigation";

interface Tab {
  href: string;
  label: string;
  icon: React.ReactNode;
}

const TABS: Tab[] = [
  { href: "/", label: "События", icon: <GlyphGrid /> },
  { href: "/search", label: "Подбор", icon: <GlyphSparkle /> },
  { href: "/me/calendar", label: "Календарь", icon: <GlyphCalendar /> },
  { href: "/map", label: "Карта", icon: <GlyphMap /> },
  { href: "/events/new", label: "Создать", icon: <GlyphPlus /> },
];

/** Mobile floating Liquid Glass tab bar — a detached capsule above the
 * safe-area inset (iOS 26 chrome). The wrapper is click-through; only the
 * capsule itself is interactive. */
export function TabBar() {
  const pathname = usePathname();
  // Hidden where the screen has its own bottom chrome or dedicated nav: the
  // create form, an event detail page (own sticky signup CTA), and the admin
  // section (own glass nav). Shown on all other top-level / list screens.
  const seg = pathname.match(/^\/events\/([^/]+)$/)?.[1];
  const isEventDetail = !!seg && seg !== "mine" && seg !== "new";
  if (pathname === "/events/new" || isEventDetail || pathname.startsWith("/admin")) {
    return null;
  }
  return (
    <nav className="pointer-events-none fixed inset-x-0 bottom-0 z-10 px-4 pb-[max(env(safe-area-inset-bottom),12px)] pt-2 sm:hidden">
      <ul className="glass pointer-events-auto mx-auto flex max-w-md items-stretch justify-around rounded-capsule px-2 ring-1 ring-inset ring-black/5 dark:ring-white/10">
        {TABS.map((tab) => {
          const active = pathname === tab.href;
          return (
            <li key={tab.href} className="flex-1">
              <Link
                href={tab.href}
                className={cn(
                  "flex flex-col items-center gap-1 py-2.5 text-[10px] font-medium transition-colors",
                  active ? "text-accent" : "text-label-secondary",
                )}
              >
                {tab.icon}
                {tab.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

function GlyphGrid() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
      <rect x="3" y="3" width="7" height="7" rx="2" />
      <rect x="14" y="3" width="7" height="7" rx="2" />
      <rect x="3" y="14" width="7" height="7" rx="2" />
      <rect x="14" y="14" width="7" height="7" rx="2" />
    </svg>
  );
}
function GlyphSparkle() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
      <path d="M12 2l1.8 5.2L19 9l-5.2 1.8L12 16l-1.8-5.2L5 9l5.2-1.8z" />
    </svg>
  );
}
function GlyphPlus() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M12 5v14M5 12h14"
        stroke="currentColor"
        strokeWidth="2.2"
        strokeLinecap="round"
      />
    </svg>
  );
}
function GlyphCalendar() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <rect x="3" y="4.5" width="18" height="16" rx="3" stroke="currentColor" strokeWidth="1.8" />
      <path d="M3 9h18M8 3v3M16 3v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  );
}
function GlyphMap() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4M9 7l6-3"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
