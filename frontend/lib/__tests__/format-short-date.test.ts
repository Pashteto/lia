import { describe, expect, it } from "vitest";
import { formatShortDate } from "../format";

describe("formatShortDate", () => {
  it("renders DD.MM in Moscow", () => {
    expect(formatShortDate("2026-07-12T13:00:00Z")).toBe("12.07");
  });
  it("uses the Moscow civil day across the UTC midnight boundary", () => {
    // 22:00Z on the 11th is already 01:00 on the 12th in Moscow.
    expect(formatShortDate("2026-07-11T22:00:00Z")).toBe("12.07");
  });
});
