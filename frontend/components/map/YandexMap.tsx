"use client";

import { useEffect, useRef, useState } from "react";
import { toLngLat } from "@/lib/coords";

export interface MapPin {
  id: string;
  lat: number;
  lon: number;
  label?: string;
  href?: string;
}

const KEY = process.env.NEXT_PUBLIC_YANDEX_MAPS_KEY ?? "";
const PIN_CLASS =
  "block h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-accent shadow";

// Load the JS API v3 script exactly once across every map instance on the page.
let loaderPromise: Promise<void> | null = null;
function loadYmaps(): Promise<void> {
  const w = window as unknown as { ymaps3?: { ready: Promise<void> } };
  if (w.ymaps3) return w.ymaps3.ready;
  if (loaderPromise) return loaderPromise;
  loaderPromise = new Promise<void>((resolve, reject) => {
    const script = document.createElement("script");
    script.src = `https://api-maps.yandex.ru/v3/?apikey=${KEY}&lang=ru_RU`;
    script.onload = () => w.ymaps3!.ready.then(() => resolve(), reject);
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
  useEffect(() => {
    onMoveRef.current = onMarkerMove;
  });
  const [ready, setReady] = useState(false);

  // init once
  useEffect(() => {
    if (!KEY) return;
    let cancelled = false;
    loadYmaps()
      .then(() => {
        if (cancelled || !elRef.current || mapRef.current) return;
        const ymaps3 = (window as any).ymaps3; // eslint-disable-line @typescript-eslint/no-explicit-any
        const { YMap, YMapDefaultSchemeLayer, YMapDefaultFeaturesLayer } = ymaps3;
        const map = new YMap(elRef.current, {
          location: { center: toLngLat(center), zoom },
        });
        map.addChild(new YMapDefaultSchemeLayer());
        map.addChild(new YMapDefaultFeaturesLayer());
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
    mapRef.current?.update?.({ location: { center: toLngLat(center), zoom } });
  }, [ready, center, zoom]);

  // single marker (static or draggable)
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps3 = (window as any).ymaps3; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps3) return;
    const { YMapMarker } = ymaps3;
    if (markerRef.current) {
      map.removeChild(markerRef.current);
      markerRef.current = null;
    }
    if (marker) {
      const el = document.createElement("div");
      el.className = PIN_CLASS;
      const m = new YMapMarker(
        {
          coordinates: toLngLat(marker),
          draggable: draggableMarker,
          onDragEnd: (coords: [number, number]) =>
            onMoveRef.current?.(coords[1], coords[0]),
        },
        el,
      );
      map.addChild(m);
      markerRef.current = m;
    }
  }, [ready, marker, draggableMarker]);

  // multi-pin layer
  useEffect(() => {
    if (!ready) return;
    const map = mapRef.current;
    const ymaps3 = (window as any).ymaps3; // eslint-disable-line @typescript-eslint/no-explicit-any
    if (!map || !ymaps3) return;
    const { YMapMarker } = ymaps3;
    pinRefs.current.forEach((m) => map.removeChild(m));
    pinRefs.current = [];
    (pins ?? []).forEach((p) => {
      const el = document.createElement(p.href ? "a" : "div");
      if (p.href) (el as HTMLAnchorElement).href = p.href;
      el.title = p.label ?? "";
      el.className = PIN_CLASS;
      const m = new YMapMarker({ coordinates: [p.lon, p.lat] }, el);
      map.addChild(m);
      pinRefs.current.push(m);
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
