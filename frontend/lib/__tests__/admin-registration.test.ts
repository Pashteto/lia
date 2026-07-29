import { describe, expect, it } from "vitest";
import { formatRegistrationMonth } from "@/lib/admin-registration";

describe("formatRegistrationMonth", () => {
  it("renders MM.YYYY", () => {
    expect(formatRegistrationMonth("2026-03-04T10:00:00Z")).toBe("03.2026");
  });

  it("uses Moscow day boundaries", () => {
    // 2025-12-31T22:00Z is 2026-01-01T01:00 in Moscow.
    expect(formatRegistrationMonth("2025-12-31T22:00:00Z")).toBe("01.2026");
  });

  it("returns an em dash for unknown or zero timestamps", () => {
    expect(formatRegistrationMonth(null)).toBe("—");
    expect(formatRegistrationMonth("")).toBe("—");
    expect(formatRegistrationMonth("not a date")).toBe("—");
    expect(formatRegistrationMonth("0001-01-01T00:00:00Z")).toBe("—");
  });
});
