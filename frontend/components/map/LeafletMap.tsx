"use client";

import { useEffect, useRef } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";

// Fix default marker icon paths under bundlers.
import marker from "leaflet/dist/images/marker-icon.png";
import marker2x from "leaflet/dist/images/marker-icon-2x.png";
import shadow from "leaflet/dist/images/marker-shadow.png";

const icon = L.icon({
  iconUrl: (marker as unknown as { src: string }).src,
  iconRetinaUrl: (marker2x as unknown as { src: string }).src,
  shadowUrl: (shadow as unknown as { src: string }).src,
  iconSize: [25, 41],
  iconAnchor: [12, 41],
});

export interface MapPin {
  id: string;
  lat: number;
  lon: number;
  label?: string;
  href?: string;
}

export function LeafletMap({
  center,
  zoom = 13,
  marker: markerPos,
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
  const mapRef = useRef<L.Map | null>(null);
  const markerRef = useRef<L.Marker | null>(null);
  const layerRef = useRef<L.LayerGroup | null>(null);

  // init once
  useEffect(() => {
    if (!elRef.current || mapRef.current) return;
    const map = L.map(elRef.current).setView(center, zoom);
    L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "© OpenStreetMap",
      maxZoom: 19,
    }).addTo(map);
    layerRef.current = L.layerGroup().addTo(map);
    mapRef.current = map;
    return () => {
      map.remove();
      mapRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // recenter
  useEffect(() => {
    mapRef.current?.setView(center, zoom);
  }, [center, zoom]);

  // single draggable/static marker
  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    if (markerPos) {
      if (!markerRef.current) {
        markerRef.current = L.marker(markerPos, { icon, draggable: draggableMarker }).addTo(map);
        markerRef.current.on("dragend", () => {
          const p = markerRef.current!.getLatLng();
          onMarkerMove?.(p.lat, p.lng);
        });
      } else {
        markerRef.current.setLatLng(markerPos);
      }
    } else if (markerRef.current) {
      markerRef.current.remove();
      markerRef.current = null;
    }
  }, [markerPos, draggableMarker, onMarkerMove]);

  // multi-pin layer
  useEffect(() => {
    const layer = layerRef.current;
    if (!layer) return;
    layer.clearLayers();
    (pins ?? []).forEach((p) => {
      const m = L.marker([p.lat, p.lon], { icon });
      if (p.label) m.bindPopup(p.href ? `<a href="${p.href}">${p.label}</a>` : p.label);
      layer.addLayer(m);
    });
  }, [pins]);

  return <div ref={elRef} className={className} />;
}
