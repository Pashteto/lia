"use client";

// Split/join the native datetime-local string ("YYYY-MM-DDTHH:mm") the create
// form already round-trips via toDatetimeLocalValue. Kept as pure functions so
// they're unit-testable without a DOM.
export function splitLocal(value: string): { date: string; time: string } {
  const [date = "", time = ""] = value.split("T");
  return { date, time };
}
export function joinLocal(date: string, time: string): string {
  if (!date) return "";
  return `${date}T${time || "00:00"}`;
}

const cls =
  "rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none focus:ring-2 focus:ring-accent";

/**
 * Themed date + time entry with an explicit «(Мск)» label. Two native controls
 * (type=date shows a locale calendar; type=time a locale clock) avoid the
 * ambiguous dd/mm/yyyy keyboard friction of a single datetime-local, and the
 * label makes the fixed Moscow zone unmistakable. Emits the same
 * "YYYY-MM-DDTHH:mm" value the form consumes.
 */
export function DateTimeField({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  const { date, time } = splitLocal(value);
  return (
    <div className="flex flex-wrap items-center gap-2">
      <input
        type="date"
        className={cls}
        value={date}
        onChange={(e) => onChange(joinLocal(e.target.value, time))}
      />
      <input
        type="time"
        className={cls}
        value={time}
        onChange={(e) => onChange(joinLocal(date, e.target.value))}
      />
      <span className="text-[13px] text-label-secondary">Мск</span>
    </div>
  );
}
