import { cn } from "@/lib/cn";

export interface ProgressBarProps {
  value: number;
  max: number;
  /** 5px in-row variant (O3 seat bars) instead of the 8px default. */
  thin?: boolean;
  className?: string;
}

/** Ink-bordered fill bar. No radius, no gradient, no animation. */
export function ProgressBar({ value, max, thin, className }: ProgressBarProps) {
  const pct = max > 0 ? Math.min(100, Math.max(0, (value / max) * 100)) : 0;
  return (
    <div
      role="progressbar"
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={max}
      className={cn("w-full border border-ink", thin ? "h-[5px]" : "h-[8px]", className)}
    >
      <div className="h-full bg-ink" style={{ width: `${pct}%` }} />
    </div>
  );
}
