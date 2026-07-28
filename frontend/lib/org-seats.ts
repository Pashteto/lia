export function seatsFill(
  event: { capacity?: number | null; seatsRemaining?: number | null },
): { filled: number; capacity: number; label: string; ratio: number } | null {
  const { capacity, seatsRemaining } = event;
  if (capacity == null || seatsRemaining == null) return null;

  const filled = capacity - seatsRemaining;
  const ratio = Math.min(1, Math.max(0, filled / capacity));
  return {
    filled,
    capacity,
    label: `${filled} / ${capacity}`,
    ratio,
  };
}

export function padCount(n: number, width = 2): string {
  const floored = Math.floor(n);
  if (floored < 0) return "0".repeat(width);
  return String(floored).padStart(width, "0");
}
