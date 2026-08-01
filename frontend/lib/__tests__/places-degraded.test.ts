import { describe, expect, it } from "vitest";
import { mergeVenueLookups, type GeoResult } from "../geocode";

/**
 * `/places` (Yandex Places, name search) 503s in production because the key is
 * unprovisioned. The modal ran both lookups through Promise.allSettled and
 * silently kept whatever came back, so a dead name search looked identical to a
 * working one. The merge has to report the degradation.
 */
const place: GeoResult = { lat: 55.7, lon: 37.6, label: "Дом Радио" };
const addr: GeoResult = { lat: 55.8, lon: 37.5, label: "ул. Тверская, 1" };

function settled<T>(v: T): PromiseSettledResult<T> {
  return { status: "fulfilled", value: v };
}
function rejected(): PromiseSettledResult<never> {
  return { status: "rejected", reason: new Error("places failed: 503") };
}

describe("mergeVenueLookups", () => {
  it("reports name search as unavailable when /places fails", () => {
    const out = mergeVenueLookups(rejected(), settled([addr]));
    expect(out.nameSearchDown).toBe(true);
    expect(out.results).toEqual([addr]);
  });

  it("reports nothing wrong when both lookups answer", () => {
    const out = mergeVenueLookups(settled([place]), settled([addr]));
    expect(out.nameSearchDown).toBe(false);
    expect(out.results).toEqual([place, addr]);
  });

  it("does not flag a working name search that simply found nothing", () => {
    const out = mergeVenueLookups(settled([]), settled([addr]));
    expect(out.nameSearchDown).toBe(false);
  });

  it("keeps places first and drops duplicate labels", () => {
    const out = mergeVenueLookups(settled([place]), settled([{ ...place }, addr]));
    expect(out.results).toEqual([place, addr]);
  });

  it("flags the address lookup too when it is the one that failed", () => {
    const out = mergeVenueLookups(settled([place]), rejected());
    expect(out.addressSearchDown).toBe(true);
    expect(out.results).toEqual([place]);
  });
});
