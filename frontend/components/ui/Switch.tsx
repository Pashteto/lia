"use client";

import { cn } from "@/lib/cn";

/**
 * Apple switch (toggle). On = systemGreen success token (DESIGN.md). Off uses an
 * explicit iOS track gray (#e9e9ea / #39393d) rather than the near-invisible
 * `--fill` token, so the off state is clearly legible in both themes.
 */
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
        "relative inline-flex h-[31px] w-[51px] shrink-0 items-center rounded-full px-0.5 transition-colors",
        checked ? "bg-success" : "bg-[#e9e9ea] dark:bg-[#39393d]",
      )}
    >
      <span
        className={cn(
          "size-[27px] rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,0.3)] transition-transform",
          checked ? "translate-x-[20px]" : "translate-x-0",
        )}
      />
    </button>
  );
}
