import { moscowDayKey } from "./calendar";
import { radiusLabel } from "./geo";
import type { LiaEvent } from "./types";

export interface MapAreaStats {
  total: string;
  radius: string;
  free: string;
  today: string;
}

type StatEvent = Pick<LiaEvent, "priceType" | "startsAt">;

/** The four U5 stat cells. All values are strings because they render in mono
 * and an unknown radius must print «—» rather than a number. */
export function mapAreaStats(
  events: ReadonlyArray<StatEvent>,
  radiusKm: number | null,
  now: Date = new Date(),
): MapAreaStats {
  const todayKey = moscowDayKey(now);
  let free = 0;
  let today = 0;
  for (const e of events) {
    if (e.priceType === "free") free += 1;
    if (moscowDayKey(new Date(e.startsAt)) === todayKey) today += 1;
  }
  return {
    total: String(events.length),
    radius: radiusLabel(radiusKm),
    free: String(free),
    today: String(today),
  };
}
