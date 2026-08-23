import { describe, expect, it } from "vitest";
import { adjustedFollowers } from "@/lib/follow-count";

describe("adjustedFollowers", () => {
  it("optimistic +1 when following overrides server false", () => {
    expect(adjustedFollowers(0, false, true)).toBe(1);
    expect(adjustedFollowers(7, false, true)).toBe(8);
  });

  it("optimistic -1 floors at 0", () => {
    expect(adjustedFollowers(0, true, false)).toBe(0);
    expect(adjustedFollowers(3, true, false)).toBe(2);
  });

  it("no override or override equals server → base", () => {
    expect(adjustedFollowers(5, true, null)).toBe(5);
    expect(adjustedFollowers(5, true, true)).toBe(5);
    expect(adjustedFollowers(undefined, false, null)).toBe(0);
  });
});
