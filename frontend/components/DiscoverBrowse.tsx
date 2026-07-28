"use client";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { EventModule } from "@/components/ui/EventModule";
import { Skeleton } from "@/components/ui/Skeleton";
import {
  fetchPublishedEvents,
  type ApiCategory,
} from "@/lib/api";
import {
  DISCOVER_CHIPS,
  intentFromChip,
  parseDiscoverQuery,
  type DiscoverIntent,
  type DiscoverIntentId,
} from "@/lib/discover-intent";
import { rankDiscover, type DiscoverHit } from "@/lib/discover-rank";
import { eventToModuleProps } from "@/lib/event-module";
import type { LatLon } from "@/lib/geo";
import type { LiaEvent } from "@/lib/types";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useState } from "react";

type Phase = "idle" | "loading" | "results" | "empty" | "error";

export function DiscoverBrowse({
  initialEvents,
  categories,
}: {
  initialEvents: LiaEvent[];
  categories: ApiCategory[];
}) {
  const router = useRouter();
  const [draft, setDraft] = useState("");
  const [phase, setPhase] = useState<Phase>("idle");
  const [answer, setAnswer] = useState("");
  const [hits, setHits] = useState<DiscoverHit[]>([]);
  const [activeChip, setActiveChip] = useState<DiscoverIntentId | null>(null);
  const [lastIntent, setLastIntent] = useState<DiscoverIntent | null>(null);
  const [now] = useState(() => new Date());

  const { data: events = initialEvents, isError, refetch } = useQuery({
    queryKey: ["events", "published", null, null],
    queryFn: () => fetchPublishedEvents(),
    initialData: initialEvents,
  });

  async function resolveGeo(wantsNearby: boolean): Promise<LatLon | null> {
    if (!wantsNearby || typeof navigator === "undefined" || !navigator.geolocation) {
      return null;
    }
    return new Promise((resolve) => {
      navigator.geolocation.getCurrentPosition(
        (pos) => resolve([pos.coords.latitude, pos.coords.longitude]),
        () => resolve(null),
        { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 },
      );
    });
  }

  async function runIntent(intent: DiscoverIntent, chipId: DiscoverIntentId | null) {
    setLastIntent(intent);
    setActiveChip(chipId);
    setPhase("loading");
    try {
      if (isError && events.length === 0) {
        setPhase("error");
        return;
      }
      const userLatLon = await resolveGeo(intent.wantsNearby);
      const catalogue = events.length > 0 ? events : initialEvents;
      const ranking = rankDiscover(catalogue, intent, { now, userLatLon });
      setAnswer(ranking.answer);
      setHits(ranking.hits);
      setPhase(ranking.hits.length > 0 ? "results" : "empty");
    } catch {
      setPhase("error");
    }
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const intent = parseDiscoverQuery(draft);
    if (!intent) return;
    void runIntent(intent, null);
  }

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
      <div className="border-b border-ink px-[20px] py-[18px] max-sm:px-[14px] max-sm:py-[13px]">
        <p className="cap mb-[6px] hidden sm:block">AI-подбор</p>
        <h1 className="mb-[14px] text-[34px] font-black leading-[0.94] tracking-[-0.03em] max-sm:mb-[10px] max-sm:text-[20px]">
          Что вам сейчас
          <br className="max-sm:hidden" />{" "}
          откликается?
        </h1>
        <form onSubmit={onSubmit} className="flex border border-ink">
          <input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="Хочу спокойный вечер с искусством и без толпы…"
            aria-label="Запрос для подбора"
            className="swiss-focus min-h-[44px] flex-1 bg-transparent px-[14px] py-[11px] text-[12px] text-field-text placeholder:text-field-text max-sm:px-[10px] max-sm:py-[9px] max-sm:text-[10.5px] max-sm:placeholder:text-[10.5px]"
          />
          <button
            type="submit"
            aria-label="Подобрать"
            className="swiss-focus hover-invert min-h-[44px] bg-ink px-[20px] py-[11px] text-[12px] font-bold text-white max-sm:px-[13px] max-sm:py-[9px] max-sm:text-[11px]"
          >
            →
          </button>
        </form>
        <div className="mt-[10px] flex gap-[6px] overflow-x-auto [scrollbar-width:none] max-sm:mt-[8px] max-sm:gap-[5px] [&::-webkit-scrollbar]:hidden">
          {DISCOVER_CHIPS.map((c) => (
            <Chip
              key={c.id}
              variant={activeChip === c.id ? "active" : "default"}
              className="min-h-[44px] max-sm:text-[8px]"
              onClick={() => {
                setDraft(c.label);
                void runIntent(intentFromChip(c.id), c.id);
              }}
            >
              {c.label}
            </Chip>
          ))}
        </div>
      </div>

      {phase === "results" ? (
        <p className="border-b border-ink px-[20px] py-[12px] text-[12.5px] max-sm:px-[14px] max-sm:py-[10px] max-sm:text-[11px]">
          {answer}
        </p>
      ) : null}

      {phase === "loading" ? (
        <div className="grid grid-cols-3 border-b border-ink max-sm:grid-cols-1">
          {Array.from({ length: 3 }, (_, i) => (
            <Skeleton key={i} className="h-[140px]" />
          ))}
        </div>
      ) : null}

      {phase === "results" ? (
        <div className="grid grid-cols-3 border-b border-ink max-sm:grid-cols-1 [&>a]:border-b [&>a]:border-r [&>a]:border-rule-inner max-sm:[&>a]:border-r-0">
          {hits.map((h) => (
            <EventModule
              key={h.event.id}
              {...eventToModuleProps(h.event, categories)}
              matchReason={h.matchReason}
            />
          ))}
        </div>
      ) : null}

      {phase === "empty" ? (
        <EmptyState
          numeral="00"
          title="Ничего не нашлось"
          text="Попробуйте другой запрос или соберите ленту вручную."
          actions={
            <Button
              variant="ghost"
              onClick={() => {
                setPhase("idle");
                setHits([]);
                setAnswer("");
                setActiveChip(null);
                setDraft("");
              }}
            >
              Сбросить
            </Button>
          }
        />
      ) : null}

      {phase === "error" ? (
        <EmptyState
          numeral="!"
          title="Не удалось загрузить события"
          text="Проверьте соединение и попробуйте ещё раз."
          actions={
            <Button
              variant="ghost"
              onClick={() => {
                void refetch().then(() => {
                  if (lastIntent) void runIntent(lastIntent, activeChip);
                });
              }}
            >
              Повторить
            </Button>
          }
        />
      ) : null}

      <div className="flex items-center justify-between gap-[12px] border-b border-ink px-[20px] py-[10px] max-sm:px-[14px]">
        <span className="cap">Не то? Соберите вручную</span>
        <Chip
          className="min-h-[44px]"
          onClick={() => router.push("/")}
        >
          Точные фильтры →
        </Chip>
      </div>
    </main>
  );
}
