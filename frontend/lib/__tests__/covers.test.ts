import { describe, expect, it } from "vitest";
import { coverPhoto } from "../covers";
import type { LiaEvent } from "../types";

const SEED_ID = "b0000000-0000-0000-0000-000000000007";
const REAL_ID = "b4e0d8a3-9a8a-4f10-9d6a-1c2b3d4e5f60";

function event(over: Partial<LiaEvent> = {}): LiaEvent {
  return {
    id: SEED_ID,
    title: "Летний фестиваль медиаискусства",
    categories: [{ id: "c1", slug: "festival", label: "Фестивали" }],
    format: "offline",
    status: "published",
    startsAt: "2026-07-25T15:00:00Z",
    priceType: "free",
    ...over,
  } as LiaEvent;
}

describe("coverPhoto", () => {
  it("an uploaded cover always wins", () => {
    expect(
      coverPhoto(event({ id: REAL_ID, coverUrl: "https://api.example/f/abc" })),
    ).toBe("https://api.example/f/abc");
  });

  it("an uploaded cover wins even on a seeded event", () => {
    expect(coverPhoto(event({ coverUrl: "https://api.example/f/xyz" }))).toBe(
      "https://api.example/f/xyz",
    );
  });

  it("a seeded event falls back to its category photo", () => {
    expect(coverPhoto(event())).toBe("/covers/festival.jpg");
    expect(
      coverPhoto(
        event({ categories: [{ id: "c2", slug: "mediation", label: "Медиации" }] }),
      ),
    ).toBe("/covers/mediation.jpg");
  });

  it("a non-seeded event without an upload has no photo", () => {
    expect(coverPhoto(event({ id: REAL_ID }))).toBeUndefined();
  });

  it("a seeded event with no category has no photo", () => {
    expect(coverPhoto(event({ categories: [] }))).toBeUndefined();
  });

  it("a seeded event in an unmapped category has no photo", () => {
    expect(
      coverPhoto(event({ categories: [{ id: "c9", slug: "reading", label: "Читательские" }] })),
    ).toBeUndefined();
  });
});
