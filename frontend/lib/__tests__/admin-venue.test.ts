import { describe, expect, it } from "vitest";

import { venueSummary } from "@/lib/admin-venue";

describe("venueSummary", () => {
  it("joins name, address and metro into one line", () => {
    expect(
      venueSummary({
        venue: { name: "Галерея «Сети»", address: "наб. реки Фонтанки, 24", metro: "Гостиный двор" },
        city: "spb",
      }),
    ).toEqual({
      text: "Галерея «Сети» · наб. реки Фонтанки, 24 · Гостиный двор",
      city: "Санкт-Петербург",
      missing: false,
    });
  });

  it("drops the parts a venue does not have", () => {
    expect(venueSummary({ venue: { name: "Дом культуры" }, city: "msk" }).text).toBe("Дом культуры");
  });

  // An online event has no venue by design — that is not a defect to flag.
  it("says so for an online event and never calls it missing", () => {
    const s = venueSummary({ venue: { name: "Галерея «Сети»" }, format: "online", city: "spb" });
    expect(s.text).toBe("Онлайн, без площадки");
    expect(s.missing).toBe(false);
  });

  // A missing venue IS a defect, and the moderator deciding on the event is
  // exactly the person who should see it.
  it("flags an offline event with no venue", () => {
    expect(venueSummary({ city: "msk" })).toEqual({
      text: "Площадка не указана",
      city: "Москва",
      missing: true,
    });
    expect(venueSummary({ venue: { name: "   " }, city: "msk" }).missing).toBe(true);
  });

  // cityBySlug falls back to Moscow for garbage so the public site cannot 500.
  // Here that fallback would quietly put a Moscow label on an event nobody
  // assigned a city to — the moderator must see the gap, not an invention.
  it("shows no city rather than inventing one", () => {
    expect(venueSummary({ venue: { name: "Дом культуры" } }).city).toBe("");
    expect(venueSummary({ venue: { name: "Дом культуры" }, city: "kzn" }).city).toBe("");
  });
});
