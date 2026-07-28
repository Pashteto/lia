import { describe, expect, it } from "vitest";
import { distanceLabel, haversineKm, radiusKmFromBounds, radiusLabel } from "../geo";

describe("haversineKm", () => {
  it("is zero for the same point", () => {
    expect(haversineKm([55.7558, 37.6173], [55.7558, 37.6173])).toBe(0);
  });
  it("matches a known Moscow distance (Kremlin → Garage ≈ 2.0 km)", () => {
    const km = haversineKm([55.752, 37.6175], [55.7351, 37.6053]);
    expect(km).toBeGreaterThan(1.8);
    expect(km).toBeLessThan(2.4);
  });
  it("is symmetric", () => {
    const a: [number, number] = [55.75, 37.61];
    const b: [number, number] = [55.8, 37.7];
    expect(haversineKm(a, b)).toBeCloseTo(haversineKm(b, a), 9);
  });
});

describe("radiusKmFromBounds", () => {
  it("measures centre → north-east corner", () => {
    // ~0.09° of latitude ≈ 10 km; a symmetric box around 55.75 N.
    const km = radiusKmFromBounds([
      [55.7, 37.5],
      [55.8, 37.7],
    ]);
    expect(km).toBeGreaterThan(6);
    expect(km).toBeLessThan(10);
  });
  it("is zero for a degenerate box", () => {
    expect(radiusKmFromBounds([[55.75, 37.61], [55.75, 37.61]])).toBe(0);
  });
});

describe("radiusLabel", () => {
  it("integers at 10 km and above", () => {
    expect(radiusLabel(12.4)).toBe("12 км");
    expect(radiusLabel(10)).toBe("10 км");
  });
  it("one decimal below 10 km", () => {
    expect(radiusLabel(4.96)).toBe("5.0 км");
    expect(radiusLabel(0.83)).toBe("0.8 км");
  });
  it("em dash when unknown", () => {
    expect(radiusLabel(null)).toBe("—");
    expect(radiusLabel(undefined)).toBe("—");
    expect(radiusLabel(Number.NaN)).toBe("—");
  });
});

describe("distanceLabel", () => {
  it("kilometres with one decimal", () => {
    expect(distanceLabel(800)).toBe("0.8 км");
    expect(distanceLabel(2100)).toBe("2.1 км");
  });
  it("null when unknown", () => {
    expect(distanceLabel(null)).toBeNull();
    expect(distanceLabel(undefined)).toBeNull();
  });
});
