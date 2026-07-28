import { describe, expect, it } from "vitest";
import { mapAreaStats } from "../map-stats";

const EVENTS = [
  { priceType: "free" as const, startsAt: "2026-07-12T13:00:00Z" },
  { priceType: "free" as const, startsAt: "2026-07-15T16:00:00Z" },
  { priceType: "fixed" as const, startsAt: "2026-07-12T07:00:00Z" },
  // 22:00Z on the 11th is 01:00 on the 12th in Moscow — counts as "today".
  { priceType: "from" as const, startsAt: "2026-07-11T22:00:00Z" },
];

const NOW = new Date("2026-07-12T09:00:00Z");

describe("mapAreaStats", () => {
  it("counts total, free and today (Moscow civil day)", () => {
    expect(mapAreaStats(EVENTS, 5, NOW)).toEqual({
      total: "4",
      radius: "5.0 км",
      free: "2",
      today: "3",
    });
  });
  it("empty area renders zeros and an em-dash radius", () => {
    expect(mapAreaStats([], null, NOW)).toEqual({
      total: "0",
      radius: "—",
      free: "0",
      today: "0",
    });
  });
});
