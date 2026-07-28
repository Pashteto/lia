import { afterEach, describe, expect, it, vi } from "vitest";

import { createTimedResolver } from "@/lib/timed-resolver";
import {
  recenterMapPreservingZoom,
  updateActivePinLayouts,
} from "@/components/map/yandex-map-controls";

afterEach(() => {
  vi.useRealTimers();
});

describe("createTimedResolver", () => {
  it("falls back when geolocation never resolves", () => {
    vi.useFakeTimers();
    const fallback = vi.fn();
    createTimedResolver(fallback, 6_000);

    vi.advanceTimersByTime(5_999);
    expect(fallback).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(fallback).toHaveBeenCalledTimes(1);
  });

  it("runs only the first geolocation outcome", () => {
    vi.useFakeTimers();
    const fallback = vi.fn();
    const success = vi.fn();
    const error = vi.fn();
    const resolver = createTimedResolver(fallback, 6_000);

    resolver.resolve(success);
    resolver.resolve(error);
    vi.advanceTimersByTime(6_000);

    expect(success).toHaveBeenCalledTimes(1);
    expect(error).not.toHaveBeenCalled();
    expect(fallback).not.toHaveBeenCalled();
  });
});

describe("Yandex map controls", () => {
  it("recenters without passing a zoom that would reset the live viewport", () => {
    const setCenter = vi.fn();

    recenterMapPreservingZoom({ setCenter }, [55.75, 37.61]);

    expect(setCenter).toHaveBeenCalledWith([55.75, 37.61]);
  });

  it("updates only the previously and newly active placemarks", () => {
    const firstSet = vi.fn();
    const secondSet = vi.fn();
    const untouchedSet = vi.fn();
    const pins = new Map([
      ["first", { options: { set: firstSet } }],
      ["second", { options: { set: secondSet } }],
      ["untouched", { options: { set: untouchedSet } }],
    ]);
    const normalLayout = {};
    const activeLayout = {};

    updateActivePinLayouts(pins, "first", "second", { normalLayout, activeLayout });

    expect(firstSet).toHaveBeenCalledWith("iconLayout", normalLayout);
    expect(secondSet).toHaveBeenCalledWith("iconLayout", activeLayout);
    expect(untouchedSet).not.toHaveBeenCalled();
  });
});
