import { describe, expect, it } from "vitest";
import { formatRelativeRu } from "../relative-time";

describe("formatRelativeRu", () => {
  const now = new Date("2026-07-12T15:00:00+03:00");
  it("formats hours ago", () => {
    expect(formatRelativeRu("2026-07-12T13:00:00+03:00", now)).toBe("2 ч назад");
  });
  it("formats yesterday", () => {
    expect(formatRelativeRu("2026-07-11T18:00:00+03:00", now)).toBe("вчера");
  });
});
