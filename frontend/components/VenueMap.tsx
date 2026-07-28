"use client";

import dynamic from "next/dynamic";

const YandexMap = dynamic(() => import("@/components/map/YandexMap").then((m) => m.YandexMap), {
  ssr: false,
});

/** Swiss map treatment: grayscale печатный план, square, hairline border. */
export function VenueMap({ lat, lon }: { lat: number; lon: number }) {
  return (
    <div className="border border-ink [filter:grayscale(1)_contrast(1.05)]">
      <YandexMap
        center={[lat, lon]}
        zoom={15}
        marker={[lat, lon]}
        className="h-[190px] w-full"
      />
    </div>
  );
}
