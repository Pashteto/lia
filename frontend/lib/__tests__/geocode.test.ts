import { describe, expect, it, vi, beforeEach } from "vitest";

vi.mock("@/lib/auth", () => ({ getToken: () => "test-token" }));

import { geocodeAddress, searchPlaces } from "@/lib/geocode";

describe("geocodeAddress", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("returns [] for a blank query without calling fetch", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    expect(await geocodeAddress("   ")).toEqual([]);
    expect(f).not.toHaveBeenCalled();
  });

  it("sends a bearer token and returns the backend results", async () => {
    const f = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify([{ lat: 55.7, lon: 37.6, label: "Москва" }]), {
        status: 200,
      }),
    );
    const out = await geocodeAddress("Москва");
    expect(out).toEqual([{ lat: 55.7, lon: 37.6, label: "Москва" }]);
    const [url, init] = f.mock.calls[0];
    expect(String(url)).toContain("/api/v1/geocode?q=");
    expect((init as RequestInit).headers).toMatchObject({
      Authorization: "Bearer test-token",
    });
  });
});

describe("searchPlaces", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("returns [] for a blank query without calling fetch", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    expect(await searchPlaces("   ")).toEqual([]);
    expect(f).not.toHaveBeenCalled();
  });

  it("sends a bearer token and returns the backend results", async () => {
    const f = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify([{ lat: 55.75, lon: 37.62, label: "Дом Радио" }]), {
        status: 200,
      }),
    );
    const out = await searchPlaces("Дом Радио");
    expect(out).toEqual([{ lat: 55.75, lon: 37.62, label: "Дом Радио" }]);
    const [url, init] = f.mock.calls[0];
    expect(String(url)).toContain("/api/v1/places?q=");
    expect((init as RequestInit).headers).toMatchObject({
      Authorization: "Bearer test-token",
    });
  });
});
