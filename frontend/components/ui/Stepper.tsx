import { cn } from "@/lib/cn";

/** O2/O5 stepper: equal-width segments; completed + active are ink-filled
 * with paper text, upcoming stay on paper; 1px rules divide. */
export function Stepper({
  steps,
  current,
  fillMode = "inclusive",
  className,
}: {
  steps: string[];
  current: number;
  fillMode?: "inclusive" | "exclusive";
  className?: string;
}) {
  const filled = (i: number) =>
    fillMode === "exclusive" ? i < current : i <= current;

  return (
    <ol className={cn("grid border-y border-on-surface [&>*+*]:border-l [&>*+*]:border-on-surface", className)} style={{ gridTemplateColumns: `repeat(${steps.length}, 1fr)` }}>
      {steps.map((step, i) => (
        <li
          key={step}
          aria-current={i === current ? "step" : undefined}
          className={cn("flex min-w-0 flex-col gap-[4px] px-[14px] py-[10px]", filled(i) && "bg-on-surface text-surface")}
        >
          <span className={cn("cap", filled(i) && "text-text-dim-dark-2")}>
            {String(i + 1).padStart(2, "0")}
          </span>
          <span className="truncate text-[12px] font-bold">{step}</span>
        </li>
      ))}
    </ol>
  );
}
