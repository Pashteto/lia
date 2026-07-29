import { describe, expect, it } from "vitest";
import { formatRelativeCompactRu } from "../admin-relative";

describe("formatRelativeCompactRu", () => {
  const now = new Date("2026-07-12T15:00:00+03:00");
  it("formats hours compact", () => {
    expect(formatRelativeCompactRu("2026-07-12T13:00:00+03:00", now)).toBe("2 ч");
  });
  it("formats days compact", () => {
    expect(formatRelativeCompactRu("2026-07-10T15:00:00+03:00", now)).toBe("2 д");
  });
});
