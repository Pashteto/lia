"use client";

import { cn } from "@/lib/cn";

/**
 * Square ink checkbox. The token sheet has no colour outside it, so a native
 * `<input type="checkbox">` — which paints the platform blue — is not usable on
 * any Presence surface. This is the same hairline square the rest of the system
 * is built from: an empty box that fills with paper (or ink, off the admin
 * surface) and carries the mark when on.
 */
export function SquareCheck({
  checked,
  onChange,
  disabled,
  id,
  "aria-labelledby": labelledBy,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  id?: string;
  "aria-labelledby"?: string;
}) {
  return (
    <button
      id={id}
      type="button"
      role="checkbox"
      aria-checked={checked}
      aria-labelledby={labelledBy}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "swiss-focus inline-flex size-[18px] shrink-0 items-center justify-center border transition-colors duration-[120ms] ease-linear",
        checked
          ? "border-paper bg-paper text-ink"
          : "border-muted-2 bg-transparent text-transparent hover:border-paper",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      <span aria-hidden className="text-[11px] font-black leading-none">
        ✓
      </span>
    </button>
  );
}
