import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { feedWindowStart, fetchPublishedEvents } from "../api";

/**
 * The discovery feed asks the backend for published events with no lower time
 * bound, and the list endpoint orders by starts_at ASC under a row cap — so the
 * cap gets spent on events that already happened and the feed opens on them.
 * The unfiltered feed must default to "from the start of today".
 */
describe("feedWindowStart", () => {
  it("is the start of the local day, so events already under way today stay", () => {
    const from = feedWindowStart(new Date("2026-07-30T19:45:00"));
    expect(from.getHours()).toBe(0);
    expect(from.getMinutes()).toBe(0);
    expect(from.getDate()).toBe(30);
  });
});

describe("fetchPublishedEvents", () => {
  let calledWith: string;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-07-30T19:45:00"));
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string) => {
        calledWith = url;
        return { ok: true, json: async () => [] } as unknown as Response;
      }),
    );
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("defaults to events starting today or later", async () => {
    await fetchPublishedEvents();
    const from = new URL(calledWith).searchParams.get("from");
    expect(from).toBe(feedWindowStart(new Date("2026-07-30T19:45:00")).toISOString());
  });

  it("leaves the upper bound open by default", async () => {
    await fetchPublishedEvents();
    expect(new URL(calledWith).searchParams.get("to")).toBeNull();
  });

  it("keeps an explicit window from the date chips", async () => {
    const from = new Date("2026-08-01T00:00:00Z");
    const to = new Date("2026-08-03T00:00:00Z");
    await fetchPublishedEvents(from, to);
    const params = new URL(calledWith).searchParams;
    expect(params.get("from")).toBe(from.toISOString());
    expect(params.get("to")).toBe(to.toISOString());
  });
});
