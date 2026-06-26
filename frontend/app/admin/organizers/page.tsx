"use client";

import { useState } from "react";

import {
  searchOrganizers,
  getAdminOrganizer,
  revokeOrganizer,
  setOrganizerAutoVerify,
  type AdminOrganizer,
} from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { cn } from "@/lib/cn";

export default function AdminOrganizersPage() {
  const [q, setQ] = useState("");
  const [results, setResults] = useState<AdminOrganizer[]>([]);
  const [selected, setSelected] = useState<AdminOrganizer | null>(null);
  const [revokeReason, setRevokeReason] = useState("");
  const [actionError, setActionError] = useState("");
  const [searching, setSearching] = useState(false);

  async function doSearch() {
    if (!q.trim()) return;
    try {
      setSearching(true);
      setActionError("");
      setResults(await searchOrganizers(q.trim()));
    } catch {
      setActionError("Ошибка поиска");
    } finally {
      setSearching(false);
    }
  }

  async function open(id: string) {
    try {
      setActionError("");
      setRevokeReason("");
      setSelected(await getAdminOrganizer(id));
    } catch {
      setActionError("Не удалось загрузить организатора");
    }
  }

  async function onRevoke() {
    if (!selected) return;
    if (!revokeReason.trim()) {
      setActionError("Укажите причину отзыва");
      return;
    }
    try {
      await revokeOrganizer(selected.id, revokeReason.trim());
      setRevokeReason("");
      setActionError("");
      await open(selected.id);
    } catch {
      setActionError("Не удалось отозвать подтверждение");
    }
  }

  async function onToggleAuto() {
    if (!selected) return;
    try {
      setActionError("");
      await setOrganizerAutoVerify(selected.id, !selected.auto_verify);
      await open(selected.id);
    } catch {
      setActionError("Не удалось изменить авто-подтверждение");
    }
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-6">
      <h1 className="mb-6 text-2xl font-semibold">Организаторы</h1>

      {/* Search bar */}
      <div className="mb-6 flex gap-2">
        <input
          className="flex-1 rounded-control bg-fill px-3.5 py-2.5 text-[15px] text-label outline-none placeholder:text-label-secondary"
          placeholder="Поиск по названию или email"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && doSearch()}
        />
        <Button variant="filled" onClick={doSearch} disabled={searching || !q.trim()}>
          Найти
        </Button>
      </div>

      {actionError && (
        <div className="mb-4 text-[13px] text-red-500">{actionError}</div>
      )}

      {/* Search results */}
      {results.length > 0 && (
        <ul className="mb-6 space-y-2">
          {results.map((o) => (
            <li key={o.id}>
              <button
                onClick={() => open(o.id)}
                className={cn(
                  "w-full rounded-card bg-bg-secondary p-4 text-left shadow-card-subtle",
                  "transition hover:opacity-80 active:scale-[0.99] motion-reduce:transform-none motion-reduce:transition-none",
                  selected?.id === o.id && "ring-2 ring-accent/40",
                )}
              >
                <span className="text-[16px] font-semibold">{o.name}</span>{" "}
                <span className="text-[13px] text-label-secondary">
                  · {statusLabel(o.verification_status)}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {/* Detail panel */}
      {selected && (
        <div className="rounded-card bg-bg-secondary p-6 shadow-card-subtle space-y-5">
          <div>
            <h2 className="text-[20px] font-bold tracking-[-0.022em]">{selected.name}</h2>
            <p className="mt-1 text-[13px] text-label-secondary">
              Статус: {statusLabel(selected.verification_status)}
            </p>
            {selected.description && (
              <p className="mt-2 text-[15px]">{selected.description}</p>
            )}
            {selected.website_url && (
              <p className="mt-1 text-[13px] text-label-secondary">{selected.website_url}</p>
            )}
          </div>

          {/* Auto-verify toggle */}
          <label className="flex cursor-pointer items-center gap-3 text-[15px]">
            <input
              type="checkbox"
              checked={selected.auto_verify}
              onChange={onToggleAuto}
              className="h-4 w-4 accent-accent"
            />
            <span>
              Авто-подтверждение{" "}
              <span className="text-[13px] text-label-secondary">
                (заявки этого организатора минуют очередь модерации)
              </span>
            </span>
          </label>

          {/* Revoke section — only when verified */}
          {selected.verification_status === "verified" && (
            <div className="space-y-2">
              <p className="text-[13px] font-medium text-label-secondary">
                Отозвать подтверждение
              </p>
              <textarea
                className="w-full resize-none rounded-control bg-fill px-3.5 py-2.5 text-[15px] text-label outline-none placeholder:text-label-secondary"
                rows={2}
                placeholder="Причина отзыва (обязательно)"
                value={revokeReason}
                onChange={(e) => setRevokeReason(e.target.value)}
              />
              <Button
                variant="tinted"
                disabled={!revokeReason.trim()}
                onClick={onRevoke}
                className="text-red-500 hover:bg-red-500/10 disabled:opacity-40"
              >
                Отозвать подтверждение
              </Button>
            </div>
          )}

          {/* Verification history */}
          {selected.history && selected.history.length > 0 && (
            <div className="space-y-1">
              <p className="text-[13px] font-medium text-label-secondary">История</p>
              <ul className="space-y-1">
                {selected.history.map((h, i) => (
                  <li key={i} className="text-[13px] text-label-secondary">
                    <span className="text-label">{h.from_status} → {h.to_status}</span>
                    {h.reason ? ` · ${h.reason}` : ""}
                    {" · "}
                    {new Date(h.created_at).toLocaleString("ru-RU")}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function statusLabel(s: string): string {
  switch (s) {
    case "pending": return "На проверке";
    case "verified": return "Подтверждён";
    case "rejected": return "Отклонён";
    case "draft": return "Черновик";
    default: return s;
  }
}
