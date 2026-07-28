"use client";

import Link from "next/link";
import { Suspense, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";

import { EventApplicationsPanel } from "@/components/EventApplicationsPanel";
import { AuthGate } from "@/components/ui/AuthGate";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { fetchMyEvents } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { LiaEvent } from "@/lib/types";

const dayMonthFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  timeZone: "Europe/Moscow",
});

function PickerSkeleton() {
  return (
    <main className="mx-auto max-w-[1360px]">
      <Skeleton className="h-[64px] w-full border-x-0 border-t-0" />
      {Array.from({ length: 4 }, (_, i) => (
        <Skeleton key={i} className="h-[44px] w-full border-x-0 border-t-0" />
      ))}
    </main>
  );
}

function OrganizerApplicationsBody() {
  const { isAuthed, ready } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const selectedId = searchParams.get("event");

  const { data: events = [], isLoading, isError } = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: ready && isAuthed,
  });

  const selectEvent = useCallback(
    (id: string | null) => {
      const url = id
        ? `/organizer/applications?event=${encodeURIComponent(id)}`
        : "/organizer/applications";
      router.replace(url, { scroll: false });
    },
    [router],
  );

  if (!ready) return <PickerSkeleton />;

  if (!isAuthed) {
    return (
      <AuthGate title="Войдите, чтобы видеть заявки на ваши события" />
    );
  }

  const applicationEvents = events.filter(
    (e: LiaEvent) => e.signupMode === "application",
  );
  const selected = selectedId
    ? applicationEvents.find((e) => e.id === selectedId)
    : undefined;

  if (isLoading) return <PickerSkeleton />;

  if (isError) {
    return (
      <EmptyState
        numeral="—"
        title="Не удалось загрузить события"
        text="Обновите страницу или попробуйте позже."
      />
    );
  }

  if (applicationEvents.length === 0) {
    return (
      <EmptyState
        numeral="00"
        title="Нет событий с заявками"
        text="Создайте событие с режимом «по заявке», чтобы принимать участников вручную."
        actions={
          <Link
            href="/events/new"
            className="swiss-focus bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black"
          >
            + СОЗДАТЬ СОБЫТИЕ
          </Link>
        }
      />
    );
  }

  // Deep-link to unknown / non-application event → fall back to picker
  if (selectedId && !selected) {
    return (
      <main className="mx-auto max-w-[1360px]">
        <div className="border-b border-on-surface px-[20px] py-[14px] max-sm:px-[14px]">
          <p className="cap mb-[5px]">Заявки</p>
          <h1 className="text-[26px] font-black tracking-[-0.02em]">Событие не найдено</h1>
          <button
            type="button"
            className="cap mt-[10px] swiss-focus hover:underline"
            onClick={() => selectEvent(null)}
          >
            ← К списку событий
          </button>
        </div>
      </main>
    );
  }

  if (selected) {
    return (
      <main className="mx-auto max-w-[1360px] pb-[64px]">
        <div className="border-b border-on-surface px-[20px] py-[8px] max-sm:px-[14px]">
          <button
            type="button"
            className="cap swiss-focus hover:underline"
            onClick={() => selectEvent(null)}
          >
            ← Все события с заявками
          </button>
        </div>
        <EventApplicationsPanel
          eventId={selected.id}
          event={{
            title: selected.title,
            startsAt: selected.startsAt,
            capacity: selected.capacity,
            seatsRemaining: selected.seatsRemaining,
            signupMode: selected.signupMode,
          }}
        />
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px]">
      <div className="border-b border-on-surface px-[20px] py-[14px] max-sm:px-[14px]">
        <p className="cap mb-[5px]">Организатор</p>
        <h1 className="text-[26px] font-black tracking-[-0.02em]">Заявки участников</h1>
      </div>
      <ul className="flex flex-col">
        {applicationEvents.map((event) => (
          <li key={event.id} className="border-b border-on-surface">
            <button
              type="button"
              className="flex w-full items-baseline justify-between gap-[12px] px-[20px] py-[12px] text-left swiss-focus hover-invert max-sm:px-[14px]"
              onClick={() => selectEvent(event.id)}
            >
              <span className="min-w-0">
                <span className="block text-[14px] font-bold leading-tight">{event.title}</span>
                <span className="cap mt-[2px] block">
                  {dayMonthFmt.format(new Date(event.startsAt))}
                </span>
              </span>
              <span className="cap shrink-0">Открыть →</span>
            </button>
          </li>
        ))}
      </ul>
    </main>
  );
}

/** O4: event picker + deep-link `?event=` + applications panel. */
export function OrganizerApplications() {
  return (
    <Suspense fallback={<PickerSkeleton />}>
      <OrganizerApplicationsBody />
    </Suspense>
  );
}
