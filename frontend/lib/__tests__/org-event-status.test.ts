import { describe, expect, it } from "vitest";
import { orgEventStatusLabel } from "../org-event-status";
import { statusChipVariant } from "../status-chip";

describe("orgEventStatusLabel", () => {
  it("maps known statuses to handoff chip strings", () => {
    expect(orgEventStatusLabel("published")).toBe("Опубликовано");
    expect(orgEventStatusLabel("draft")).toBe("Черновик");
    expect(orgEventStatusLabel("pending_review")).toBe("На модерации");
  });
  it("feeds statusChipVariant correctly", () => {
    expect(statusChipVariant(orgEventStatusLabel("published"))).toBe("active");
    expect(statusChipVariant(orgEventStatusLabel("draft"))).toBe("default");
    expect(statusChipVariant(orgEventStatusLabel("pending_review"))).toBe("signal");
  });
});
