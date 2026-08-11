import { afterEach, describe, expect, it, vi } from "vitest";

import {
  recenterMapPreservingZoom,
  updateActivePinLayouts,
} from "@/components/map/yandex-map-controls";

afterEach(() => {
  vi.useRealTimers();
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
