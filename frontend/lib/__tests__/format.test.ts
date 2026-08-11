import { describe, expect, it } from "vitest";
import { attendanceShort, formatEndTime, formatEventRange } from "@/lib/format";

describe("formatEventRange", () => {
  it("single-day (no end) keeps the time form", () => {
    // 13 June 2026 19:00 MSK == 16:00Z
    expect(formatEventRange({ startsAt: "2026-06-13T16:00:00Z" }))
      .toBe("13 июня в 19:00");
  });

  it("same civil day (end present) still shows the start time only", () => {
    expect(
      formatEventRange({
        startsAt: "2026-06-13T16:00:00Z",
        endsAt: "2026-06-13T18:00:00Z",
      }),
    ).toBe("13 июня в 19:00");
  });

  it("multi-day within one month → '15–17 августа'", () => {
    expect(
      formatEventRange({
        startsAt: "2026-08-15T09:00:00Z",
        endsAt: "2026-08-17T09:00:00Z",
      }),
    ).toBe("15–17 августа");
  });

  it("multi-day across months → '31 июля – 2 августа'", () => {
    expect(
      formatEventRange({
        startsAt: "2026-07-31T09:00:00Z",
        endsAt: "2026-08-02T09:00:00Z",
      }),
    ).toBe("31 июля – 2 августа");
  });

  it("treats a zero-time end as unset (open-ended)", () => {
    expect(
      formatEventRange({
        startsAt: "2026-06-13T16:00:00Z",
        endsAt: "0001-01-01T00:00:00Z",
      }),
    ).toBe("13 июня в 19:00");
  });

  it("same month across years does not compress the year away", () => {
    expect(
      formatEventRange({
        startsAt: "2026-01-15T09:00:00Z",
        endsAt: "2027-01-17T09:00:00Z",
      }),
    ).toBe("15 января – 17 января");
  });
});

describe("formatEndTime", () => {
  it("returns the Moscow end time for a real end", () => {
    // 18:00Z == 21:00 MSK
    expect(formatEndTime("2026-06-13T18:00:00Z")).toBe("21:00");
  });
  it("returns null for a missing or zero-time end", () => {
    expect(formatEndTime(undefined)).toBeNull();
    expect(formatEndTime("0001-01-01T00:00:00Z")).toBeNull();
  });
});

describe("attendanceShort", () => {
  it("spells out an unset limit instead of a dash", () => {
    expect(attendanceShort({})).toBe("Без ограничения");
    expect(attendanceShort({ attendeeCount: 0 })).toBe("Без ограничения");
  });
  it("shows the bare count without a limit", () => {
    expect(attendanceShort({ attendeeCount: 64 })).toBe("64");
  });
  it("shows count / capacity when limited", () => {
    expect(attendanceShort({ attendeeCount: 12, capacity: 40 })).toBe("12 / 40");
    expect(attendanceShort({ capacity: 40 })).toBe("0 / 40");
  });
});
