"use client";

import { createVenue, searchVenues, type ApiVenue } from "@/lib/api";
import { cn } from "@/lib/cn";
import { VenueGeoModal } from "@/components/VenueGeoModal";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

const inputCls =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

/**
 * Venue typeahead: debounced search over GET /venues; pick a result (sets value
 * to its id) or create a new venue inline (POST /venues, then select it).
 * `value` is the selected venue id ("" = none); `onChange` reports the new id.
 *
 * NOTE: an external reset (e.g. form.reset() setting value back to "") clears the
 * id but not the displayed text — remount via a `key` prop if you need a full
 * visual reset. Harmless in the create-event form, which never externally resets.
 */
export function VenuePicker({
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  value: _value,
  onChange,
}: {
  /** Current venue id ("" = none). Kept in interface for controlled-component API. */
  value: string;
  onChange: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [selected, setSelected] = useState<ApiVenue | null>(null);
  const [open, setOpen] = useState(false);
  const [geoOpen, setGeoOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const t = setTimeout(() => setDebounced(query), 250);
    return () => clearTimeout(t);
  }, [query]);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const { data: results = [] } = useQuery({
    queryKey: ["venues", debounced],
    queryFn: () => searchVenues(debounced),
    enabled: open,
  });

  const createMut = useMutation({
    mutationFn: (name: string) => createVenue({ name }),
    onSuccess: (venue) => {
      setSelected(venue);
      onChange(venue.id);
      setQuery(venue.name);
      setOpen(false);
    },
  });


  const pick = (v: ApiVenue) => {
    setSelected(v);
    onChange(v.id);
    setQuery(v.name);
    setOpen(false);
  };

  const trimmed = query.trim();
  const exactExists = results.some(
    (v) => v.name.toLowerCase() === trimmed.toLowerCase(),
  );

  return (
    <div ref={containerRef} className="relative">
      <input
        className={inputCls}
        placeholder="Площадка — начните вводить название"
        value={query}
        onChange={(e) => {
          setQuery(e.target.value);
          setOpen(true);
          if (selected) {
            setSelected(null);
            onChange("");
          }
        }}
        onFocus={() => setOpen(true)}
      />
      {open && (trimmed !== "" || results.length > 0) && (
        <div className="absolute z-20 mt-1 max-h-60 w-full overflow-auto rounded-control bg-bg-secondary shadow-card">
          {results.map((v) => (
            <button
              key={v.id}
              type="button"
              onClick={() => pick(v)}
              className="block w-full px-3.5 py-2.5 text-left text-[15px] hover:bg-fill"
            >
              {v.name}
              {v.metro ? (
                <span className="text-label-secondary"> · м. {v.metro}</span>
              ) : null}
            </button>
          ))}
          {trimmed !== "" && !exactExists && (
            <button
              type="button"
              onClick={() => createMut.mutate(trimmed)}
              disabled={createMut.isPending}
              className={cn(
                "block w-full px-3.5 py-2.5 text-left text-[15px] text-accent hover:bg-fill",
                createMut.isPending && "opacity-50",
              )}
            >
              {createMut.isPending ? "Создание…" : `Создать «${trimmed}»`}
            </button>
          )}
          {results.length === 0 && trimmed === "" && (
            <div className="px-3.5 py-2.5 text-[13px] text-label-secondary">
              Начните вводить название площадки
            </div>
          )}
        </div>
      )}
      {selected && (
        <button
          type="button"
          className="mt-1.5 text-[13px] text-accent"
          onClick={() => setGeoOpen(true)}
        >
          Указать на карте
        </button>
      )}
      {geoOpen && selected && (
        <VenueGeoModal
          venue={selected}
          onSaved={(v) => {
            setSelected(v);
            setGeoOpen(false);
          }}
          onClose={() => setGeoOpen(false)}
        />
      )}
    </div>
  );
}
