"use client";

import { cn } from "@/lib/cn";

/** Capsule filter chip — the one intentional "pill" in the system (DESIGN.md). */
export function FilterChip({
  label,
  active = false,
  onClick,
}: {
  label: string;
  active?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        "shrink-0 rounded-capsule px-4 py-2 text-[15px] font-medium transition whitespace-nowrap active:scale-[0.96] motion-reduce:transform-none motion-reduce:transition-none",
        active
          ? "bg-accent text-white"
          : "bg-fill text-label hover:opacity-80",
      )}
    >
      {label}
    </button>
  );
}
