"use client";

import { cn } from "@/lib/cn";

export interface SegmentedOption<T extends string> {
  value: T;
  label: string;
}

/** Apple segmented control — for mutually-exclusive options (DESIGN.md). */
export function Segmented<T extends string>({
  options,
  value,
  onChange,
  className,
  disabled,
}: {
  options: SegmentedOption<T>[];
  value: T;
  onChange: (value: T) => void;
  className?: string;
  /** Renders the control read-only (locked), e.g. signup mode after publication. */
  disabled?: boolean;
}) {
  return (
    <div
      className={cn(
        "inline-flex w-full rounded-seg bg-fill p-0.5",
        disabled && "opacity-60",
        className,
      )}
      role="tablist"
    >
      {options.map((opt) => {
        const active = opt.value === value;
        return (
          <button
            key={opt.value}
            type="button"
            role="tab"
            aria-selected={active}
            disabled={disabled}
            onClick={() => onChange(opt.value)}
            className={cn(
              "flex-1 rounded-[7px] px-3 py-1.5 text-[14px] font-medium transition",
              active
                ? "bg-bg-secondary text-label shadow-card-subtle"
                : "text-label-secondary hover:text-label",
              disabled && "cursor-not-allowed",
            )}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}
