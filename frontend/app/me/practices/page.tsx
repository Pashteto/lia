"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";

import { FeedbackForm } from "@/components/FeedbackForm";
import { Button } from "@/components/ui/Button";
import { FilterChip } from "@/components/ui/FilterChip";
import { cancelRsvp, fetchMyPractices } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { formatEventDate } from "@/lib/format";
import type { Rsvp, RsvpStatus } from "@/lib/types";

type Tab = "upcoming" | "past";

const TABS: { key: Tab; label: string }[] = [
  { key: "upcoming", label: "Предстоящие" },
  { key: "past", label: "Прошедшие" },
];

const STATUS_CHIP: Partial<Record<RsvpStatus, { label: string; className: string }>> = {
  going: { label: "вы записаны", className: "bg-green-500/15 text-green-700 dark:text-green-400" },
  waitlist: { label: "в листе ожидания", className: "bg-amber-500/15 text-amber-700 dark:text-amber-400" },
  accepted: { label: "заявка принята", className: "bg-accent/15 text-accent" },
};

function StatusChip({ status }: { status: RsvpStatus }) {
  const chip = STATUS_CHIP[status];
  if (!chip) return null;
  return (
    <span
      className={`inline-block rounded-full px-2.5 py-0.5 text-[12px] font-medium ${chip.className}`}
    >
      {chip.label}
    </span>
  );
}

function PracticeRow({ rsvp, tab }: { rsvp: Rsvp; tab: Tab }) {
  const queryClient = useQueryClient();
  const [cancelling, setCancelling] = useState(false);
  const [showFeedback, setShowFeedback] = useState(false);

  const event = rsvp.event;
  const canCancel = tab === "upcoming" && (rsvp.status === "going" || rsvp.status === "waitlist");
  const cancelLabel = rsvp.status === "waitlist" ? "Покинуть лист" : "Отписаться";

  async function handleCancel() {
    setCancelling(true);
    try {
      await cancelRsvp(rsvp.eventId);
      await queryClient.invalidateQueries({ queryKey: ["my-practices", tab] });
    } finally {
      setCancelling(false);
    }
  }

  return (
    <div className="flex flex-col gap-2 rounded-card bg-bg-secondary p-4 shadow-card-subtle">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-col gap-1 min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <StatusChip status={rsvp.status} />
            {event && (
              <span className="text-[13px] text-label-secondary">
                {formatEventDate(event.startsAt)}
              </span>
            )}
          </div>
          {event ? (
            <Link
              href={`/events/${rsvp.eventId}`}
              className="text-[17px] font-semibold leading-snug hover:text-accent transition-colors line-clamp-2"
            >
              {event.title}
            </Link>
          ) : (
            <span className="text-[17px] font-semibold leading-snug text-label-secondary">
              Событие #{rsvp.eventId.slice(0, 8)}
            </span>
          )}
          {event?.venue?.name && (
            <p className="text-[13px] text-label-secondary">{event.venue.name}</p>
          )}
        </div>

        {canCancel && (
          <Button
            variant="tinted"
            onClick={handleCancel}
            disabled={cancelling}
            className="shrink-0 self-start sm:self-center"
          >
            {cancelling ? "…" : cancelLabel}
          </Button>
        )}

        {tab === "past" && (
          <Button
            variant="tinted"
            onClick={() => setShowFeedback((v) => !v)}
            className="shrink-0 self-start sm:self-center"
          >
            {showFeedback ? "Свернуть" : "Оставить отзыв"}
          </Button>
        )}
      </div>

      {tab === "past" && showFeedback && <FeedbackForm eventId={rsvp.eventId} />}
    </div>
  );
}

export default function MyPracticesPage() {
  const { isAuthed, ready } = useAuth();
  const [tab, setTab] = useState<Tab>("upcoming");

  const {
    data: rsvps = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["my-practices", tab],
    queryFn: () => fetchMyPractices(tab),
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
          <h1 className="text-[28px] font-bold tracking-[-0.022em]">Мои практики</h1>
          <p className="mt-3 text-label-secondary">
            Войдите, чтобы увидеть свои записи на события.
          </p>
        </div>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-8 max-sm:pb-28">
      <Link href="/" className="mb-4 inline-flex items-center text-[17px] text-accent">
        ‹ События
      </Link>
      <h1 className="text-[28px] font-bold tracking-[-0.022em]">Мои практики</h1>

      <div className="mt-4 flex gap-2 overflow-x-auto pb-1">
        {TABS.map((t) => (
          <FilterChip
            key={t.key}
            label={t.label}
            active={tab === t.key}
            onClick={() => setTab(t.key)}
          />
        ))}
      </div>

      {isLoading ? (
        <p className="mt-8 text-label-secondary">Загрузка…</p>
      ) : isError ? (
        <p className="mt-8 text-label-secondary">Не удалось загрузить данные.</p>
      ) : rsvps.length === 0 ? (
        <p className="mt-8 text-label-secondary">
          {tab === "upcoming"
            ? "Пока нет предстоящих записей."
            : "Пока ничего прошедшего."}
        </p>
      ) : (
        <div className="mt-6 flex flex-col gap-3">
          {rsvps.map((rsvp) => (
            <PracticeRow key={rsvp.id} rsvp={rsvp} tab={tab} />
          ))}
        </div>
      )}
    </main>
  );
}
