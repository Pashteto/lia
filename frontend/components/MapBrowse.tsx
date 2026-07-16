"use client";

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { fetchNearbyEvents } from "@/lib/api";
import type { LiaEvent } from "@/lib/types";

const YandexMap = dynamic(() => import("@/components/map/YandexMap").then((m) => m.YandexMap), {
  ssr: false,
});

const MOSCOW: [number, number] = [55.7558, 37.6173];
const PIN_CAP = 100;

export function MapBrowse() {
  const [center, setCenter] = useState<[number, number]>(MOSCOW);
  const [events, setEvents] = useState<LiaEvent[]>([]);
  const [truncated, setTruncated] = useState(false);

  const load = async (lat: number, lon: number) => {
    const all = await fetchNearbyEvents(lat, lon, 200);
    const withCoords = all.filter((e) => e.venue?.lat != null && e.venue?.lon != null);
    setTruncated(withCoords.length > PIN_CAP);
    setEvents(withCoords.slice(0, PIN_CAP));
  };

  useEffect(() => {
    navigator.geolocation?.getCurrentPosition(
      (pos) => {
        const c: [number, number] = [pos.coords.latitude, pos.coords.longitude];
        setCenter(c);
        void load(c[0], c[1]);
      },
      () => void load(MOSCOW[0], MOSCOW[1]),
    );
  }, []);

  const pins = events.map((e) => ({
    id: e.id,
    lat: e.venue!.lat!,
    lon: e.venue!.lon!,
    label: e.title,
    href: `/events/${e.id}`,
  }));

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h1 className="text-[22px] font-semibold">События на карте</h1>
        <button
          type="button"
          className="rounded-control bg-fill px-3 py-2 text-[14px]"
          onClick={() => void load(center[0], center[1])}
        >
          Искать в этой области
        </button>
      </div>
      <YandexMap center={center} pins={pins} className="h-[60vh] w-full rounded-control" />
      {truncated && (
        <p className="text-[12px] text-label-secondary">Показаны первые {PIN_CAP} событий.</p>
      )}
    </div>
  );
}
