"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { FilterChip } from "@/components/ui/FilterChip";
import { cancelRsvp, fetchMyApplications } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { formatEventDate } from "@/lib/format";
import type { Rsvp, RsvpStatus } from "@/lib/types";

type Tab = "applied" | "accepted" | "declined" | "withdrawn";

const TABS: { key: Tab; label: string }[] = [
  { key: "applied", label: "В ожидании" },
  { key: "accepted", label: "Принятые" },
  { key: "declined", label: "Отклонённые" },
  { key: "withdrawn", label: "Отозванные" },
];

const STATUS_CHIP: Partial<Record<RsvpStatus, { label: string; className: string }>> = {
  applied: { label: "ожидает ответа", className: "bg-amber-500/15 text-amber-700 dark:text-amber-400" },
  accepted: { label: "принята", className: "bg-green-500/15 text-green-700 dark:text-green-400" },
  declined: { label: "отклонена", className: "bg-red-500/15 text-red-700 dark:text-red-400" },
  withdrawn: { label: "отозвана", className: "bg-fill text-label-secondary" },
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

function ApplicationCard({ rsvp, tab }: { rsvp: Rsvp; tab: Tab }) {
  const queryClient = useQueryClient();
  const [expanded, setExpanded] = useState(false);
  const [withdrawing, setWithdrawing] = useState(false);

  const event = rsvp.event;
  const canWithdraw = tab === "applied";

  async function handleWithdraw() {
    setWithdrawing(true);
    try {
      await cancelRsvp(rsvp.eventId);
      await queryClient.invalidateQueries({ queryKey: ["my-applications", tab] });
    } finally {
      setWithdrawing(false);
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-card bg-bg-secondary p-4 shadow-card-subtle">
      <div className="flex flex-wrap items-center gap-2">
        <StatusChip status={rsvp.status} />
        {event && (
          <span className="text-[13px] text-label-secondary">
            {formatEventDate(event.startsAt)}
          </span>
        )}
        <span className="text-[13px] text-label-secondary ml-auto">
          {formatEventDate(rsvp.createdAt)}
        </span>
      </div>

      {event ? (
        <Link
          href={`/events/${rsvp.eventId}`}
          className="text-[17px] font-semibold leading-snug hover:text-accent transition-colors"
        >
          {event.title}
        </Link>
      ) : (
        <span className="text-[17px] font-semibold leading-snug text-label-secondary">
          Событие #{rsvp.eventId.slice(0, 8)}
        </span>
      )}

      {event?.curatorQuestion && (
        <div className="space-y-1">
          <p className="text-[13px] font-medium text-label-secondary">
            {event.curatorQuestion}
          </p>
          {rsvp.applicationAnswer ? (
            <div>
              <p
                className={`text-[15px] leading-relaxed ${!expanded ? "line-clamp-3" : ""}`}
              >
                {rsvp.applicationAnswer}
              </p>
              {rsvp.applicationAnswer.length > 120 && (
                <button
                  type="button"
                  onClick={() => setExpanded((v) => !v)}
                  className="mt-1 text-[13px] text-accent hover:opacity-70 transition-opacity"
                >
                  {expanded ? "Свернуть" : "Читать полностью"}
                </button>
              )}
            </div>
          ) : (
            <p className="text-[15px] text-label-secondary italic">Ответ не указан</p>
          )}
        </div>
      )}

      {canWithdraw && (
        <div className="pt-1">
          <Button
            variant="tinted"
            onClick={handleWithdraw}
            disabled={withdrawing}
            className="text-red-600 bg-red-500/10 hover:bg-red-500/20"
          >
            {withdrawing ? "…" : "Отозвать заявку"}
          </Button>
        </div>
      )}
    </div>
  );
}

export default function MyApplicationsPage() {
  const { isAuthed, ready } = useAuth();
  const [tab, setTab] = useState<Tab>("applied");

  const {
    data: rsvps = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["my-applications", tab],
    queryFn: () => fetchMyApplications(tab),
    enabled: ready && isAuthed,
  });

  if (!ready) {
    return <div className="min-h-screen bg-bg-grouped" />;
  }

  if (!isAuthed) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16 text-center">
        <h1 className="text-[28px] font-bold tracking-[-0.022em]">Мои заявки</h1>
        <p className="mt-3 text-label-secondary">
          Войдите, чтобы увидеть свои заявки на события.
        </p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <h1 className="text-[28px] font-bold tracking-[-0.022em]">Мои заявки</h1>

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
        <p className="mt-8 text-label-secondary">Пока ничего.</p>
      ) : (
        <div className="mt-6 flex flex-col gap-3">
          {rsvps.map((rsvp) => (
            <ApplicationCard key={rsvp.id} rsvp={rsvp} tab={tab} />
          ))}
        </div>
      )}
    </main>
  );
}
