import type { LatLon } from "@/lib/geo";

interface MapCenterController {
  setCenter(center: LatLon): unknown;
}

interface PlacemarkLayoutController {
  options: {
    set(key: "iconLayout", layout: unknown): unknown;
  };
}

interface PinLayouts {
  normalLayout: unknown;
  activeLayout: unknown;
}

export function recenterMapPreservingZoom(
  map: MapCenterController,
  center: LatLon,
): void {
  map.setCenter(center);
}

export function updateActivePinLayouts(
  pins: ReadonlyMap<string, PlacemarkLayoutController>,
  previousActiveId: string | null,
  activeId: string | null,
  layouts: PinLayouts,
): void {
  if (previousActiveId === activeId) return;
  if (previousActiveId) {
    pins.get(previousActiveId)?.options.set("iconLayout", layouts.normalLayout);
  }
  if (activeId) {
    pins.get(activeId)?.options.set("iconLayout", layouts.activeLayout);
  }
}
