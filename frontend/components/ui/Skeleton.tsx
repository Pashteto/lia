import { cn } from "@/lib/cn";

/** Loading placeholder: hairline-boxed block at final dimensions.
 * Never a spinner, never shimmer (handoff → States → Loading). */
export function Skeleton({ className }: { className?: string }) {
  return <div aria-hidden className={cn("border border-rule-inner bg-cell-blank", className)} />;
}
