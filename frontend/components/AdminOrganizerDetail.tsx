"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusChip } from "@/components/ui/StatusChip";
import {
  getAdminOrganizer,
  reinstateEvent,
  setOrganizerDailyLimit,
  takedownEvent,
  type AdminEvent,
} from "@/lib/api";
import { concatenateReasons, REJECT_REASON_CHIPS } from "@/lib/admin-reject-reasons";
import { formatShortDate } from "@/lib/format";
import { orgEventStatusLabel } from "@/lib/org-event-status";

const VERIFICATION_LABEL: Record<string, string> = {
  draft: "Черновик",
  pending: "На проверке",
  verified: "Верифицирован",
  rejected: "Отклонён",
};

/** Moderation acts on published events; a takedown is undone by reinstating.
 * A takedown needs a reason — the backend rejects an empty one, and the
 * organizer is shown what it said. */
function EventRow({
  event,
  onChanged,
}: {
  event: AdminEvent;
  onChanged: () => void;
}) {
  const [picking, setPicking] = useState(false);
  const [reasons, setReasons] = useState<Set<string>>(new Set());

  const takedown = useMutation({
    mutationFn: () => takedownEvent(event.id, concatenateReasons([...reasons])),
    onSuccess: () => {
      setPicking(false);
      setReasons(new Set());
      onChanged();
    },
  });
  const reinstate = useMutation({
    mutationFn: () => reinstateEvent(event.id),
    onSuccess: onChanged,
  });
  const busy = takedown.isPending || reinstate.isPending;

  return (
    <div className="grid grid-cols-[64px_1fr_120px_160px] items-center border-b border-muted-2/40">
      <span className="border-r border-muted-2/40 px-[8px] py-[10px] text-center font-mono text-[10px] font-bold">
        {formatShortDate(event.starts_at)}
      </span>
      <span className="min-w-0 border-r border-muted-2/40 px-[12px] py-[9px]">
        <Link href={`/events/${event.id}`} className="swiss-focus block truncate text-[12.5px] font-bold hover:underline">
          {event.title}
        </Link>
        {event.reason ? (
          <span className="mt-[2px] block text-[10px] text-signal">Причина: {event.reason}</span>
        ) : null}
      </span>
      <span className="flex items-center border-r border-muted-2/40 px-[8px] py-[9px]">
        <StatusChip
          status={orgEventStatusLabel(event.status)}
          className="px-[6px] py-[3px] text-[8px]"
        />
      </span>
      <span className="flex gap-[5px] px-[8px] py-[7px]">
        {event.status === "published" ? (
          <Button
            variant="destructive"
            size="sm"
            disabled={busy}
            onClick={() => setPicking((v) => !v)}
            className="min-h-[44px] flex-1 px-[4px] text-[8px] tracking-[0.05em]"
          >
            СНЯТЬ
          </Button>
        ) : event.status === "rejected" ? (
          <Button
            variant="inverted"
            size="sm"
            disabled={busy}
            onClick={() => reinstate.mutate()}
            className="min-h-[44px] flex-1 px-[4px] text-[8px] tracking-[0.05em]"
          >
            ВЕРНУТЬ
          </Button>
        ) : (
          <span className="cap flex-1 self-center text-muted-2">Нет действий</span>
        )}
      </span>

      {picking ? (
        <div className="col-span-4 flex flex-wrap items-center gap-[6px] border-t border-muted-2/40 px-[12px] py-[10px]">
          <span className="cap">Причина</span>
          {REJECT_REASON_CHIPS.map((label) => (
            <Chip
              key={label}
              variant={reasons.has(label) ? "active" : "default"}
              aria-pressed={reasons.has(label)}
              onClick={() =>
                setReasons((prev) => {
                  const next = new Set(prev);
                  if (next.has(label)) next.delete(label);
                  else next.add(label);
                  return next;
                })
              }
            >
              {label}
            </Chip>
          ))}
          <Button
            variant="destructive"
            size="sm"
            disabled={busy || reasons.size === 0}
            onClick={() => takedown.mutate()}
            className="min-h-[44px] px-[10px] text-[8px]"
          >
            {takedown.isPending ? "…" : "ПОДТВЕРДИТЬ"}
          </Button>
        </div>
      ) : null}
      {takedown.isError || reinstate.isError ? (
        <span className="col-span-4 px-[12px] pb-[6px] text-[10px] text-signal">
          Не удалось выполнить действие.
        </span>
      ) : null}
    </div>
  );
}

/** The per-organizer override of the daily creation cap. */
function DailyLimitControl({
  id,
  current,
  onChanged,
}: {
  id: string;
  current?: number;
  onChanged: () => void;
}) {
  const [value, setValue] = useState(current == null ? "" : String(current));

  const save = useMutation({
    mutationFn: () => {
      const trimmed = value.trim();
      if (trimmed === "") return setOrganizerDailyLimit(id, null);
      const n = Number(trimmed);
      if (!Number.isInteger(n) || n < 0) throw new Error("invalid");
      return setOrganizerDailyLimit(id, n);
    },
    onSuccess: onChanged,
  });

  return (
    <div className="flex flex-col gap-[6px] px-[20px] py-[14px]">
      <span className="cap">Лимит событий в сутки</span>
      <div className="flex items-center gap-[8px]">
        <input
          value={value}
          onChange={(e) => setValue(e.target.value.replace(/\D/g, ""))}
          inputMode="numeric"
          placeholder="по умолчанию"
          className="w-[140px] border border-muted-2 bg-transparent px-[10px] py-[8px] text-[12px] swiss-focus"
        />
        <Button
          variant="inverted"
          size="sm"
          disabled={save.isPending}
          onClick={() => save.mutate()}
          className="min-h-[44px] px-[12px] text-[9px]"
        >
          {save.isPending ? "…" : "СОХРАНИТЬ"}
        </Button>
      </div>
      <p className="text-[10px] text-muted-2">
        Пусто — общий лимит по умолчанию. 0 — без ограничения.
      </p>
      {save.isError ? (
        <p className="text-[10px] text-signal">Не удалось сохранить лимит.</p>
      ) : null}
    </div>
  );
}

export function AdminOrganizerDetail({ id }: { id: string }) {
  const qc = useQueryClient();
  const org = useQuery({
    queryKey: ["admin-organizer", id],
    queryFn: () => getAdminOrganizer(id),
  });

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ["admin-organizer", id] });
  };

  if (org.isLoading) {
    return (
      <main className="mx-auto max-w-[1360px]">
        <Skeleton className="h-[70px] w-full border-x-0 border-t-0" />
        <Skeleton className="h-[240px] w-full border-x-0 border-t-0" />
      </main>
    );
  }

  if (org.isError || !org.data) {
    return (
      <EmptyState numeral="!" title="Организатор не найден" text="Вернитесь к списку организаторов." />
    );
  }

  const o = org.data;
  const events = o.events ?? [];
  const published = events.filter((e) => e.status === "published").length;

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px]">
      <div className="border-b border-muted-2/40 px-[20px] py-[10px]">
        <Link href="/admin/organizers" className="cap swiss-focus hover:underline">
          ← Все организаторы
        </Link>
      </div>

      <div className="flex items-start justify-between gap-[16px] border-b border-muted-2/40 px-[20px] py-[16px]">
        <div>
          <h1 className="text-[26px] font-black leading-[0.94] tracking-[-0.03em]">{o.name}</h1>
          {o.website_url ? (
            <p className="cap mt-[4px]">{o.website_url}</p>
          ) : null}
          {o.description ? (
            <p className="mt-[6px] max-w-[60ch] text-[12px] text-muted-2">{o.description}</p>
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-[8px]">
          <Chip as="span" variant="default">
            {VERIFICATION_LABEL[o.verification_status] ?? o.verification_status}
          </Chip>
          <Link
            href={`/organizers/${o.id}`}
            className="cap swiss-focus px-[8px] py-[8px] hover:underline"
          >
            Публичная страница ↗
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-[1fr_300px] max-md:grid-cols-1">
        <div className="border-r border-muted-2/40 max-md:border-r-0">
          <div className="flex items-baseline justify-between border-b border-muted-2/40 px-[20px] py-[10px]">
            <span className="cap">События · {events.length}</span>
            <span className="cap">Опубликовано · {published}</span>
          </div>
          {events.length === 0 ? (
            <EmptyState numeral="00" title="Событий нет" text="Этот организатор ещё ничего не создал." />
          ) : (
            <div className="flex flex-col">
              {events.map((e) => (
                <EventRow key={e.id} event={e} onChanged={refresh} />
              ))}
            </div>
          )}
        </div>

        <aside>
          <DailyLimitControl id={o.id} current={o.daily_event_limit} onChanged={refresh} />
          <div className="border-t border-muted-2/40 px-[20px] py-[14px]">
            <span className="cap">Жалоб</span>
            <p className="mt-[2px] font-mono text-[18px] font-bold">
              {o.complaints_count ?? 0}
            </p>
          </div>
          {o.history && o.history.length > 0 ? (
            <div className="border-t border-muted-2/40 px-[20px] py-[14px]">
              <span className="cap">История проверки</span>
              <ul className="mt-[6px] flex flex-col gap-[5px]">
                {o.history.map((h, i) => (
                  <li key={i} className="text-[11px]">
                    <span className="font-mono">{h.created_at.slice(0, 10)}</span>{" "}
                    {h.from_status} → <strong>{h.to_status}</strong>
                    {h.reason ? <span className="block text-muted-2">{h.reason}</span> : null}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </aside>
      </div>
    </main>
  );
}
