import { describe, expect, it } from "vitest";
import { memberSince } from "../member-since";

describe("memberSince", () => {
  it("renders the genitive month and year", () => {
    expect(memberSince("2026-03-14T10:00:00Z")).toBe("Участник с марта 2026");
    expect(memberSince("2026-07-01T00:00:00Z")).toBe("Участник с июля 2026");
  });
  it("uses the Moscow civil month at the boundary", () => {
    // 2026-01-31T22:00Z is already 2026-02-01 01:00 in Moscow (UTC+3).
    expect(memberSince("2026-01-31T22:00:00Z")).toBe("Участник с февраля 2026");
  });
  it("returns null for missing / zero / unparseable input", () => {
    expect(memberSince(undefined)).toBeNull();
    expect(memberSince(null)).toBeNull();
    expect(memberSince("")).toBeNull();
    expect(memberSince("0001-01-01T00:00:00Z")).toBeNull();
    expect(memberSince("not a date")).toBeNull();
  });
});
