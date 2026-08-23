import type { ReactNode } from "react";

import { cn } from "@/lib/cn";

/** Horizontally scrollable chip row with a right-edge fade on mobile — the
 * affordance that there are more chips off-screen (QA-23-aug №10: rows used
 * to cut off mid-word with no scroll hint). */
export function ChipRow({
  children,
  className,
  innerClassName,
}: {
  children: ReactNode;
  className?: string;
  innerClassName?: string;
}) {
  return (
    <div className={cn("relative", className)}>
      <div
        className={cn(
          "flex items-stretch gap-[10px] overflow-x-auto px-[20px] py-[9px] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
          innerClassName,
        )}
      >
        {children}
      </div>
      <div
        data-fade
        aria-hidden
        className="pointer-events-none absolute inset-y-0 right-0 w-8 bg-gradient-to-l from-paper to-transparent sm:hidden"
      />
    </div>
  );
}
