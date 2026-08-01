import { describe, expect, it } from "vitest";
import { apiEventToLia } from "../api";
import type { ApiEvent } from "../types";

/**
 * The backend exposes `seats_remaining` (= capacity − going, clamped at 0) but
 * never an attendee count, while the U2 «Места» cell renders `attendeeCount`.
 * The adapter has to derive one from the other, or the cell reads "—" on every
 * event while the sidebar right beside it shows the seats left.
 */
function base(over: Partial<ApiEvent> = {}): ApiEvent {
  return {
    id: "e1",
    title: "Событие",
    status: "published",
    starts_at: "2026-08-15T09:00:00Z",
    ...over,
  };
}

describe("apiEventToLia — attendeeCount", () => {
  it("derives the attendee count from capacity and seats left", () => {
    expect(apiEventToLia(base({ capacity: 12, seats_remaining: 1 })).attendeeCount).toBe(11);
  });

  it("is 0 when no one has signed up yet", () => {
    expect(apiEventToLia(base({ capacity: 20, seats_remaining: 20 })).attendeeCount).toBe(0);
  });

  it("stays undefined without a capacity (unlimited event)", () => {
    expect(apiEventToLia(base()).attendeeCount).toBeUndefined();
  });

  it("stays undefined when the backend omitted seats_remaining", () => {
    expect(apiEventToLia(base({ capacity: 20 })).attendeeCount).toBeUndefined();
  });

  it("never goes negative if seats left exceeds capacity", () => {
    expect(apiEventToLia(base({ capacity: 5, seats_remaining: 8 })).attendeeCount).toBe(0);
  });
});
