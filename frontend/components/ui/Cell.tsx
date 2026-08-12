import Link from "next/link";
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
  /** Navigation target. Present ⇒ the cell renders as a link with a « →»
   * caption suffix and hover inversion — the product-wide affordance for
   * "this tile is a button" (spec 2026-08-12-clickable-dashboard-tiles). */
  href?: string;
  /** Per-screen size override on the value; the reference sets it inline
   * (U6 15px, U5 mobile 11px). Merged last, so `text-[15px]` wins over 12px. */
  valueClassName?: string;
  className?: string;
}

/** The atomic module of the system: uppercase caption over a bold value. */
export function Cell({ caption, value, mono, roomy, invert, href, valueClassName, className }: CellProps) {
  const inner = (
    <>
      <span className={cn("cap", invert && "text-text-dim-dark-2")}>
        {caption}
        {href ? " →" : null}
      </span>
      <span className={cn("text-[12px] font-bold leading-[1.25]", mono && "font-mono", valueClassName)}>
        {value}
      </span>
    </>
  );
  const shared = cn(
    "flex min-w-0 flex-col gap-[4px]",
    roomy ? "px-[20px] py-[16px]" : "px-[14px] py-[10px]",
    invert && "bg-on-surface text-surface",
    className,
  );
  if (href) {
    return (
      <Link href={href} className={cn(shared, "swiss-focus hover-invert cursor-pointer")}>
        {inner}
      </Link>
    );
  }
  return <div className={shared}>{inner}</div>;
}

/** Horizontal strip of cells divided by hairlines, closed by a bottom rule.
 * Dividers are structural (#111 on paper / #F2F0EC on ink), matching the
 * reference's `.r` class — not the lighter inner-separation rule. */
export function CellStrip({ children, cols, className }: { children: ReactNode; cols: number; className?: string }) {
  return (
    <div
      className={cn("grid border-b border-on-surface [&>*+*]:border-l [&>*+*]:border-on-surface", className)}
      style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}
    >
      {children}
    </div>
  );
}
