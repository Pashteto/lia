"use client";

import { usePathname } from "next/navigation";
import { BottomTabBar } from "@/components/ui/BottomTabBar";

const HIDDEN_PREFIXES = [
  "/admin",
  "/organizer",
  "/events/new",
  "/events/mine",
  "/login",
  "/signup",
  "/auth",
];

/** The tab bar is user-layer navigation only (handoff → Navigation):
 * organizer/admin mobile use header context, auth screens are chromeless. */
export function TabBarGate() {
  const pathname = usePathname();
  if (HIDDEN_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`))) return null;
  if (/^\/events\/[^/]+\/edit/.test(pathname)) return null;
  return <BottomTabBar />;
}
