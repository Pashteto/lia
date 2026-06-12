"use client";

import { cn } from "@/lib/cn";

/** Apple switch (toggle). On = systemGreen success token (DESIGN.md). */
export function Switch({
  checked,
  onChange,
  id,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  id?: string;
}) {
  return (
    <button
      id={id}
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative h-[31px] w-[51px] shrink-0 rounded-capsule transition-colors",
        checked ? "bg-success" : "bg-fill",
      )}
    >
      <span
        className={cn(
          "absolute top-0.5 size-[27px] rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,0.3)] transition-transform",
          checked ? "translate-x-[22px]" : "translate-x-0.5",
        )}
      />
    </button>
  );
}
