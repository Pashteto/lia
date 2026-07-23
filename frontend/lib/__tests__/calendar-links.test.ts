import { describe, expect, it } from "vitest";
import { googleCalendarUrl } from "@/lib/calendar-links";
import type { LiaEvent } from "@/lib/types";

const ev = {
  id: "1", title: "Bla Bla Meet", description: "Митап",
  categories: [], format: "offline", status: "published",
  startsAt: "2026-08-01T18:00:00Z", endsAt: "2026-08-01T20:00:00Z",
  priceType: "free",
  venue: { id: "v1", name: "Дом Радио", address: "СПб, наб. Мойки, 20" },
} as unknown as LiaEvent;

describe("googleCalendarUrl", () => {
  it("builds a render TEMPLATE link with compact UTC dates", () => {
    const u = new URL(googleCalendarUrl(ev));
    expect(u.origin + u.pathname).toBe("https://calendar.google.com/calendar/render");
    expect(u.searchParams.get("action")).toBe("TEMPLATE");
    expect(u.searchParams.get("text")).toBe("Bla Bla Meet");
    expect(u.searchParams.get("dates")).toBe("20260801T180000Z/20260801T200000Z");
    expect(u.searchParams.get("location")).toContain("Дом Радио");
  });
  it("defaults end to +2h when endsAt is missing", () => {
    const u = new URL(googleCalendarUrl({ ...ev, endsAt: undefined }));
    expect(u.searchParams.get("dates")).toBe("20260801T180000Z/20260801T200000Z");
  });
});
