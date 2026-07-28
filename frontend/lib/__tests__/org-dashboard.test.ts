import { describe, expect, it } from "vitest";
import { nextOrganizerEvent, orgDashboardStats } from "../org-dashboard";

describe("orgDashboardStats", () => {
  it("counts by status and sums measurable seats", () => {
    const s = orgDashboardStats([
      { status: "published", capacity: 40, seatsRemaining: 12 },
      { status: "published", capacity: 10, seatsRemaining: 10 },
      { status: "pending_review" },
      { status: "draft" },
      { status: "draft" },
    ]);
    expect(s).toEqual({
      published: 2,
      pendingReview: 1,
      drafts: 2,
      totalRegistrations: 28, // 28 + 0
    });
  });
  it("returns null totalRegistrations when no seats measurable", () => {
    expect(orgDashboardStats([{ status: "published" }]).totalRegistrations).toBeNull();
  });
});

describe("nextOrganizerEvent", () => {
  const now = new Date("2026-07-10T12:00:00Z");
  it("picks the soonest future start", () => {
    const n = nextOrganizerEvent(
      [
        { id: "a", startsAt: "2026-07-12T13:00:00Z" },
        { id: "b", startsAt: "2026-07-11T10:00:00Z" },
        { id: "c", startsAt: "2026-07-01T10:00:00Z" },
      ],
      now,
    );
    expect(n?.id).toBe("b");
  });
  it("returns null when all past", () => {
    expect(nextOrganizerEvent([{ startsAt: "2026-07-01T10:00:00Z" }], now)).toBeNull();
  });
});
