import { cn } from "@/lib/cn";

/**
 * Large-title navigation as a floating Liquid Glass bar (iOS 26): a detached
 * rounded slab inset from the viewport edges, so content scrolls visibly behind
 * it. Glass is chrome only — never content cards (DESIGN.md).
 */
export function GlassNav({
  title,
  actions,
  className,
}: {
  title: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}) {
  return (
    <header className={cn("sticky top-0 z-10 px-3 pt-3", className)}>
      <div className="glass mx-auto flex max-w-5xl items-center justify-between gap-4 rounded-card px-5 py-3 ring-1 ring-inset ring-black/5 dark:ring-white/10">
        <span className="shrink-0 text-[20px] font-bold tracking-[-0.022em]">
          {title}
        </span>
        <div className="flex items-center gap-2">{actions}</div>
      </div>
    </header>
  );
}
