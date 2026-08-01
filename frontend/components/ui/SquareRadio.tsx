"use client";

import { cn } from "@/lib/cn";

/**
 * Square ink radio — the single-choice sibling of SquareCheck.
 *
 * A native `<input type="radio">` paints the platform blue, which puts colour
 * outside the token sheet. Selection here is carried by the filled ink square,
 * the same way the rest of the system marks state.
 */
export function SquareRadio({
  checked,
  onChange,
  disabled,
  name,
  value,
  label,
}: {
  checked: boolean;
  onChange: () => void;
  disabled?: boolean;
  name: string;
  value: string;
  label: string;
}) {
  return (
    <label
      className={cn(
        "flex min-h-[44px] cursor-pointer items-center gap-[10px] text-[12px]",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      <input
        type="radio"
        name={name}
        value={value}
        checked={checked}
        disabled={disabled}
        onChange={onChange}
        // Visually hidden, not display:none — the input stays focusable and
        // keyboard arrow-key group navigation keeps working.
        className="peer sr-only"
      />
      <span
        aria-hidden
        className={cn(
          "inline-flex size-[16px] shrink-0 items-center justify-center border border-ink transition-colors duration-[120ms] ease-linear",
          "peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-ink",
          checked ? "bg-ink" : "bg-transparent",
        )}
      >
        <span className={cn("size-[6px]", checked ? "bg-paper" : "bg-transparent")} />
      </span>
      {label}
    </label>
  );
}
