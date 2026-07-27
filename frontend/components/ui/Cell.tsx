import { cn } from "@/lib/cn";
import type { ReactNode } from "react";

export interface CellProps {
  caption: ReactNode;
  value: ReactNode;
  /** Numeric value → JetBrains Mono (handoff: ALL numbers in mono). */
  mono?: boolean;
  /** Roomy 16/20 padding instead of dense 10/14. */
  roomy?: boolean;
  /** Ink-filled emphasis cell (e.g. O1 «На модерации»). */
  invert?: boolean;
  className?: string;
}

/** The atomic module of the system: uppercase caption over a bold value. */
export function Cell({ caption, value, mono, roomy, invert, className }: CellProps) {
  return (
    <div
      className={cn(
        "flex min-w-0 flex-col gap-[4px]",
        roomy ? "px-[20px] py-[16px]" : "px-[14px] py-[10px]",
        invert && "bg-ink text-paper",
        className,
      )}
    >
      <span className={cn("cap", invert && "text-text-dim-dark-2")}>{caption}</span>
      <span className={cn("text-[12px] font-bold leading-[1.25]", mono && "font-mono")}>
        {value}
      </span>
    </div>
  );
}

/** Horizontal strip of cells divided by hairlines, closed by a bottom rule. */
export function CellStrip({ children, cols, className }: { children: ReactNode; cols: number; className?: string }) {
  return (
    <div
      className={cn("grid border-b border-on-surface [&>*+*]:border-l [&>*+*]:border-rule-inner", className)}
      style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}
    >
      {children}
    </div>
  );
}
