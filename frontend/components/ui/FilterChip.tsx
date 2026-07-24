"use client";

import { cn } from "@/lib/cn";
import type React from "react";

/** Capsule filter chip — the one intentional "pill" in the system (DESIGN.md). */
export function FilterChip({
  label,
  active = false,
  onClick,
  icon,
  trailing,
  disabled = false,
  "aria-label": ariaLabel,
}: {
  label: string;
  active?: boolean;
  onClick?: () => void;
  /** Optional leading glyph (e.g. a location pin for the near-me chip). */
  icon?: React.ReactNode;
  /** Optional trailing glyph (e.g. a × to signal an active chip can be cleared). */
  trailing?: React.ReactNode;
  disabled?: boolean;
  "aria-label"?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-pressed={active}
      aria-label={ariaLabel}
      className={cn(
        "inline-flex shrink-0 items-center gap-1.5 rounded-capsule px-4 py-2 text-[15px] font-medium transition whitespace-nowrap active:scale-[0.96] disabled:opacity-60 motion-reduce:transform-none motion-reduce:transition-none",
        active ? "bg-accent text-white" : "bg-fill text-label hover:opacity-80",
      )}
    >
      {icon}
      {label}
      {trailing}
    </button>
  );
}
