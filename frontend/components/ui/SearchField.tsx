import { cn } from "@/lib/cn";
import type { InputHTMLAttributes } from "react";

/** Apple search field: `fill` background, rounded, leading glyph. */
export function SearchField({
  className,
  ...props
}: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-control bg-fill px-3.5 py-2.5",
        className,
      )}
    >
      <svg
        width="17"
        height="17"
        viewBox="0 0 24 24"
        fill="none"
        className="shrink-0 text-label-secondary"
        aria-hidden
      >
        <circle cx="11" cy="11" r="7" stroke="currentColor" strokeWidth="2" />
        <path
          d="m20 20-3-3"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
        />
      </svg>
      <input
        type="search"
        className="w-full bg-transparent text-[17px] text-label outline-none placeholder:text-label-secondary"
        {...props}
      />
    </div>
  );
}
