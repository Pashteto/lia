"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";

import { EventApplicationsPanel } from "@/components/EventApplicationsPanel";
import { fetchMyEvents } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { LiaEvent } from "@/lib/types";

// Aggregated organizer view: each application-mode event the organizer owns,
// with its applicant list (named) and inline accept/decline (via the panel).
export default function OrganizerApplicationsPage() {
  const { isAuthed, ready } = useAuth();
  const { data: events = [], isLoading, isError } = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: ready && isAuthed,
  });

  if (!ready) return <div className="min-h-screen bg-bg-grouped" />;

  if (!isAuthed) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16">
        <h1 className="text-[28px] font-bold tracking-[-0.022em]">Заявки участников</h1>
        <p className="mt-2 text-label-secondary">Войдите, чтобы видеть заявки на ваши события.</p>
      </main>
    );
  }

  const applicationEvents = events.filter(
    (e: LiaEvent) => e.signupMode === "application",
  );

  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <Link href="/organizer" className="text-[14px] text-accent">← Организаторам</Link>
      <h1 className="mt-2 mb-6 text-[28px] font-bold tracking-[-0.022em]">Заявки участников</h1>

      {isLoading && <p className="text-label-secondary">Загрузка…</p>}
      {isError && <p className="text-red-500">Не удалось загрузить события.</p>}

      {!isLoading && !isError && applicationEvents.length === 0 && (
        <p className="text-label-secondary">
          У вас пока нет событий с записью по заявкам. Создайте событие с режимом
          «по заявке», чтобы принимать участников вручную.
        </p>
      )}

      <div className="space-y-6">
        {applicationEvents.map((event: LiaEvent) => (
          <section key={event.id} className="rounded-card bg-bg p-4 shadow-card-subtle">
            <div className="flex items-center justify-between gap-3">
              <Link href={`/events/${event.id}`} className="text-[17px] font-semibold">
                {event.title}
              </Link>
            </div>
            <EventApplicationsPanel eventId={event.id} eventTitle={undefined} />
          </section>
        ))}
      </div>
    </main>
  );
}
