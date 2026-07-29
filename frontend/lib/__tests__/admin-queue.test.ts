import { describe, expect, it } from "vitest";
import { nextQueueIndex } from "../admin-queue";

describe("nextQueueIndex", () => {
  it("stays on same index when a later item remains", () => {
    expect(nextQueueIndex(0, 2)).toBe(0); // removed index 0 from len 3 → len 2
  });
  it("steps back when acting on last", () => {
    expect(nextQueueIndex(2, 2)).toBe(1);
  });
  it("returns -1 when empty", () => {
    expect(nextQueueIndex(0, 0)).toBe(-1);
  });
});
