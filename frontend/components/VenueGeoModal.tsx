"use client";

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { geocodeAddress, searchPlaces, type GeoResult } from "@/lib/geocode";
import { updateVenue, type ApiVenue } from "@/lib/api";

const YandexMap = dynamic(
  () => import("@/components/map/YandexMap").then((m) => m.YandexMap),
  { ssr: false },
);

const MOSCOW: [number, number] = [55.7558, 37.6173];

export function VenueGeoModal({
  venue,
  onSaved,
  onClose,
}: {
  venue: ApiVenue;
  onSaved: (v: ApiVenue) => void;
  onClose: () => void;
}) {
  const [q, setQ] = useState(venue.address ?? venue.name);
  const [debounced, setDebounced] = useState("");
  const [results, setResults] = useState<GeoResult[]>([]);
  const [pos, setPos] = useState<[number, number] | null>(
    venue.lat != null && venue.lon != null ? [venue.lat, venue.lon] : null,
  );
  const [saving, setSaving] = useState(false);

  // Debounce address lookups to ~1 req/s.
  useEffect(() => {
    const t = setTimeout(() => setDebounced(q), 700);
    return () => clearTimeout(t);
  }, [q]);

  useEffect(() => {
    if (debounced.trim() === "") return;
    let live = true;
    Promise.allSettled([searchPlaces(debounced), geocodeAddress(debounced)]).then(
      ([places, addrs]) => {
        if (!live) return;
        const p = places.status === "fulfilled" ? places.value : [];
        const a = addrs.status === "fulfilled" ? addrs.value : [];
        const seen = new Set<string>();
        const merged = [...p, ...a].filter((r) => {
          if (seen.has(r.label)) return false;
          seen.add(r.label);
          return true;
        });
        setResults(merged);
      },
    );
    return () => {
      live = false;
    };
  }, [debounced]);

  const save = async () => {
    if (!pos) return;
    setSaving(true);
    try {
      const v = await updateVenue(venue.id, { lat: pos[0], lon: pos[1] });
      onSaved(v);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg rounded-card bg-bg p-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-2 text-[17px] font-medium">
          Где находится «{venue.name}»?
        </h2>
        <input
          className="mb-2 w-full rounded-control bg-fill px-3.5 py-2.5 text-[15px] text-label outline-none placeholder:text-label-secondary"
          placeholder="Название или адрес"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        {results.length > 0 && (
          <div className="mb-2 max-h-40 overflow-auto rounded-control bg-bg-secondary">
            {results.map((r, i) => (
              <button
                key={i}
                type="button"
                className="block w-full px-3.5 py-2 text-left text-[14px] text-label hover:bg-fill"
                onClick={() => {
                  setPos([r.lat, r.lon]);
                  setResults([]);
                  setQ(r.label);
                }}
              >
                {r.label}
              </button>
            ))}
          </div>
        )}
        <YandexMap
          center={pos ?? MOSCOW}
          marker={pos ?? undefined}
          draggableMarker
          onMarkerMove={(lat, lon) => setPos([lat, lon])}
        />
        <p className="mt-1 text-[12px] text-label-secondary">
          Поиск по названию и адресу — © Яндекс. Перетащите метку для точности.
        </p>
        <div className="mt-3 flex justify-end gap-2">
          <button
            type="button"
            className="px-3 py-2 text-[15px] text-label"
            onClick={onClose}
          >
            Отмена
          </button>
          <button
            type="button"
            disabled={!pos || saving}
            className="rounded-control bg-accent px-3 py-2 text-[15px] text-white disabled:opacity-50"
            onClick={save}
          >
            {saving ? "Сохранение…" : "Сохранить"}
          </button>
        </div>
      </div>
    </div>
  );
}
