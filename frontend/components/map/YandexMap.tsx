"use client";

import { useEffect, useRef, useState } from "react";
import { radiusKmFromBounds, type LatLon, type MapBounds } from "@/lib/geo";

export interface MapPin {
  id: string;
  lat: number;
  lon: number;
  label?: string;
  href?: string;
  /** Positional numeral shown inside the square marker ("01"). Must match the
   * numeral of the same event in the list rail (handoff U5). */
  numeral?: string;
}

export interface MapViewport {
  center: LatLon;
  radiusKm: number;
}

const KEY = process.env.NEXT_PUBLIC_YANDEX_MAPS_KEY ?? "";

// Event titles are user-supplied and land in balloon HTML — escape them.
function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

// Load the JS API v2.1 script exactly once across every map instance on the page.
let loaderPromise: Promise<void> | null = null;
function loadYmaps(): Promise<void> {
  const w = window as unknown as { ymaps?: { ready: (cb: () => void) => void } };
  if (loaderPromise) return loaderPromise;
  loaderPromise = new Promise<void>((resolve, reject) => {
    if (w.ymaps) {
      w.ymaps.ready(() => resolve());
      return;
    }
    const script = document.createElement("script");
    script.src = `https://api-maps.yandex.ru/2.1/?apikey=${KEY}&lang=ru_RU`;
    script.onload = () => w.ymaps!.ready(() => resolve());
    script.onerror = () => reject(new Error("yandex maps failed to load"));
    document.head.appendChild(script);
  });
  return loaderPromise;
}

export function YandexMap({
  center,
  zoom = 13,
  marker,
  draggableMarker = false,
  onMarkerMove,
  pins,
  hideControls = false,
  activePinId = null,
  onPinClick,
  onViewportChange,
  className = "h-64 w-full",
}: {
  center: LatLon;
  zoom?: number;
  marker?: LatLon;
  draggableMarker?: boolean;
  onMarkerMove?: (lat: number, lon: number) => void;
  pins?: MapPin[];
  /** U5 hides the zoom control; the venue map keeps it. */
  hideControls?: boolean;
  activePinId?: string | null;
  onPinClick?: (id: string) => void;
  /** Debounced (250ms) viewport report — drives the U5 «Радиус» cell and the
   * "search this area" pill. */
  onViewportChange?: (viewport: MapViewport) => void;
  className?: string;
}) {
  const elRef = useRef<HTMLDivElement>(null);
  // ymaps objects are untyped (the JS API ships no bundled TS types).
  const mapRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const markerRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const pinRefs = useRef<any[]>([]); // eslint-disable-line @typescript-eslint/no-explicit-any
  const onMoveRef = useRef(onMarkerMove);
  const onPinClickRef = useRef(onPinClick);
  const onViewportRef = useRef(onViewportChange);
  const viewportTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [ready, setReady] = useState(false);

  // Keep callbacks current without re-creating the map or its geo objects.
  useEffect(() => {
    onMoveRef.current = onMarkerMove;
    onPinClickRef.current = onPinClick;
    onViewportRef.current = onViewportChange;
  });

  // init once
  useEffect(() => {
    if (!KEY) return;
    let cancelled = false;
    loadYmaps()
      .then(() => {
        if (cancelled || !elRef.current || mapRef.current) return;
        const ymaps = (window as any).ymaps; // eslint-disable-line @typescript-eslint/no-explicit-any
        // v2.1 takes [lat, lon] — same order as our props, no conversion.
        const map = new ymaps.Map(elRef.current, {
          center,
          zoom,
          controls: hideControls ? [] : ["zoomControl"],
        });
        map.events.add("boundschange", () => {
          if (!onViewportRef.current) return;
          if (viewportTimer.current) clearTimeout(viewportTimer.current);
          viewportTimer.current = setTimeout(() => {
            const bounds = map.getBounds() as MapBounds;
            const c = map.getCenter() as LatLon;
            onViewportRef.current?.({
              center: [c[0], c[1]],
              radiusKm: radiusKmFromBounds(bounds),
            });
          }, 250);
        });
        mapRef.current = map;
        setReady(true);
      })
      .catch(() => {
        /* leave placeholder; page stays up */
      });
    return () => {
      cancelled = true;
      if (viewportTimer.current) clearTimeout(viewportTimer.current);
      mapRef.current?.destroy?.();
      mapRef.current = null;
      setReady(false);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // recenter
  useEffect(() => {
    if (!ready) return;
    mapRef.current?.setCenter(center, zoom);
  }, [ready, center, zoom]);

  // single marker (static or draggable)
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps = (window as any).ymaps; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps) return;
    if (markerRef.current) {
      map.geoObjects.remove(markerRef.current);
      markerRef.current = null;
    }
    if (marker) {
      const pm = new ymaps.Placemark(marker, {}, { draggable: draggableMarker });
      pm.events.add("dragend", () => {
        const c = pm.geometry.getCoordinates(); // [lat, lon]
        onMoveRef.current?.(c[0], c[1]);
      });
      map.geoObjects.add(pm);
      markerRef.current = pm;
    }
  }, [ready, marker, draggableMarker]);

  // multi-pin layer — square ink markers numbered in mono
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps = (window as any).ymaps; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps) return;

    // Two layouts (normal / selected) built per run; $[properties.iconContent]
    // is the documented v2.1 substitution for per-placemark text.
    const layoutFor = (active: boolean) =>
      ymaps.templateLayoutFactory.createClass(
        `<div class="swiss-pin${active ? " swiss-pin--on" : ""}">$[properties.iconContent]</div>`,
      );
    const normalLayout = layoutFor(false);
    const activeLayout = layoutFor(true);

    pinRefs.current.forEach((pm) => map.geoObjects.remove(pm));
    pinRefs.current = [];
    (pins ?? []).forEach((p) => {
      const label = escapeHtml(p.label ?? "");
      const balloon = p.href ? `<a href="${escapeHtml(p.href)}">${label}</a>` : label;
      const isActive = p.id === activePinId;
      const pm = new ymaps.Placemark(
        [p.lat, p.lon],
        {
          hintContent: label,
          balloonContent: balloon,
          iconContent: escapeHtml(p.numeral ?? label),
        },
        {
          iconLayout: isActive ? activeLayout : normalLayout,
          // Hit area in coordinate space *after* .swiss-pin's translate: the box
          // sits above-right of the point, its stem tip on it. Slightly generous
          // so a 22px square is comfortably clickable. Verify in Step 3.
          iconShape: { type: "Rectangle", coordinates: [[-9, -31], [15, -7]] },
          // The balloon is redundant when the caller handles selection itself.
          openBalloonOnClick: !onPinClickRef.current,
        },
      );
      pm.events.add("click", () => onPinClickRef.current?.(p.id));
      map.geoObjects.add(pm);
      pinRefs.current.push(pm);
    });
  }, [ready, pins, activePinId]);

  if (!KEY) {
    return (
      <div
        className={`${className} flex items-center justify-center border border-rule-inner bg-cell-blank text-[11.5px] text-text-dim`}
      >
        Карта недоступна
      </div>
    );
  }
  return <div ref={elRef} className={className} />;
}
