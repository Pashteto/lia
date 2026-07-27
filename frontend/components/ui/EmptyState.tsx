import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

/** U8 pattern: name the situation, explain in one sentence, offer exactly one
 * obvious next action (two at most). Used for empty / gated / error surfaces. */
export function EmptyState({
  numeral = "00",
  title,
  text,
  actions,
  className,
}: {
  numeral?: string;
  title: string;
  text?: string;
  actions?: ReactNode; // pass ≤2 <Button>s / links
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col items-start gap-[10px] px-[20px] py-[26px]", className)}>
      <span className="font-mono text-[38px] font-bold leading-none">{numeral}</span>
      <h2 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">{title}</h2>
      {text ? <p className="max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim">{text}</p> : null}
      {actions ? <div className="mt-[6px] flex gap-[8px]">{actions}</div> : null}
    </div>
  );
}
