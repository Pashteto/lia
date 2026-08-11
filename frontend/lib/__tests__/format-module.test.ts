import { describe, expect, it } from "vitest";
import { attendanceShort, formatModuleDate, formatStartTime } from "../format";

describe("formatModuleDate", () => {
  it("single day → DD.MM · HH:mm (Moscow)", () => {
    expect(formatModuleDate("2026-07-12T13:00:00Z")).toBe("12.07 · 16:00");
  });
  it("zero end is ignored", () => {
    expect(formatModuleDate("2026-07-12T13:00:00Z", "0001-01-01T00:00:00Z")).toBe(
      "12.07 · 16:00",
    );
  });
  it("same-civil-day end keeps single form", () => {
    expect(
      formatModuleDate("2026-07-12T13:00:00Z", "2026-07-12T18:00:00Z"),
    ).toBe("12.07 · 16:00");
  });
  it("multi-day → DD.MM – DD.MM", () => {
    expect(
      formatModuleDate("2026-08-15T09:00:00Z", "2026-08-17T18:00:00Z"),
    ).toBe("15.08 – 17.08");
  });
});

describe("formatStartTime", () => {
  it("Moscow wall-clock time", () => {
    expect(formatStartTime("2026-07-12T13:00:00Z")).toBe("16:00");
  });
});

describe("attendanceShort", () => {
  it("count / capacity", () => {
    expect(attendanceShort({ attendeeCount: 12, capacity: 40 })).toBe("12 / 40");
  });
  it("count only", () => {
    expect(attendanceShort({ attendeeCount: 64 })).toBe("64");
  });
  it("no limit set → spelled out, not an em dash (design review P2)", () => {
    expect(attendanceShort({})).toBe("Без ограничения");
  });
});
