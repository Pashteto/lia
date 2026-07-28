import { cn } from "@/lib/cn";

/** O2/O5 stepper: equal-width segments. Inclusive (O2): filled through current
 * (active ink). Exclusive (O5): filled before current (current stays on paper).
 * Adjacent filled segments use a paper divider so ink|ink edges stay visible. */
export function Stepper({
  steps,
  current,
  fillMode = "inclusive",
  numeralPrefix,
  className,
}: {
  steps: string[];
  current: number;
  fillMode?: "inclusive" | "exclusive";
  /** O2 mock captions read «Шаг 01»; omit for O5. */
  numeralPrefix?: string;
  className?: string;
}) {
  const filled = (i: number) =>
    fillMode === "exclusive" ? i < current : i <= current;

  return (
    <ol
      className={cn("grid border-y border-on-surface", className)}
      style={{ gridTemplateColumns: `repeat(${steps.length}, 1fr)` }}
    >
      {steps.map((step, i) => (
        <li
          key={step}
          aria-current={i === current ? "step" : undefined}
          className={cn(
            "flex min-w-0 flex-col gap-[4px] px-[14px] py-[10px]",
            i > 0 && (filled(i) && filled(i - 1) ? "border-l border-paper" : "border-l border-on-surface"),
            filled(i) && "bg-on-surface text-surface",
          )}
        >
          <span className={cn("cap", filled(i) && "text-text-dim-dark-2")}>
            {numeralPrefix ?? ""}
            {String(i + 1).padStart(2, "0")}
          </span>
          <span className="truncate text-[12px] font-bold">{step}</span>
        </li>
      ))}
    </ol>
  );
}
