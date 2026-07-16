"use client";

import { useEffect, useRef, useState } from "react";

export interface MapPin {
  id: string;
  lat: number;
  lon: number;
  label?: string;
  href?: string;
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
  className = "h-64 w-full rounded-control",
}: {
  center: [number, number];
  zoom?: number;
  marker?: [number, number];
  draggableMarker?: boolean;
  onMarkerMove?: (lat: number, lon: number) => void;
  pins?: MapPin[];
  className?: string;
}) {
  const elRef = useRef<HTMLDivElement>(null);
  // ymaps objects are untyped (the JS API ships no bundled TS types).
  const mapRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const markerRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const pinRefs = useRef<any[]>([]); // eslint-disable-line @typescript-eslint/no-explicit-any
  const onMoveRef = useRef(onMarkerMove);
  const [ready, setReady] = useState(false);

  // Keep the drag callback current without re-creating the marker.
  useEffect(() => {
    onMoveRef.current = onMarkerMove;
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
          controls: ["zoomControl"],
        });
        mapRef.current = map;
        setReady(true);
      })
      .catch(() => {
        /* leave placeholder; page stays up */
      });
    return () => {
      cancelled = true;
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

  // multi-pin layer
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps = (window as any).ymaps; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps) return;
    pinRefs.current.forEach((pm) => map.geoObjects.remove(pm));
    pinRefs.current = [];
    (pins ?? []).forEach((p) => {
      const label = escapeHtml(p.label ?? "");
      const balloon = p.href
        ? `<a href="${escapeHtml(p.href)}">${label}</a>`
        : label;
      const pm = new ymaps.Placemark(
        [p.lat, p.lon],
        { hintContent: label, balloonContent: balloon },
        {},
      );
      map.geoObjects.add(pm);
      pinRefs.current.push(pm);
    });
  }, [ready, pins]);

  if (!KEY) {
    return (
      <div
        className={`${className} flex items-center justify-center bg-fill text-[13px] text-label-secondary`}
      >
        Карта недоступна
      </div>
    );
  }
  return <div ref={elRef} className={className} />;
}
