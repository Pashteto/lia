import { cn } from "@/lib/cn";

/** Uppercase accent caption — category kickers / section labels (DESIGN.md). */
export function Kicker({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "text-[12px] font-semibold uppercase tracking-[0.03em] text-accent",
        className,
      )}
    >
      {children}
    </span>
  );
}
