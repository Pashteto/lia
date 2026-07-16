"use client";

import dynamic from "next/dynamic";

const YandexMap = dynamic(() => import("@/components/map/YandexMap").then((m) => m.YandexMap), {
  ssr: false,
});

export function VenueMap({ lat, lon }: { lat: number; lon: number }) {
  return <YandexMap center={[lat, lon]} marker={[lat, lon]} zoom={15} />;
}
