import { cn } from "@/lib/cn";

/**
 * Large-title navigation bar on Liquid Glass (desktop chrome).
 * Glass is applied to chrome only — never to content cards (DESIGN.md).
 */
export function GlassNav({
  title,
  actions,
  className,
}: {
  title: string;
  actions?: React.ReactNode;
  className?: string;
}) {
  return (
    <header
      className={cn(
        "glass sticky top-0 z-10 border-b border-separator",
        className,
      )}
    >
      <div className="mx-auto flex max-w-3xl items-center justify-between px-5 py-3">
        <span className="text-[20px] font-bold tracking-[-0.022em]">{title}</span>
        <div className="flex items-center gap-2">{actions}</div>
      </div>
    </header>
  );
}
