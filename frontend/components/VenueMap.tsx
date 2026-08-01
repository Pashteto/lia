"use client";

import dynamic from "next/dynamic";

const YandexMap = dynamic(() => import("@/components/map/YandexMap").then((m) => m.YandexMap), {
  ssr: false,
});

/** Swiss map treatment: grayscale печатный план, square, hairline border.
 * `numeral` puts the event's category numeral in the marker, matching the
 * square ink markers on /map. */
export function VenueMap({
  lat,
  lon,
  numeral,
}: {
  lat: number;
  lon: number;
  numeral?: string;
}) {
  return (
    <div className="border border-ink [filter:grayscale(1)_contrast(1.05)]">
      <YandexMap
        center={[lat, lon]}
        zoom={15}
        marker={[lat, lon]}
        markerNumeral={numeral}
        hideControls
        className="h-[190px] w-full"
      />
    </div>
  );
}
