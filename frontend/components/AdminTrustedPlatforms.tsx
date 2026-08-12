"use client";

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import {
  type AdminTrustedPlatform,
  adminAddTrustedPlatform,
  adminListTrustedPlatforms,
  adminRemoveTrustedPlatform,
} from "@/lib/api";
import { cn } from "@/lib/cn";

const CATEGORIES = [
  { value: "ticketing", label: "Билеты" },
  { value: "afisha", label: "Афиша" },
  { value: "gov", label: "Госорганы" },
  { value: "social", label: "Соцсети" },
] as const;

function categoryLabel(value: string): string {
  return CATEGORIES.find((c) => c.value === value)?.label ?? value;
}

const inputClass =
  "swiss-focus w-full border border-muted-2 bg-transparent px-[10px] py-[8px] text-[12px] text-on-surface outline-none placeholder:text-muted-2";

/** Whitelist of external registration platforms — events linking to a listed
 * domain skip the pending-review moderation step. See spec
 * docs/superpowers/specs/2026-08-12-external-registration-whitelist-design.md. */
export function AdminTrustedPlatforms() {
  const [rows, setRows] = useState<AdminTrustedPlatform[] | null>(null);
  const [error, setError] = useState(false);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState("");
  const [tick, setTick] = useState(0);

  const [domainSuffix, setDomainSuffix] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [category, setCategory] = useState<string>(CATEGORIES[0].value);

  useEffect(() => {
    let cancelled = false;
    adminListTrustedPlatforms()
      .then((list) => {
        if (!cancelled) {
          setRows(list);
          setError(false);
        }
      })
      .catch(() => {
        if (!cancelled) setError(true);
      });
    return () => {
      cancelled = true;
    };
  }, [tick]);

  function reload() {
    setRows(null);
    setTick((t) => t + 1);
  }

  async function onAdd() {
    if (busy) return;
    const domain = domainSuffix.trim().toLowerCase();
    const name = displayName.trim();
    if (!domain || !name) {
      setActionError("Заполните домен и название");
      return;
    }
    setBusy(true);
    setActionError("");
    try {
      await adminAddTrustedPlatform({ domainSuffix: domain, displayName: name, category });
      setDomainSuffix("");
      setDisplayName("");
      setCategory(CATEGORIES[0].value);
      reload();
    } catch {
      setActionError("Не удалось добавить платформу");
    } finally {
      setBusy(false);
    }
  }

  async function onRemove(id: string) {
    if (busy) return;
    setBusy(true);
    setActionError("");
    try {
      await adminRemoveTrustedPlatform(id);
      reload();
    } catch {
      setActionError("Не удалось отключить платформу");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="border-t border-paper">
      <div className="flex items-baseline justify-between gap-[12px] border-b border-paper px-[20px] py-[14px] max-sm:px-[14px]">
        <h2 className="text-[15px] font-black leading-[1.05] tracking-[-0.02em]">
          Доверенные платформы
        </h2>
        <span className="cap text-muted-2">Внешняя регистрация</span>
      </div>

      {actionError ? (
        <p className="border-b border-rule-inner px-[20px] py-[8px] text-[11px] text-signal max-sm:px-[14px]">
          {actionError}
        </p>
      ) : null}

      <div className="flex flex-wrap items-end gap-[10px] border-b border-rule-inner px-[20px] py-[14px] max-sm:px-[14px]">
        <label className="flex min-w-[180px] flex-1 flex-col gap-[4px]">
          <span className="cap text-muted-2">Домен</span>
          <input
            value={domainSuffix}
            onChange={(e) => setDomainSuffix(e.target.value)}
            placeholder="timepad.ru"
            className={inputClass}
          />
        </label>
        <label className="flex min-w-[180px] flex-1 flex-col gap-[4px]">
          <span className="cap text-muted-2">Название</span>
          <input
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="TimePad"
            className={inputClass}
          />
        </label>
        <label className="flex min-w-[140px] flex-col gap-[4px]">
          <span className="cap text-muted-2">Категория</span>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className={inputClass}
          >
            {CATEGORIES.map((c) => (
              <option key={c.value} value={c.value}>
                {c.label}
              </option>
            ))}
          </select>
        </label>
        <Button variant="inverted" type="button" disabled={busy} onClick={onAdd} className="min-h-[44px]">
          Добавить
        </Button>
      </div>

      {rows === null ? (
        <div className="flex flex-col gap-[8px] px-[20px] py-[16px] max-sm:px-[14px]">
          <Skeleton className="h-[36px] w-full" />
          <Skeleton className="h-[36px] w-full" />
        </div>
      ) : error ? (
        <EmptyState
          numeral="!"
          title="Не удалось загрузить"
          text="Проверьте соединение и попробуйте ещё раз."
          actions={
            <Button variant="inverted" type="button" onClick={reload}>
              Повторить
            </Button>
          }
        />
      ) : rows.length === 0 ? (
        <EmptyState numeral="00" title="Список пуст" text="Добавьте первую доверенную платформу." />
      ) : (
        <div className="flex flex-col">
          <div className="grid grid-cols-[1fr_1fr_110px_90px_100px] border-b border-paper bg-surface-head">
            {(["Домен", "Название", "Категория", "Статус", ""] as const).map((h, i) => (
              <span key={h + i} className="cap px-[10px] py-[6px] text-muted-2">
                {h}
              </span>
            ))}
          </div>
          {rows.map((row) => (
            <div
              key={row.id}
              className="grid grid-cols-[1fr_1fr_110px_90px_100px] items-center border-b border-rule-inner"
            >
              <span className="px-[10px] py-[9px] font-mono text-[11px]">{row.domainSuffix}</span>
              <span className="px-[10px] py-[9px] text-[12px] font-bold">{row.displayName}</span>
              <span className="px-[10px] py-[9px]">
                <Chip as="span" variant="dark-muted" className="px-[6px] py-[2px] text-[8px]">
                  {categoryLabel(row.category)}
                </Chip>
              </span>
              <span className="px-[10px] py-[9px]">
                <Chip
                  as="span"
                  variant={row.isActive ? "dark-active" : "dark-muted"}
                  className={cn("px-[6px] py-[2px] text-[8px]", !row.isActive && "opacity-60")}
                >
                  {row.isActive ? "Активна" : "Отключена"}
                </Chip>
              </span>
              <span className="px-[10px] py-[9px]">
                {row.isActive ? (
                  <Button
                    variant="dark-ghost"
                    size="sm"
                    type="button"
                    disabled={busy}
                    onClick={() => onRemove(row.id)}
                  >
                    Отключить
                  </Button>
                ) : null}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
