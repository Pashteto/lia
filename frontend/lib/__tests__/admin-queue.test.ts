import { describe, expect, it } from "vitest";
import { nextQueueIndex, queueEffect } from "../admin-queue";

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

describe("queueEffect", () => {
  // The regression: a taken-down event disappeared from «Все» and came back on
  // reload, because «Все» lists published AND rejected.
  it("keeps a taken-down event in «Все», marked as rejected", () => {
    expect(queueEffect("all", "rejected")).toBe("restatus");
  });
  it("removes a taken-down event from «Ждут»", () => {
    expect(queueEffect("waiting", "rejected")).toBe("remove");
  });
  it("removes a reviewed event from every list, «Все» included", () => {
    expect(queueEffect("waiting", "reviewed")).toBe("remove");
    expect(queueEffect("all", "reviewed")).toBe("remove");
  });
  it("removes a published event from «На проверке» once approved", () => {
    expect(queueEffect("links", "published")).toBe("remove");
  });
  it("removes a reinstated event from «Все» — it is reviewed now", () => {
    expect(queueEffect("all", "published")).toBe("remove");
  });
});
