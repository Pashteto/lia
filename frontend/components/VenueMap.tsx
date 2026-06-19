"use client";

import dynamic from "next/dynamic";

const LeafletMap = dynamic(() => import("@/components/map/LeafletMap").then((m) => m.LeafletMap), {
  ssr: false,
});

export function VenueMap({ lat, lon }: { lat: number; lon: number }) {
  return <LeafletMap center={[lat, lon]} marker={[lat, lon]} zoom={15} />;
}
