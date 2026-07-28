import { cn } from "@/lib/cn";
import type { ButtonHTMLAttributes } from "react";

export type ChipVariant =
  | "default" // 1px currentColor border, transparent
  | "active" // ink fill, paper text
  | "signal" // red border + text
  | "dark-active" // paper fill, ink text (on ink surface)
  | "dark-muted"; // muted border + text (on ink surface)

// On data-surface="ink", ink-filled "active" is invisible — callers use "dark-active"/"dark-muted" there.
const VARIANTS: Record<ChipVariant, string> = {
  default: "border-current",
  active: "border-ink bg-ink text-white",
  signal: "border-signal text-signal",
  "dark-active": "border-paper bg-paper text-ink",
  "dark-muted": "border-muted-2 text-muted-2",
};

export interface ChipProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ChipVariant;
  /** Render a non-interactive span (status display) instead of a button. */
  as?: "button" | "span";
}

/** Swiss Grid chip: 9px / 0.12em / uppercase / 1px border / zero radius.
 * Counters live inside the label: `Все · 6`. Interactive chips hover-invert. */
export function Chip({ variant = "default", as = "button", className, children, ...props }: ChipProps) {
  const base = cn(
    "inline-flex items-center whitespace-nowrap border px-[9px] py-[4px] text-[9px] uppercase tracking-[0.12em]",
    VARIANTS[variant],
    as === "button" && "cursor-pointer swiss-focus hover-invert",
    className,
  );
  if (as === "span") return <span className={base} {...(props as React.HTMLAttributes<HTMLSpanElement>)}>{children}</span>;
  return (
    <button type="button" className={base} {...props}>
      {children}
    </button>
  );
}
