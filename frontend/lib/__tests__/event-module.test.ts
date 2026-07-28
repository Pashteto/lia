import { describe, expect, it } from "vitest";
import { eventToModuleProps } from "../event-module";
import type { LiaEvent } from "../types";

const CATS = [{ slug: "festival" }, { slug: "mediation" }, { slug: "lecture" }];

const EVENT = {
  id: "evt-1",
  title: "Медиация по выставке «Свет»",
  categories: [{ id: "c2", slug: "mediation", label: "Медиации" }],
  format: "online",
  status: "published",
  startsAt: "2026-07-12T13:00:00Z",
  priceType: "free",
} as LiaEvent;

describe("eventToModuleProps", () => {
  it("maps every module field", () => {
    expect(eventToModuleProps(EVENT, CATS)).toEqual({
      numeral: "02",
      category: "Медиации",
      title: "Медиация по выставке «Свет»",
      venue: "Онлайн",
      date: "12.07 · 16:00",
      price: "FREE",
      href: "/events/evt-1",
    });
  });
  it("venue name wins; online without venue → Онлайн; offline without venue → —", () => {
    expect(
      eventToModuleProps(
        { ...EVENT, venue: { id: "v", name: "Винзавод" } } as LiaEvent,
        CATS,
      ).venue,
    ).toBe("Винзавод");
    expect(eventToModuleProps({ ...EVENT, format: "online" } as LiaEvent, CATS).venue).toBe(
      "Онлайн",
    );
    expect(
      eventToModuleProps({ ...EVENT, format: "offline" } as LiaEvent, CATS).venue,
    ).toBe("—");
  });
  it("paid + from prices go through priceLabel", () => {
    expect(
      eventToModuleProps(
        { ...EVENT, priceType: "from", priceMin: 1500 } as LiaEvent,
        CATS,
      ).price,
    ).toBe(`от 1\u00a0500 ₽`);
  });
  it("no categories → numeral —, category —", () => {
    const p = eventToModuleProps({ ...EVENT, categories: [] } as LiaEvent, CATS);
    expect(p.numeral).toBe("—");
    expect(p.category).toBe("—");
  });
});
