"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { complaintsCountRu } from "@/lib/admin-copy";
import { orgEventStatusLabel } from "@/lib/org-event-status";
import { padCount } from "@/lib/org-seats";
import {
  COMPLAINT_CATEGORIES,
  listComplaints,
  resolveComplaints,
  type ComplaintCategory,
  type ComplaintGroup,
} from "@/lib/api";

const CATEGORY_LABEL = new Map(COMPLAINT_CATEGORIES.map((c) => [c.value, c.label]));

export default function ComplaintsInbox() {
  const [items, setItems] = useState<ComplaintGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);
  const [pending, setPending] = useState<ComplaintGroup | null>(null);
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [acting, setActing] = useState(false);

  useEffect(() => {
    let cancelled = false;
    listComplaints()
      .then((data) => {
        if (!cancelled) {
          setItems(data);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setItems([]);
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [tick]);

  function reload() {
    setLoading(true);
    setTick((n) => n + 1);
  }

  async function confirmTakedown() {
    if (!pending || acting || !reason.trim()) return;
    setActing(true);
    try {
      await resolveComplaints(pending.event_id, "takedown", reason.trim());
      setError("");
      setPending(null);
      setReason("");
      reload();
    } catch (err) {
      if (err instanceof Error && err.message.includes("409")) {
        setError("");
        setPending(null);
        setReason("");
        reload();
      } else {
        setError("Не удалось снять событие");
      }
    } finally {
      setActing(false);
    }
  }

  async function onDismiss(eventId: string) {
    if (acting) return;
    setActing(true);
    try {
      await resolveComplaints(eventId, "dismiss", "");
      setError("");
      reload();
    } catch (err) {
      if (err instanceof Error && err.message.includes("409")) {
        setError("");
        reload();
      } else {
        setError("Не удалось отклонить жалобы");
      }
    } finally {
      setActing(false);
    }
  }

  return (
    <div className="flex flex-col">
      <div className="flex items-baseline justify-between gap-[12px] border-b border-paper px-[20px] py-[14px] max-sm:px-[14px]">
        <h1 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">Жалобы</h1>
        <span className="cap text-muted-2">
          {loading ? "Загрузка" : `Событий · ${padCount(items.length)}`}
        </span>
      </div>

      {error ? (
        <p className="border-b border-rule-inner px-[20px] py-[8px] text-[11px] text-signal max-sm:px-[14px]">
          {error}
        </p>
      ) : null}

      {loading ? (
        <div className="flex flex-col gap-[8px] px-[20px] py-[16px] max-sm:px-[14px]">
          <Skeleton className="h-[72px] w-full" />
          <Skeleton className="h-[72px] w-full" />
        </div>
      ) : items.length === 0 ? (
        <EmptyState
          numeral="00"
          title="Жалоб нет"
          text="Никто не пожаловался на события. Жалобы появятся здесь, как только придут."
          actions={
            <Link
              href="/admin"
              className="swiss-focus inline-flex min-h-[44px] items-center justify-center bg-paper px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink"
            >
              К обзору
            </Link>
          }
        />
      ) : (
        <ul className="flex flex-col">
          {items.map((g) => (
            <li
              key={g.event_id}
              className="flex items-start justify-between gap-[16px] border-b border-rule-inner px-[20px] py-[14px] max-sm:px-[14px]"
            >
              <div className="min-w-0 flex-1 space-y-[6px]">
                <Link
                  href={`/events/${g.event_id}`}
                  className="text-[12px] font-bold leading-snug hover:underline"
                >
                  {g.event_title}
                </Link>
                <div className="cap font-mono text-muted-2">
                  {complaintsCountRu(g.report_count)} · {orgEventStatusLabel(g.event_status)}
                </div>
                <div className="flex flex-wrap gap-[6px]">
                  {Object.entries(g.categories).map(([cat, n]) => (
                    <Chip key={cat} variant="dark-muted" as="span">
                      {CATEGORY_LABEL.get(cat as ComplaintCategory) ?? cat}: {n}
                    </Chip>
                  ))}
                </div>
                {g.latest_note ? (
                  <div className="text-[11.5px] text-text-dim">«{g.latest_note}»</div>
                ) : null}
              </div>
              <div className="flex shrink-0 flex-col gap-[8px]">
                {g.event_status === "published" ? (
                  <Button variant="destructive" size="sm" onClick={() => setPending(g)}>
                    Снять
                  </Button>
                ) : null}
                <Button
                  variant="dark-ghost"
                  size="sm"
                  disabled={acting}
                  onClick={() => onDismiss(g.event_id)}
                >
                  Отклонить
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {pending ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-[16px]"
          onClick={() => setPending(null)}
        >
          <div
            className="w-full max-w-md border border-paper bg-surface-head p-[14px]"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="mb-[12px] text-[14px] font-black leading-[1.05] tracking-[-0.02em]">
              Снять «{pending.event_title}»
            </h2>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Причина снятия (обязательно)"
              rows={3}
              className="swiss-focus w-full border border-muted-2 bg-transparent px-[10px] py-[8px] text-[12px] text-on-surface outline-none"
            />
            <div className="mt-[12px] flex justify-end gap-[8px]">
              <Button variant="dark-ghost" size="sm" onClick={() => setPending(null)}>
                Отмена
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={confirmTakedown}
                disabled={acting || !reason.trim()}
              >
                {acting ? "Снимаем…" : "Снять и закрыть жалобы"}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
