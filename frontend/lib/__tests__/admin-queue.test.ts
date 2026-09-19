import { describe, expect, it } from "vitest";
import { nextQueueIndex, queueEffect, queueKeyTarget } from "../admin-queue";

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

describe("queueKeyTarget", () => {
  it("steps down and up through the queue", () => {
    expect(queueKeyTarget("ArrowDown", 0, 5)).toBe(1);
    expect(queueKeyTarget("ArrowUp", 3, 5)).toBe(2);
  });

  // Clamping, not wrapping: in a conveyor, jumping from the last event back to
  // the first reads as "the list ended" and loses the moderator's place.
  it("stops at both ends instead of wrapping", () => {
    expect(queueKeyTarget("ArrowDown", 4, 5)).toBe(null);
    expect(queueKeyTarget("ArrowUp", 0, 5)).toBe(null);
  });

  it("ignores every other key", () => {
    expect(queueKeyTarget("Enter", 1, 5)).toBe(null);
    expect(queueKeyTarget("ArrowLeft", 1, 5)).toBe(null);
    expect(queueKeyTarget("j", 1, 5)).toBe(null);
  });

  it("does nothing on an empty queue or with nothing selected", () => {
    expect(queueKeyTarget("ArrowDown", 0, 0)).toBe(null);
    expect(queueKeyTarget("ArrowDown", -1, 5)).toBe(0);
    expect(queueKeyTarget("ArrowUp", -1, 5)).toBe(null);
  });
});
