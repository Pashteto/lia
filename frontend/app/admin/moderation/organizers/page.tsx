"use client";

import { useEffect, useState } from "react";

import {
  listModerationOrganizers,
  verifyOrganizer,
  rejectOrganizer,
  type AdminOrganizer,
  type VerificationStatus,
} from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { FilterChip } from "@/components/ui/FilterChip";
import { cn } from "@/lib/cn";

const TABS: { key: VerificationStatus; label: string }[] = [
  { key: "pending", label: "На проверке" },
  { key: "verified", label: "Подтверждённые" },
  { key: "rejected", label: "Отклонённые" },
];

export default function ModerationOrganizersPage() {
  const [tab, setTab] = useState<VerificationStatus>("pending");
  const [items, setItems] = useState<AdminOrganizer[]>([]);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);
  const [rejectTarget, setRejectTarget] = useState<AdminOrganizer | null>(null);
  const [reason, setReason] = useState("");
  const [actionError, setActionError] = useState("");
  const [acting, setActing] = useState(false);

  useEffect(() => {
    let cancelled = false;
    listModerationOrganizers(tab)
      .then((data) => { if (!cancelled) { setItems(data); setLoading(false); setActionError(""); } })
      .catch(() => { if (!cancelled) { setItems([]); setLoading(false); setActionError("Ошибка загрузки"); } });
    return () => { cancelled = true; };
  }, [tab, tick]);

  function reload() {
    setLoading(true);
    setTick((n) => n + 1);
  }

  async function onVerify(id: string) {
    try {
      await verifyOrganizer(id);
      setActionError("");
      reload();
    } catch {
      setActionError("Не удалось подтвердить организатора");
    }
  }

  async function confirmReject() {
    if (!rejectTarget || acting || !reason.trim()) return;
    setActing(true);
    try {
      await rejectOrganizer(rejectTarget.id, reason.trim());
      setActionError("");
      setRejectTarget(null);
      setReason("");
      reload();
    } catch (err) {
      if (err instanceof Error && err.message.includes("409")) {
        // Already rejected — the mutation effectively happened, refetch quietly.
        setActionError("");
        setRejectTarget(null);
        setReason("");
        reload();
      } else {
        setActionError("Не удалось отклонить организатора");
      }
    } finally {
      setActing(false);
    }
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-6">
      <h1 className="mb-6 text-2xl font-semibold">Модерация организаторов</h1>

      {/* Tab bar */}
      <div className="mb-6 flex gap-2">
        {TABS.map((t) => (
          <FilterChip
            key={t.key}
            label={t.label}
            active={tab === t.key}
            onClick={() => setTab(t.key)}
          />
        ))}
      </div>

      {actionError && (
        <div className="mb-4 text-[13px] text-red-500">{actionError}</div>
      )}

      {/* Content */}
      {loading ? (
        <p className="text-[15px] text-label-secondary">Загрузка…</p>
      ) : items.length === 0 ? (
        <p className="text-[15px] text-label-secondary">Пусто.</p>
      ) : (
        <ul className="space-y-3">
          {items.map((o) => (
            <li
              key={o.id}
              className="flex items-center justify-between gap-4 rounded-card bg-bg-secondary p-4 shadow-card-subtle"
            >
              <div className="min-w-0 flex-1 space-y-0.5">
                <div className="text-[16px] font-semibold leading-snug">
                  {o.name}
                </div>
                {o.website_url && (
                  <div className="text-[13px] text-label-secondary">
                    {o.website_url}
                  </div>
                )}
                {o.description && (
                  <div className="text-[13px] text-label-secondary">
                    {o.description}
                  </div>
                )}
                {tab === "rejected" && o.latest_reason ? (
                  <div className="text-[13px] text-red-500">
                    Причина: {o.latest_reason}
                  </div>
                ) : null}
              </div>
              {tab === "pending" && (
                <div className="flex shrink-0 gap-2">
                  <Button
                    variant="filled"
                    onClick={() => onVerify(o.id)}
                  >
                    Подтвердить
                  </Button>
                  <Button
                    variant="tinted"
                    onClick={() => {
                      setRejectTarget(o);
                      setReason("");
                      setActionError("");
                    }}
                    className="text-red-500 hover:bg-red-500/10"
                  >
                    Отклонить
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      {/* Reject reason modal */}
      {rejectTarget ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          onClick={() => {
            setRejectTarget(null);
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
              Отклонить «{rejectTarget.name}»?
            </h2>
            <textarea
              value={reason}
              onChange={(ev) => setReason(ev.target.value)}
              placeholder="Причина отклонения (обязательно)"
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
                  setRejectTarget(null);
                  setReason("");
                  setActionError("");
                }}
              >
                Отмена
              </Button>
              <Button
                variant="filled"
                disabled={acting || !reason.trim()}
                onClick={confirmReject}
                className="bg-red-500 hover:opacity-90 disabled:opacity-40"
              >
                {acting ? "Отклоняем…" : "Отклонить"}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
