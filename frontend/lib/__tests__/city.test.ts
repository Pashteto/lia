import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

import {
  CITIES,
  CITY_COOKIE,
  CURRENT_CITY,
  cityBySlug,
  cityTagline,
  cityLoginCaption,
} from "@/lib/city";

describe("lib/city", () => {
  it("Moscow is the single available city, with grammar and map center", () => {
    expect(CURRENT_CITY.name).toBe("Москва");
    expect(CURRENT_CITY.genitive).toBe("Москвы");
    // Same default the /map browse view has always used.
    expect(CURRENT_CITY.center).toEqual([55.742, 37.618]);
    expect(CITIES.filter((c) => c.available)).toHaveLength(1);
  });

  it("derives feed tagline and login caption from the city's genitive", () => {
    expect(cityTagline(CURRENT_CITY)).toContain("культурной Москвы");
    expect(cityLoginCaption(CURRENT_CITY)).toBe("События Москвы");
    const spb = CITIES.find((c) => c.code === "СПБ")!;
    expect(cityTagline(spb)).toContain("культурного Петербурга");
    expect(cityLoginCaption(spb)).toBe("События Петербурга");
  });
});

describe("cityBySlug", () => {
  it("resolves known slugs (cookie / ?city= values)", () => {
    expect(cityBySlug("spb").code).toBe("СПБ");
    expect(cityBySlug("msk").code).toBe("МСК");
  });

  it("falls back to the default city for garbage, null and undefined", () => {
    for (const bad of ["", "СПБ", "ekb", undefined, null]) {
      expect(cityBySlug(bad).slug).toBe("msk");
    }
  });

  it("slugs are backend-compatible latin lowercase and the cookie has a name", () => {
    expect(CITIES.map((c) => c.slug)).toEqual(["msk", "spb"]);
    expect(CITY_COOKIE).toBe("lia_city");
  });
});

describe("city copy is not hardcoded anymore", () => {
  it.each([
    "../../components/DiscoveryFeed.tsx",
    "../../app/login/page.tsx",
    "../../components/ui/CityControl.tsx",
  ])("%s carries no literal «Москв…»", (rel) => {
    const src = readFileSync(join(__dirname, rel), "utf8");
    expect(src).not.toMatch(/Москв/);
    expect(src).toContain('from "@/lib/city"');
  });
});
