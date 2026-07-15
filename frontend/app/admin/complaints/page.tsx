"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import {
  COMPLAINT_CATEGORIES,
  listComplaints,
  resolveComplaints,
  type ComplaintCategory,
  type ComplaintGroup,
} from "@/lib/api";
import { Button } from "@/components/ui/Button";

const CATEGORY_LABEL = new Map(COMPLAINT_CATEGORIES.map((c) => [c.value, c.label]));

export default function ComplaintsInbox() {
  const [items, setItems] = useState<ComplaintGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);
  const [pending, setPending] = useState<ComplaintGroup | null>(null); // takedown target
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
        // Event already taken down — the mutation effectively happened, refetch quietly.
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
        // Already resolved — the mutation effectively happened, refetch quietly.
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
    <div className="mx-auto max-w-3xl px-4 py-6">
      <Link href="/admin" className="mb-4 inline-flex items-center text-[17px] text-accent">
        ‹ Админ
      </Link>
      <h1 className="mb-6 text-2xl font-semibold">Жалобы</h1>

      {error ? <p className="mb-4 text-[13px] text-red-500">{error}</p> : null}

      {loading ? (
        <p className="text-[15px] text-label-secondary">Загрузка…</p>
      ) : items.length === 0 ? (
        <p className="text-[15px] text-label-secondary">Жалоб нет.</p>
      ) : (
        <ul className="space-y-3">
          {items.map((g) => (
            <li
              key={g.event_id}
              className="flex items-start justify-between gap-4 rounded-card bg-bg-secondary p-4 shadow-card-subtle"
            >
              <div className="min-w-0 flex-1 space-y-1">
                <Link
                  href={`/events/${g.event_id}`}
                  className="text-[16px] font-semibold leading-snug hover:underline"
                >
                  {g.event_title}
                </Link>
                <div className="text-[13px] text-label-secondary">
                  {g.report_count} жалоб · статус: {g.event_status}
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {Object.entries(g.categories).map(([cat, n]) => (
                    <span
                      key={cat}
                      className="rounded-full bg-bg-tertiary px-2 py-0.5 text-[12px] text-label-secondary"
                    >
                      {CATEGORY_LABEL.get(cat as ComplaintCategory) ?? cat}: {n}
                    </span>
                  ))}
                </div>
                {g.latest_note ? (
                  <div className="text-[13px] text-label-secondary">«{g.latest_note}»</div>
                ) : null}
              </div>
              <div className="flex shrink-0 flex-col gap-2">
                {g.event_status === "published" ? (
                  <Button
                    variant="tinted"
                    onClick={() => setPending(g)}
                    className="text-red-500 hover:bg-red-500/10"
                  >
                    Снять
                  </Button>
                ) : null}
                <Button variant="tinted" disabled={acting} onClick={() => onDismiss(g.event_id)}>
                  Отклонить
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {pending ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          onClick={() => setPending(null)}
        >
          <div
            className="w-full max-w-md rounded-card bg-bg-secondary p-5 shadow-card"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="mb-3 text-[18px] font-semibold">Снять «{pending.event_title}»</h2>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Причина снятия (обязательно)"
              rows={3}
              className="w-full rounded-control bg-bg-tertiary p-3 text-[15px]"
            />
            <div className="mt-4 flex justify-end gap-2">
              <Button variant="tinted" onClick={() => setPending(null)}>
                Отмена
              </Button>
              <Button
                variant="filled"
                onClick={confirmTakedown}
                disabled={acting || !reason.trim()}
                className="text-red-500"
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
