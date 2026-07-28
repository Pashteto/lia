import { describe, expect, it } from "vitest";

import { eventToDuplicateDraftInput } from "../org-duplicate-event";
import type { LiaEvent } from "../types";

function baseEvent(overrides: Partial<LiaEvent> = {}): LiaEvent {
  return {
    id: "src",
    title: "Медиация",
    description: "Описание",
    categories: [{ id: "c1", slug: "mediation", label: "Медиации" }],
    format: "offline",
    status: "published",
    startsAt: "2026-07-12T13:00:00Z",
    endsAt: "2026-07-12T15:00:00Z",
    priceType: "from",
    priceMin: 500,
    capacity: 40,
    signupMode: "application",
    curatorQuestion: "Почему?",
    venue: { id: "v1", name: "Гараж" },
    ...overrides,
  };
}

describe("eventToDuplicateDraftInput", () => {
  it("copies fields and forces draft status", () => {
    expect(eventToDuplicateDraftInput(baseEvent())).toEqual({
      title: "Медиация",
      description: "Описание",
      category_ids: ["c1"],
      venue_id: "v1",
      status: "draft",
      format: "offline",
      price_type: "from",
      price_min: 500,
      starts_at: "2026-07-12T13:00:00Z",
      ends_at: "2026-07-12T15:00:00Z",
      signup_mode: "application",
      capacity: 40,
      curator_question: "Почему?",
      external_registration_url: undefined,
    });
  });

  it("drops zero-time ends_at and free price_min", () => {
    const input = eventToDuplicateDraftInput(
      baseEvent({
        priceType: "free",
        priceMin: 0,
        endsAt: "0001-01-01T00:00:00Z",
        categories: [],
        venue: undefined,
        signupMode: "open",
        curatorQuestion: undefined,
      }),
    );
    expect(input.status).toBe("draft");
    expect(input.price_min).toBeUndefined();
    expect(input.ends_at).toBeUndefined();
    expect(input.category_ids).toBeUndefined();
    expect(input.venue_id).toBeUndefined();
  });
});
