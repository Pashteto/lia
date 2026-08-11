import { afterEach, describe, expect, it, vi } from "vitest";

import { MOCK_EVENTS, ssrFallbackEvents, ssrFallbackEvent } from "../mock-events";

// A failed SSR fetch must NEVER show demo data to a real visitor. Mocks are a
// local-dev convenience, gated on the API URL being unconfigured (the same
// convention the rest of the app uses for "mock mode").
describe("ssrFallbackEvents", () => {
  afterEach(() => vi.unstubAllEnvs());

  it("returns an empty feed when the API is configured (prod)", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "https://api.presence.tarski.ru");
    expect(ssrFallbackEvents()).toEqual([]);
  });

  it("returns mock events in local dev (no API URL)", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "");
    expect(ssrFallbackEvents()).toBe(MOCK_EVENTS);
  });
});

describe("ssrFallbackEvent", () => {
  afterEach(() => vi.unstubAllEnvs());

  it("returns null when the API is configured (prod)", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "https://api.presence.tarski.ru");
    expect(ssrFallbackEvent(MOCK_EVENTS[0].id)).toBeNull();
  });

  it("looks up the mock event in local dev", () => {
    vi.stubEnv("NEXT_PUBLIC_API_URL", "");
    expect(ssrFallbackEvent(MOCK_EVENTS[0].id)).toBe(MOCK_EVENTS[0]);
    expect(ssrFallbackEvent("no-such-id")).toBeNull();
  });
});
