"use client";

import { useRouter } from "next/navigation";

import { useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusChip } from "@/components/ui/StatusChip";
import { adminOrgStatusLabel } from "@/lib/admin-org-status";
import { isLikelyTestContent } from "@/lib/admin-test-heuristic";
import {
  type AdminOrganizer,
  getAdminOrganizer,
  listModerationOrganizers,
  rejectOrganizer,
  revokeOrganizer,
  searchOrganizers,
  setOrganizerAutoVerify,
  verifyOrganizer,
} from "@/lib/api";
import { cn } from "@/lib/cn";
import { statusChipVariant } from "@/lib/status-chip";
import { tileCount } from "@/lib/tile-count";

type Filter = "all" | "pending" | "verified" | "complaints";
type MenuMode = "menu" | "reject" | "revoke";
type PrimaryKind = "verify" | "open" | "delete";

const GRID = "grid-cols-[44px_1fr_110px_90px_90px_170px]";

function parseFilter(raw: string | undefined): Filter {
  if (raw === "pending" || raw === "verified" || raw === "complaints") return raw;
  return "all";
}

/** Mock uses short numeric; we show first 2 hex of UUID uppercased. */
function adminOrgShortId(id: string): string {
  if (!id) return "—";
  const hex = id.replace(/-/g, "").match(/^[0-9a-fA-F]{2}/);
  return hex ? hex[0].toUpperCase() : "—";
}

function mergeOrgs(...lists: AdminOrganizer[][]): AdminOrganizer[] {
  const seen = new Set<string>();
  const out: AdminOrganizer[] = [];
  for (const list of lists) {
    for (const o of list) {
      if (seen.has(o.id)) continue;
      seen.add(o.id);
      out.push(o);
    }
  }
  return out;
}

function is409(err: unknown): boolean {
  return err instanceof Error && err.message.includes("409");
}

function primaryKind(org: AdminOrganizer): PrimaryKind | null {
  if (isLikelyTestContent(org.name, org.website_url)) return "delete";
  if (org.verification_status === "pending") return "verify";
  if (org.verification_status === "verified") return "open";
  return null;
}

function primaryLabel(kind: PrimaryKind): string {
  switch (kind) {
    case "verify":
      return "Проверить";
    case "open":
      return "Открыть";
    case "delete":
      return "Удалить";
  }
}

async function loadAllOrganizers(): Promise<AdminOrganizer[]> {
  const searched = await searchOrganizers("");
  if (searched.length > 0) return searched;
  const [pending, verified, rejected] = await Promise.all([
    listModerationOrganizers("pending"),
    listModerationOrganizers("verified"),
    listModerationOrganizers("rejected"),
  ]);
  return mergeOrgs(pending, verified, rejected);
}

function OrganizersSkeleton() {
  return (
    <div className="flex flex-col gap-[8px] px-[20px] py-[14px]">
      <Skeleton className="h-[32px] w-full" />
      <Skeleton className="h-[36px] w-full" />
      <Skeleton className="h-[48px] w-full" />
      <Skeleton className="h-[48px] w-full" />
      <Skeleton className="h-[48px] w-full" />
    </div>
  );
}

/** A3 · Организаторы и верификация — registry table + absorbed verify/reject. */
export function AdminOrganizers({ initialFilter }: { initialFilter?: string }) {
  const router = useRouter();
  const [filter, setFilter] = useState<Filter>(() => parseFilter(initialFilter));
  const [rows, setRows] = useState<AdminOrganizer[]>([]);
  const [counts, setCounts] = useState({ all: 0, pending: 0, verified: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [actionError, setActionError] = useState("");
  const [busy, setBusy] = useState(false);
  const [tick, setTick] = useState(0);

  const [searchOpen, setSearchOpen] = useState(false);
  const [searchDraft, setSearchDraft] = useState("");
  const [searchQ, setSearchQ] = useState("");

  const [menuId, setMenuId] = useState<string | null>(null);
  const [menuMode, setMenuMode] = useState<MenuMode>("menu");
  const [reason, setReason] = useState("");

  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<AdminOrganizer | null>(null);
  const [detailForId, setDetailForId] = useState<string | null>(null);

  // Debounce search draft → query
  useEffect(() => {
    if (!searchOpen) return;
    const t = window.setTimeout(() => setSearchQ(searchDraft.trim()), 300);
    return () => window.clearTimeout(t);
  }, [searchDraft, searchOpen]);

  // Counts + filtered rows
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        if (filter === "complaints" && !searchQ) {
          const [pending, verified, rejected] = await Promise.all([
            listModerationOrganizers("pending"),
            listModerationOrganizers("verified"),
            listModerationOrganizers("rejected"),
          ]);
          if (cancelled) return;
          setCounts({
            all: mergeOrgs(pending, verified, rejected).length,
            pending: pending.length,
            verified: verified.length,
          });
          setRows([]);
          setError(false);
          setActionError("");
          return;
        }

        if (searchQ) {
          const found = await searchOrganizers(searchQ);
          if (cancelled) return;
          const [pending, verified, rejected] = await Promise.all([
            listModerationOrganizers("pending"),
            listModerationOrganizers("verified"),
            listModerationOrganizers("rejected"),
          ]);
          if (cancelled) return;
          setCounts({
            all: mergeOrgs(pending, verified, rejected).length,
            pending: pending.length,
            verified: verified.length,
          });
          const filtered =
            filter === "pending"
              ? found.filter((o) => o.verification_status === "pending")
              : filter === "verified"
                ? found.filter((o) => o.verification_status === "verified")
                : filter === "complaints"
                  ? []
                  : found;
          setRows(filtered);
          setError(false);
          setActionError("");
          return;
        }

        if (filter === "pending") {
          const [pending, verified, rejected] = await Promise.all([
            listModerationOrganizers("pending"),
            listModerationOrganizers("verified"),
            listModerationOrganizers("rejected"),
          ]);
          if (cancelled) return;
          setCounts({
            all: mergeOrgs(pending, verified, rejected).length,
            pending: pending.length,
            verified: verified.length,
          });
          setRows(pending);
        } else if (filter === "verified") {
          const [pending, verified, rejected] = await Promise.all([
            listModerationOrganizers("pending"),
            listModerationOrganizers("verified"),
            listModerationOrganizers("rejected"),
          ]);
          if (cancelled) return;
          setCounts({
            all: mergeOrgs(pending, verified, rejected).length,
            pending: pending.length,
            verified: verified.length,
          });
          setRows(verified);
        } else {
          const [all, pending, verified] = await Promise.all([
            loadAllOrganizers(),
            listModerationOrganizers("pending"),
            listModerationOrganizers("verified"),
          ]);
          if (cancelled) return;
          setCounts({
            all: all.length,
            pending: pending.length,
            verified: verified.length,
          });
          setRows(all);
        }
        setError(false);
        setActionError("");
      } catch {
        if (!cancelled) {
          setRows([]);
          setError(true);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [filter, searchQ, tick]);

  // Detail for expanded row — key by detailForId so stale detail is ignored (no sync clear).
  useEffect(() => {
    if (!expandedId) return;
    let cancelled = false;
    getAdminOrganizer(expandedId)
      .then((org) => {
        if (!cancelled) {
          setDetail(org);
          setDetailForId(expandedId);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setDetail(null);
          setDetailForId(expandedId);
          setActionError("Не удалось загрузить организатора");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [expandedId]);

  function reload() {
    setLoading(true);
    setTick((n) => n + 1);
  }

  function changeFilter(next: Filter) {
    if (next === filter) return;
    setLoading(true);
    setFilter(next);
    setMenuId(null);
    setExpandedId(null);
    setActionError("");
  }

  function openMenu(id: string, mode: MenuMode = "menu", prefill = "") {
    setMenuId(id);
    setMenuMode(mode);
    setReason(prefill);
    setActionError("");
  }

  function closeMenu() {
    setMenuId(null);
    setMenuMode("menu");
    setReason("");
  }

  async function onVerify(id: string) {
    if (busy) return;
    setBusy(true);
    try {
      await verifyOrganizer(id);
      setActionError("");
      closeMenu();
      reload();
    } catch (err) {
      if (is409(err)) {
        setActionError("");
        reload();
      } else {
        setActionError("Не удалось подтвердить организатора");
      }
    } finally {
      setBusy(false);
    }
  }

  async function onReject(id: string) {
    if (busy || !reason.trim()) {
      if (!reason.trim()) setActionError("Укажите причину отклонения");
      return;
    }
    setBusy(true);
    try {
      await rejectOrganizer(id, reason.trim());
      setActionError("");
      closeMenu();
      reload();
    } catch (err) {
      if (is409(err)) {
        setActionError("");
        closeMenu();
        reload();
      } else {
        setActionError("Не удалось отклонить организатора");
      }
    } finally {
      setBusy(false);
    }
  }

  async function onRevoke(id: string) {
    if (busy || !reason.trim()) {
      if (!reason.trim()) setActionError("Укажите причину отзыва");
      return;
    }
    setBusy(true);
    try {
      await revokeOrganizer(id, reason.trim());
      setActionError("");
      closeMenu();
      setExpandedId(null);
      reload();
    } catch (err) {
      if (is409(err)) {
        setActionError("");
        closeMenu();
        reload();
      } else {
        setActionError("Не удалось отозвать подтверждение");
      }
    } finally {
      setBusy(false);
    }
  }

  async function onToggleAuto(org: AdminOrganizer) {
    if (busy) return;
    setBusy(true);
    try {
      await setOrganizerAutoVerify(org.id, !org.auto_verify);
      setActionError("");
      if (expandedId === org.id) {
        setDetail(await getAdminOrganizer(org.id));
        setDetailForId(org.id);
      }
      reload();
    } catch (err) {
      if (is409(err)) {
        reload();
      } else {
        setActionError("Не удалось изменить авто-подтверждение");
      }
    } finally {
      setBusy(false);
    }
  }

  async function onPrimary(org: AdminOrganizer) {
    const kind = primaryKind(org);
    if (!kind) return;
    if (kind === "verify") {
      await onVerify(org.id);
      return;
    }
    if (kind === "open") {
      // The inline expander only showed the profile; the events being moderated
      // live on the detail page, which is what «Открыть» should reach.
      closeMenu();
      router.push(`/admin/organizers/${org.id}`);
      return;
    }
    // delete / test → open revoke confirm with fixed reason
    openMenu(org.id, "revoke", "Тестовый организатор");
  }

  if (loading) return <OrganizersSkeleton />;

  if (error) {
    return (
      <EmptyState
        numeral="!"
        title="Не удалось загрузить"
        text="Проверьте соединение и попробуйте ещё раз."
        actions={
          <Button
            variant="inverted"
            onClick={() => {
              setLoading(true);
              setTick((t) => t + 1);
            }}
          >
            Повторить
          </Button>
        }
      />
    );
  }

  if (filter === "complaints") {
    return (
      <div>
        <FilterBar
          filter={filter}
          counts={counts}
          searchOpen={searchOpen}
          searchDraft={searchDraft}
          onFilter={changeFilter}
          onSearchOpen={() => setSearchOpen(true)}
          onSearchChange={setSearchDraft}
          onSearchClose={() => {
            setSearchOpen(false);
            setSearchDraft("");
            setSearchQ("");
          }}
        />
        <EmptyState
          numeral="00"
          title="Фильтр по жалобам появится позже"
          text="Пока нет данных о жалобах на организаторов."
          actions={
            <Button variant="inverted" onClick={() => changeFilter("all")}>
              Все организаторы
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <div>
      <FilterBar
        filter={filter}
        counts={counts}
        searchOpen={searchOpen}
        searchDraft={searchDraft}
        onFilter={changeFilter}
        onSearchOpen={() => setSearchOpen(true)}
        onSearchChange={setSearchDraft}
        onSearchClose={() => {
          setSearchOpen(false);
          setSearchDraft("");
          setSearchQ("");
        }}
      />

      {actionError ? (
        <p className="border-b border-rule-inner px-[20px] py-[8px] text-[11px] text-signal">
          {actionError}
        </p>
      ) : null}

      <div
        className={cn(
          "grid border-b border-paper bg-surface-head",
          GRID,
        )}
      >
        {(["ID", "Организатор", "Статус", "Событий", "Жалоб", "Действия"] as const).map(
          (h, i) => (
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
          ),
        )}
      </div>

      {rows.length === 0 ? (
        <EmptyState
          numeral="00"
          title="Список пуст"
          text={
            searchQ
              ? "По этому запросу ничего не найдено."
              : "Нет организаторов в выбранном фильтре."
          }
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
        rows.map((org) => {
          const test = isLikelyTestContent(org.name, org.website_url);
          const label = adminOrgStatusLabel(org.verification_status, { test });
          const tone = statusChipVariant(label);
          const kind = primaryKind(org);
          const menuOpen = menuId === org.id;
          const expanded = expandedId === org.id;
          const shownDetail =
            expanded && detailForId === org.id ? detail : null;
          const caption = org.website_url?.trim() || "—";

          return (
            <div key={org.id} className="border-b border-rule-inner">
              <div className={cn("grid items-center", GRID)}>
                <span
                  className="px-[6px] py-[9px] text-center font-mono text-[9.5px] text-muted-2"
                  title={org.id}
                  data-id={org.id}
                >
                  {adminOrgShortId(org.id)}
                </span>

                <span className="min-w-0 px-[12px] py-[8px]">
                  <span
                    className={cn(
                      "block text-[12px] font-bold leading-[1.25]",
                      test && "text-signal",
                    )}
                  >
                    {org.name}
                  </span>
                  <span className="cap mt-[2px] block text-muted-2">{caption}</span>
                </span>

                <span className="px-[8px] py-[8px]">
                  <StatusChip
                    status={label}
                    className={cn(
                      tone === "active" && "border-paper bg-paper text-ink",
                      "text-[7.5px] px-[6px] py-[2px]",
                    )}
                  />
                </span>

                <span className="px-[8px] py-[8px] font-mono text-[11px] font-bold">
                  {tileCount(org.events_count)}
                </span>
                <span
                  className={cn(
                    "px-[8px] py-[8px] font-mono text-[11px] font-bold",
                    (org.complaints_count ?? 0) > 0 && "text-signal",
                  )}
                >
                  {tileCount(org.complaints_count)}
                </span>

                <span className="flex gap-[5px] px-[8px] py-[7px]">
                  {kind ? (
                    <Button
                      variant="inverted"
                      size="sm"
                      disabled={busy}
                      onClick={() => onPrimary(org)}
                      className="min-h-[44px] flex-1 px-[4px] py-[5px] text-[8px] tracking-[0.05em]"
                    >
                      {primaryLabel(kind)}
                    </Button>
                  ) : (
                    <span className="flex-1" />
                  )}
                  <Button
                    variant="dark-ghost"
                    size="sm"
                    disabled={busy}
                    onClick={() =>
                      menuOpen ? closeMenu() : openMenu(org.id, "menu")
                    }
                    className="min-h-[44px] shrink-0 px-[8px] py-[5px] text-[8px]"
                    aria-label="Ещё"
                  >
                    ···
                  </Button>
                </span>
              </div>

              {menuOpen ? (
                <div className="border-t border-rule-inner bg-surface-head px-[14px] py-[12px]">
                  {menuMode === "menu" ? (
                    <div className="flex flex-wrap gap-[8px]">
                      {org.verification_status === "pending" ? (
                        <Button
                          variant="destructive"
                          size="sm"
                          disabled={busy}
                          onClick={() => openMenu(org.id, "reject")}
                        >
                          Отклонить
                        </Button>
                      ) : null}
                      {org.verification_status === "verified" || test ? (
                        <Button
                          variant="destructive"
                          size="sm"
                          disabled={busy}
                          onClick={() =>
                            openMenu(
                              org.id,
                              "revoke",
                              test ? "Тестовый организатор" : "",
                            )
                          }
                        >
                          Отозвать
                        </Button>
                      ) : null}
                      <Button
                        variant="dark-ghost"
                        size="sm"
                        disabled={busy}
                        onClick={() => onToggleAuto(org)}
                      >
                        Авто-проверка: {org.auto_verify ? "вкл" : "выкл"}
                      </Button>
                      <Button
                        variant="dark-ghost"
                        size="sm"
                        onClick={closeMenu}
                      >
                        Закрыть
                      </Button>
                    </div>
                  ) : (
                    <div className="flex max-w-[420px] flex-col gap-[8px]">
                      <label className="cap text-muted-2">
                        {menuMode === "reject"
                          ? "Причина отклонения"
                          : "Причина отзыва"}
                      </label>
                      <input
                        value={reason}
                        onChange={(e) => setReason(e.target.value)}
                        className="swiss-focus w-full border border-muted-2 bg-transparent px-[10px] py-[8px] text-[12px] text-on-surface outline-none"
                        autoFocus
                      />
                      <div className="flex gap-[8px]">
                        <Button
                          variant="destructive"
                          size="sm"
                          disabled={busy || !reason.trim()}
                          onClick={() =>
                            menuMode === "reject"
                              ? onReject(org.id)
                              : onRevoke(org.id)
                          }
                        >
                          {menuMode === "reject" ? "Отклонить" : "Отозвать"}
                        </Button>
                        <Button
                          variant="dark-ghost"
                          size="sm"
                          onClick={() => openMenu(org.id, "menu")}
                        >
                          Назад
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              ) : null}

              {expanded ? (
                <div className="border-t border-rule-inner px-[14px] py-[12px]">
                  {shownDetail ? (
                    <div className="space-y-[8px]">
                      {shownDetail.description ? (
                        <p className="text-[11.5px] leading-[1.45] text-text-dim">
                          {shownDetail.description}
                        </p>
                      ) : null}
                      <p className="cap text-muted-2">
                        Авто-проверка: {shownDetail.auto_verify ? "вкл" : "выкл"}
                      </p>
                      {shownDetail.history && shownDetail.history.length > 0 ? (
                        <ul className="space-y-[4px]">
                          <li className="cap text-muted-2">История</li>
                          {shownDetail.history.map((h, i) => (
                            <li
                              key={`${h.created_at}-${i}`}
                              className="font-mono text-[10px] text-text-dim"
                            >
                              {h.from_status} → {h.to_status}
                              {h.reason ? ` · ${h.reason}` : ""}
                              {" · "}
                              {new Date(h.created_at).toLocaleString("ru-RU")}
                            </li>
                          ))}
                        </ul>
                      ) : (
                        <p className="cap text-muted-2">Истории нет</p>
                      )}
                    </div>
                  ) : (
                    <Skeleton className="h-[48px] w-full" />
                  )}
                </div>
              ) : null}
            </div>
          );
        })
      )}
    </div>
  );
}

function FilterBar({
  filter,
  counts,
  searchOpen,
  searchDraft,
  onFilter,
  onSearchOpen,
  onSearchChange,
  onSearchClose,
}: {
  filter: Filter;
  counts: { all: number; pending: number; verified: number };
  searchOpen: boolean;
  searchDraft: string;
  onFilter: (f: Filter) => void;
  onSearchOpen: () => void;
  onSearchChange: (v: string) => void;
  onSearchClose: () => void;
}) {
  return (
    <div className="flex items-center justify-between gap-[12px] border-b border-paper px-[20px] py-[11px]">
      <div className="flex flex-wrap gap-[6px]">
        <Chip
          variant={filter === "all" ? "dark-active" : "dark-muted"}
          onClick={() => onFilter("all")}
        >
          Все · {counts.all}
        </Chip>
        <Chip
          variant={filter === "pending" ? "signal" : "dark-muted"}
          onClick={() => onFilter("pending")}
        >
          На проверке · {counts.pending}
        </Chip>
        <Chip
          variant={filter === "verified" ? "dark-active" : "dark-muted"}
          onClick={() => onFilter("verified")}
        >
          Верифицированы · {counts.verified}
        </Chip>
        <Chip
          variant={filter === "complaints" ? "dark-active" : "dark-muted"}
          onClick={() => onFilter("complaints")}
        >
          С жалобами · 0
        </Chip>
      </div>

      {searchOpen ? (
        <input
          autoFocus
          value={searchDraft}
          onChange={(e) => onSearchChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Escape") onSearchClose();
          }}
          placeholder="Поиск"
          className="swiss-focus w-[180px] shrink-0 border border-muted-2 bg-transparent px-[8px] py-[4px] text-[11px] text-on-surface outline-none placeholder:text-muted-2"
          aria-label="Поиск организаторов"
        />
      ) : (
        <button
          type="button"
          onClick={onSearchOpen}
          className="cap swiss-focus shrink-0 text-muted-2"
        >
          Поиск ⌕
        </button>
      )}
    </div>
  );
}
