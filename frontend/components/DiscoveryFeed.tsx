"use client";

import { EventCard } from "@/components/ui/EventCard";
import { FilterChip } from "@/components/ui/FilterChip";
import { SearchField } from "@/components/ui/SearchField";
import { FILTERS, MOCK_EVENTS } from "@/lib/mock-events";
import { useMemo, useState } from "react";

/**
 * Discovery feed: large title, search field, capsule filter row, event grid.
 * Filtering here is a scaffold stub over mock data — real filtering will hit
 * the backend search endpoint.
 */
export function DiscoveryFeed() {
  const [active, setActive] = useState("all");
  const [query, setQuery] = useState("");

  const events = useMemo(() => {
    return MOCK_EVENTS.filter((e) => {
      const matchesFilter = active === "all" || e.category.slug === active;
      const matchesQuery =
        query.trim() === "" ||
        e.title.toLowerCase().includes(query.toLowerCase()) ||
        e.organizer.name.toLowerCase().includes(query.toLowerCase());
      return matchesFilter && matchesQuery;
    });
  }, [active, query]);

  return (
    <main className="mx-auto max-w-3xl px-5 pb-28 pt-6">
      <h1 className="mb-4 text-[34px] font-bold tracking-[-0.022em]">События</h1>

      <SearchField
        placeholder="Поиск по названию, месту, ведущему"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        className="mb-4"
      />

      <div className="-mx-5 mb-6 flex gap-2 overflow-x-auto px-5 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {FILTERS.map((f) => (
          <FilterChip
            key={f.slug}
            label={f.label}
            active={active === f.slug}
            onClick={() => setActive(f.slug)}
          />
        ))}
      </div>

      {events.length > 0 ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {events.map((event) => (
            <EventCard key={event.id} event={event} />
          ))}
        </div>
      ) : (
        <p className="py-16 text-center text-[15px] text-label-secondary">
          Ничего не нашлось. Попробуйте другой фильтр.
        </p>
      )}
    </main>
  );
}
