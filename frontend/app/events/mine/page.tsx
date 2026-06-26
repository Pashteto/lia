"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";

import { EventApplicationsPanel } from "@/components/EventApplicationsPanel";
import { PublishEventButton } from "@/components/PublishEventButton";
import { EventCard } from "@/components/ui/EventCard";
import { fetchMyEvents } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { LiaEvent } from "@/lib/types";

const STATUS_LABEL: Record<string, string> = {
  draft: "Черновик",
  pending_review: "На модерации",
  rejected: "Снято модератором",
  cancelled: "Отменено",
};

/** Collapsible "Заявки" expander shown on application-mode events. */
function ApplicationsExpander({ event }: { event: LiaEvent }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="mt-1">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-1 rounded-control px-2 py-1.5 text-[13px] font-medium text-accent hover:bg-accent/8 transition"
      >
        <span>{open ? "▾" : "▸"}</span>
        <span>Заявки</span>
      </button>
      {open && <EventApplicationsPanel eventId={event.id} />}
    </div>
  );
}

// "Мои события" — events created by the signed-in user, including drafts that
// don't appear in the public discovery feed. Backed by GET /events/mine.
export default function MyEventsPage() {
  const { isAuthed, ready } = useAuth();

  const {
    data: events = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: ready && isAuthed,
  });

  if (!ready) {
    return <div className="min-h-screen bg-bg-grouped" />;
  }

  if (!isAuthed) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16">
        <Link href="/" className="inline-flex items-center text-[17px] text-accent">
          ‹ События
        </Link>
        <div className="mt-8 text-center">
          <h1 className="text-[28px] font-bold tracking-[-0.022em]">Мои события</h1>
          <p className="mt-3 text-label-secondary">
            Войдите, чтобы увидеть созданные вами события.
          </p>
        </div>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-5xl px-4 py-8 max-sm:pb-28">
      <Link href="/" className="mb-4 inline-flex items-center text-[17px] text-accent">
        ‹ События
      </Link>
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-[28px] font-bold tracking-[-0.022em]">Мои события</h1>
        <Link
          href="/events/new"
          className="rounded-full bg-accent px-4 py-2 text-[15px] font-medium text-white"
        >
          Создать
        </Link>
      </div>

      {isLoading ? (
        <p className="mt-8 text-label-secondary">Загрузка…</p>
      ) : isError ? (
        <p className="mt-8 text-label-secondary">Не удалось загрузить события.</p>
      ) : events.length === 0 ? (
        <p className="mt-8 text-label-secondary">
          Вы ещё не создали ни одного события.{" "}
          <Link href="/events/new" className="text-accent">
            Создать первое →
          </Link>
        </p>
      ) : (
        <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {events.map((e) => (
            <div key={e.id} className="relative">
              {e.status !== "published" && (
                <span className="absolute left-3 top-3 z-10 rounded-full bg-black/70 px-2 py-0.5 text-[12px] font-medium text-white">
                  {STATUS_LABEL[e.status] ?? e.status}
                </span>
              )}
              <EventCard event={e} />
              {e.status === "draft" && <PublishEventButton eventId={e.id} />}
              {e.signupMode === "application" && (
                <ApplicationsExpander event={e} />
              )}
            </div>
          ))}
        </div>
      )}
    </main>
  );
}
