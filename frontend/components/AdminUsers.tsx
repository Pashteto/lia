"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { formatRegistrationMonth } from "@/lib/admin-registration";
import { adminUserShortId } from "@/lib/admin-id";
import { isLikelyTestContent } from "@/lib/admin-test-heuristic";
import { adminUserRoleLabel } from "@/lib/admin-user-role";
import {
  hideAllHygiene,
  listAdminUsers,
  listHygieneIssues,
  type AdminUser,
  type HygieneIssue,
} from "@/lib/api";
import { cn } from "@/lib/cn";
import {
  hygieneIssueSource,
  hygieneIssueValue,
  hygieneKindLabel,
} from "@/lib/hygiene-labels";
import { padCount } from "@/lib/org-seats";

const GRID = "grid-cols-[44px_1fr_96px_84px_96px]";
const HEADS = ["ID", "Пользователь", "Регистрация", "Записей", "Роль"] as const;
const PAGE_SIZE = 100;

/** A4 · Пользователи и контент-гигиена — 1fr 300px split. */
export function AdminUsers() {
  return (
    <div className="grid min-h-[calc(100vh-56px)] grid-cols-[1fr_300px]">
      <UserRegistry />
      <HygieneRail />
    </div>
  );
}

function UserRegistry() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [more, setMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    listAdminUsers({ limit: PAGE_SIZE })
      .then((rows) => {
        if (cancelled) return;
        setUsers(rows);
        setMore(rows.length === PAGE_SIZE);
        setError(false);
      })
      .catch(() => {
        if (!cancelled) {
          setUsers([]);
          setError(true);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tick]);

  const loadMore = useCallback(async () => {
    if (loadingMore) return;
    setLoadingMore(true);
    try {
      const rows = await listAdminUsers({ limit: PAGE_SIZE, offset: users.length });
      setUsers((prev) => [...prev, ...rows]);
      setMore(rows.length === PAGE_SIZE);
    } catch {
      setMore(false);
      setError(true);
    } finally {
      setLoadingMore(false);
    }
  }, [loadingMore, users.length]);

  if (loading) {
    return (
      <div className="flex flex-col gap-[8px] border-r border-paper px-[16px] py-[14px]">
        <Skeleton className="h-[28px] w-full" />
        <Skeleton className="h-[48px] w-full" />
        <Skeleton className="h-[48px] w-full" />
        <Skeleton className="h-[48px] w-full" />
        <Skeleton className="h-[48px] w-full" />
      </div>
    );
  }

  if (error && users.length === 0) {
    return (
      <div className="border-r border-paper">
        <EmptyState
          numeral="!"
          title="Не удалось загрузить реестр"
          text="Проверьте соединение и попробуйте ещё раз."
          actions={
            <Button
              variant="inverted"
              onClick={() => {
                setLoading(true);
                setTick((n) => n + 1);
              }}
            >
              Повторить
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <div className="flex min-w-0 flex-col border-r border-paper">
      <div className={cn("grid border-b border-paper bg-surface-head", GRID)}>
        {HEADS.map((h, i) => (
          <span
            key={h}
            className={cn(
              "cap px-[8px] py-[6px] text-muted-2",
              i === 0 && "text-center",
              i === 1 && "px-[12px]",
            )}
          >
            {h}
          </span>
        ))}
      </div>

      {users.length === 0 ? (
        <EmptyState
          numeral="00"
          title="Пользователей нет"
          text="В реестре пока нет активных аккаунтов."
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
        users.map((u) => {
          const test = isLikelyTestContent(u.name, u.email);
          const role = adminUserRoleLabel(u, { test });
          return (
            <div
              key={u.id}
              className={cn("grid min-h-[44px] items-center border-b border-rule-inner", GRID)}
            >
              <span
                className="px-[6px] py-[11px] text-center font-mono text-[9.5px] text-muted-2"
                title={u.id}
                data-id={u.id}
              >
                {adminUserShortId(u.id)}
              </span>

              <span className="min-w-0 px-[12px] py-[10px]">
                <span
                  className={cn(
                    "block truncate text-[12px] font-bold leading-[1.25]",
                    test && "text-signal",
                  )}
                >
                  {u.name || "—"}
                </span>
                <span className="cap mt-[2px] block truncate text-muted-2">
                  {u.email || "—"}
                </span>
              </span>

              <span className="px-[8px] py-[10px] font-mono text-[10px] text-text-dim">
                {formatRegistrationMonth(u.created_at)}
              </span>

              <span className="px-[8px] py-[10px] font-mono text-[11px] font-bold">
                {padCount(u.bookings)}
              </span>

              <span className="px-[8px] py-[10px]">
                <Chip
                  as="span"
                  variant={test ? "signal" : "dark-muted"}
                  className="px-[6px] py-[2px] text-[7.5px]"
                >
                  {role}
                </Chip>
              </span>
            </div>
          );
        })
      )}

      {more ? (
        <div className="px-[16px] py-[12px]">
          <Button
            variant="dark-ghost"
            size="sm"
            disabled={loadingMore}
            onClick={loadMore}
            className="min-h-[44px] px-[11px]"
          >
            {loadingMore ? "Загружаем…" : "Показать ещё"}
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function HygieneRail() {
  const [issues, setIssues] = useState<HygieneIssue[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [hiding, setHiding] = useState(false);
  const [done, setDone] = useState<{ hidden: number; skipped: number } | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    listHygieneIssues()
      .then((rows) => {
        if (cancelled) return;
        setIssues(rows);
        setError("");
      })
      .catch(() => {
        if (!cancelled) {
          setIssues([]);
          setError("Не удалось загрузить гигиену контента");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tick]);

  function reload() {
    setLoading(true);
    setConfirming(false);
    setTick((n) => n + 1);
  }

  async function onHideAll() {
    if (hiding) return;
    setHiding(true);
    try {
      const res = await hideAllHygiene();
      setDone(res);
      setError("");
      reload();
    } catch {
      setError("Не удалось скрыть события");
    } finally {
      setHiding(false);
    }
  }

  return (
    <div className="flex flex-col">
      <div className="border-b border-paper px-[16px] py-[11px]">
        <span className="cap font-bold text-signal">
          Гигиена контента · <span className="font-mono">{loading ? "—" : issues.length}</span>
        </span>
      </div>

      {loading ? (
        <div className="flex flex-col gap-[8px] px-[16px] py-[10px]">
          <Skeleton className="h-[52px] w-full" />
          <Skeleton className="h-[52px] w-full" />
          <Skeleton className="h-[52px] w-full" />
        </div>
      ) : error ? (
        <div className="flex flex-col items-start gap-[8px] px-[16px] py-[10px]">
          <p className="text-[11px] text-signal">{error}</p>
          <Button variant="dark-ghost" size="sm" className="min-h-[44px]" onClick={reload}>
            Повторить
          </Button>
        </div>
      ) : issues.length === 0 ? (
        <div className="px-[16px] py-[10px]">
          <p className="text-[11px] font-bold leading-[1.2]">Проблем не найдено</p>
          <p className="cap mt-[3px] text-muted-2">
            {done
              ? `скрыто ${done.hidden}, пропущено ${done.skipped}`
              : "в ленте нет тестовых данных и странных цен"}
          </p>
        </div>
      ) : (
        issues.map((issue) => {
          const source = hygieneIssueSource(issue);
          return (
            <div
              key={`${issue.kind}-${issue.event_id}`}
              className="border-b border-rule-inner px-[16px] py-[10px]"
            >
              <div className="cap mb-[3px] text-muted-2">{hygieneKindLabel(issue.kind)}</div>
              <div className="text-[11px] font-bold leading-[1.2]">
                {hygieneIssueValue(issue)}
              </div>
              {source ? <div className="cap mt-[3px] text-muted-2">{source}</div> : null}
            </div>
          );
        })
      )}

      {!loading && issues.length > 0 ? (
        <div className="mt-auto px-[16px] py-[11px]">
          {confirming ? (
            <div className="flex flex-col gap-[8px]">
              <p className="text-[11px] leading-[1.35] text-text-dim">
                Скрыть <span className="font-mono font-bold">{issues.length}</span> событий из
                ленты? Их можно вернуть в «Модерации».
              </p>
              <div className="flex gap-[8px]">
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={hiding}
                  onClick={onHideAll}
                  className="min-h-[44px] flex-1 text-[10px]"
                >
                  {hiding ? "Скрываем…" : "Подтвердить"}
                </Button>
                <Button
                  variant="dark-ghost"
                  size="sm"
                  disabled={hiding}
                  onClick={() => setConfirming(false)}
                  className="min-h-[44px] px-[10px] text-[10px]"
                >
                  Отмена
                </Button>
              </div>
            </div>
          ) : (
            <Button
              variant="destructive"
              onClick={() => setConfirming(true)}
              className="w-full min-h-[44px] py-[9px] text-[10px]"
            >
              Скрыть всё из ленты
            </Button>
          )}
        </div>
      ) : null}
    </div>
  );
}
