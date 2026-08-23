import { describe, expect, it } from "vitest";
import { apiEventToLia } from "@/lib/api";
import type { ApiEvent } from "@/lib/types";

const base: ApiEvent = {
  id: "x",
  title: "t",
  status: "published",
  starts_at: "2026-09-30T19:00:00Z",
} as ApiEvent;

describe("apiEventToLia owner fields", () => {
  it("maps is_owner and pending_applications_count", () => {
    const ev = apiEventToLia({
      ...base,
      is_owner: true,
      pending_applications_count: 3,
    });
    expect(ev.isOwner).toBe(true);
    expect(ev.pendingApplicationsCount).toBe(3);
  });

  it("defaults to false/0 when absent", () => {
    const ev = apiEventToLia(base);
    expect(ev.isOwner).toBe(false);
    expect(ev.pendingApplicationsCount).toBe(0);
  });
});
