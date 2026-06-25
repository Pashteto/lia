"use client";

import { useEffect, useState } from "react";

import {
  type AdminEvent,
  listModerationEvents,
  reinstateEvent,
  takedownEvent,
} from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { FilterChip } from "@/components/ui/FilterChip";
import { cn } from "@/lib/cn";

type Tab = "published" | "rejected";

export default function ModerationQueue() {
  const [tab, setTab] = useState<Tab>("published");
  const [items, setItems] = useState<AdminEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0); // bump to reload
  const [pending, setPending] = useState<AdminEvent | null>(null); // take-down target
  const [reason, setReason] = useState("");
  const [actionError, setActionError] = useState("");

  useEffect(() => {
    let cancelled = false;
    listModerationEvents(tab)
      .then((data) => { if (!cancelled) { setItems(data); setLoading(false); } })
      .catch(() => { if (!cancelled) { setItems([]); setLoading(false); } });
    return () => { cancelled = true; };
  }, [tab, tick]);

  function reload() {
    setLoading(true);
    setTick((n) => n + 1);
  }

  async function confirmTakedown() {
    if (!pending || !reason.trim()) return;
    try {
      await takedownEvent(pending.id, reason.trim());
      setActionError("");
      setPending(null);
      setReason("");
      reload();
    } catch {
      setActionError("Не удалось снять событие");
    }
  }

  async function onReinstate(id: string) {
    try {
      await reinstateEvent(id);
      setActionError("");
      reload();
    } catch {
      setActionError("Не удалось вернуть событие");
    }
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-6">
      <h1 className="mb-6 text-2xl font-semibold">Модерация событий</h1>

      {/* Tab bar */}
      <div className="mb-6 flex gap-2">
        <FilterChip
          label="Опубликованные"
          active={tab === "published"}
          onClick={() => setTab("published")}
        />
        <FilterChip
          label="Снятые"
          active={tab === "rejected"}
          onClick={() => setTab("rejected")}
        />
      </div>

      {/* Content */}
      {loading ? (
        <p className="text-[15px] text-label-secondary">Загрузка…</p>
      ) : items.length === 0 ? (
        <p className="text-[15px] text-label-secondary">Пусто.</p>
      ) : (
        <ul className="space-y-3">
          {items.map((e) => (
            <li
              key={e.id}
              className="flex items-center justify-between gap-4 rounded-card bg-bg-secondary p-4 shadow-card-subtle"
            >
              <div className="min-w-0 flex-1 space-y-0.5">
                <div className="text-[16px] font-semibold leading-snug">
                  {e.title}
                </div>
                <div className="text-[13px] text-label-secondary">
                  {[
                    e.organizer_name,
                    new Date(e.starts_at).toLocaleString("ru-RU"),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </div>
                {tab === "rejected" && e.reason ? (
                  <div className="text-[13px] text-red-500">
                    Причина: {e.reason}
                  </div>
                ) : null}
              </div>
              <div className="shrink-0">
                {tab === "published" ? (
                  <Button
                    variant="tinted"
                    onClick={() => setPending(e)}
                    className="text-red-500 hover:bg-red-500/10"
                  >
                    Снять
                  </Button>
                ) : (
                  <Button variant="tinted" onClick={() => onReinstate(e.id)}>
                    Вернуть
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      {/* Take-down reason modal */}
      {pending ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          onClick={() => {
            setPending(null);
            setReason("");
            setActionError("");
          }}
        >
          <div
            className={cn(
              "w-full max-w-md rounded-card bg-bg p-6",
              "shadow-card-subtle",
            )}
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="mb-4 text-[17px] font-medium">
              Снять «{pending.title}»?
            </h2>
            <textarea
              value={reason}
              onChange={(ev) => setReason(ev.target.value)}
              placeholder="Причина снятия (обязательно)"
              rows={3}
              className="mb-4 w-full resize-none rounded-control bg-fill px-3.5 py-2.5 text-[15px] text-label outline-none placeholder:text-label-secondary"
            />
            {actionError && (
              <div className="mb-4 text-[13px] text-red-500">{actionError}</div>
            )}
            <div className="flex justify-end gap-2">
              <Button
                variant="plain"
                onClick={() => {
                  setPending(null);
                  setReason("");
                  setActionError("");
                }}
              >
                Отмена
              </Button>
              <Button
                variant="filled"
                disabled={!reason.trim()}
                onClick={confirmTakedown}
                className="bg-red-500 hover:opacity-90 disabled:opacity-40"
              >
                Снять
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
