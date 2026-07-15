import { describe, expect, it } from "vitest";
import { eventDayKeys } from "@/lib/calendar";

describe("eventDayKeys", () => {
  it("single-day → one key", () => {
    expect(eventDayKeys("2026-08-15T09:00:00Z", undefined)).toEqual(["2026-08-15"]);
  });
  it("multi-day → every civil day inclusive", () => {
    expect(eventDayKeys("2026-08-15T09:00:00Z", "2026-08-17T09:00:00Z")).toEqual([
      "2026-08-15",
      "2026-08-16",
      "2026-08-17",
    ]);
  });
  it("zero-time end → single day", () => {
    expect(eventDayKeys("2026-08-15T09:00:00Z", "0001-01-01T00:00:00Z")).toEqual([
      "2026-08-15",
    ]);
  });
});
