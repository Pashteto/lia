"use client";

import { useEffect, useState } from "react";

import { AdminTrustedPlatforms } from "@/components/AdminTrustedPlatforms";
import { Chip } from "@/components/ui/Chip";
import { Skeleton } from "@/components/ui/Skeleton";
import { SquareCheck } from "@/components/ui/SquareCheck";
import { getAdminSettings, setAdminSetting } from "@/lib/api";

const AUTO_VERIFY_ALL = "organizers.auto_verify_all";

/** Every switch the admin surface exposes, in the order they appear. */
const FLAGS = [
  {
    key: AUTO_VERIFY_ALL,
    label: "Авто-подтверждение всех организаторов",
    text:
      "Каждая отправленная заявка организатора подтверждается автоматически, минуя очередь модерации. Включайте, когда за очередью никто не следит.",
  },
] as const;

export default function AdminSettingsPage() {
  const [settings, setSettings] = useState<Record<string, boolean> | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAdminSettings()
      .then(setSettings)
      .catch(() => setError("Не удалось загрузить настройки"));
  }, []);

  const toggle = async (key: string) => {
    if (!settings || busy) return;
    const next = !settings[key];
    setBusy(key);
    setError(null);
    try {
      await setAdminSetting(key, next);
      setSettings((s) => ({ ...(s ?? {}), [key]: next }));
    } catch {
      setError("Не удалось сохранить настройку");
    } finally {
      setBusy(null);
    }
  };

  return (
    <div className="flex flex-col">
      <div className="flex items-baseline justify-between gap-[12px] border-b border-paper px-[20px] py-[14px] max-sm:px-[14px]">
        <h1 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">Настройки</h1>
        <span className="cap text-muted-2">Действуют сразу</span>
      </div>

      {error ? (
        <p className="border-b border-rule-inner px-[20px] py-[8px] text-[11px] text-signal max-sm:px-[14px]">
          {error}
        </p>
      ) : null}

      {settings === null ? (
        <div className="flex flex-col gap-[8px] px-[20px] py-[16px] max-sm:px-[14px]">
          <Skeleton className="h-[72px] w-full" />
        </div>
      ) : (
        FLAGS.map((flag) => {
          const on = !!settings[flag.key];
          return (
            <label
              key={flag.key}
              htmlFor={flag.key}
              className="flex min-h-[44px] cursor-pointer items-start gap-[14px] border-b border-rule-inner px-[20px] py-[14px] max-sm:px-[14px]"
            >
              <SquareCheck
                id={flag.key}
                checked={on}
                disabled={busy === flag.key}
                onChange={() => toggle(flag.key)}
                aria-labelledby={`${flag.key}-label`}
              />
              <span className="min-w-0 flex-1">
                <span className="flex items-baseline gap-[8px]">
                  <span id={`${flag.key}-label`} className="text-[12px] font-bold">
                    {flag.label}
                  </span>
                  <Chip as="span" variant={on ? "signal" : "dark-muted"} className="px-[6px] py-[2px] text-[7.5px]">
                    {on ? "Включено" : "Выключено"}
                  </Chip>
                </span>
                <span className="mt-[4px] block max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim">
                  {flag.text}
                </span>
              </span>
            </label>
          );
        })
      )}

      <AdminTrustedPlatforms />
    </div>
  );
}
