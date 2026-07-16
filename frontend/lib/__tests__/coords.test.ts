import { describe, expect, it } from "vitest";
import { toLngLat } from "@/lib/coords";

describe("toLngLat", () => {
  it("flips [lat, lon] to [lon, lat] for Yandex", () => {
    expect(toLngLat([55.7558, 37.6173])).toEqual([37.6173, 55.7558]);
  });
});
