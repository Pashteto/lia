"use client";

import { useEffect, useState } from "react";
import { getAdminSettings, setAdminSetting } from "@/lib/api";

const AUTO_VERIFY_ALL = "organizers.auto_verify_all";

export default function AdminSettingsPage() {
  const [settings, setSettings] = useState<Record<string, boolean>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAdminSettings().then(setSettings).catch((e) => setError(String(e)));
  }, []);

  const toggle = async (key: string) => {
    setBusy(true);
    setError(null);
    const next = !settings[key];
    try {
      await setAdminSetting(key, next);
      setSettings((s) => ({ ...s, [key]: next }));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-col gap-[12px] px-[20px] py-[26px] max-sm:px-[14px]">
      <h1 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">Настройки</h1>
      {error ? (
        <p className="border-b border-rule-inner py-[8px] text-[11px] text-signal">{error}</p>
      ) : null}
      <div className="space-y-[12px] border border-rule-inner p-[14px]">
        <label className="flex items-start gap-[12px]">
          <input
            type="checkbox"
            checked={!!settings[AUTO_VERIFY_ALL]}
            disabled={busy}
            onChange={() => toggle(AUTO_VERIFY_ALL)}
            className="mt-[4px] swiss-focus"
          />
          <span>
            <span className="text-[12px] font-bold">Авто-подтверждение всех организаторов</span>
            <span className="mt-[4px] block text-[11.5px] text-text-dim">
              Когда включено, каждая отправленная заявка организатора подтверждается автоматически,
              минуя очередь модерации. Включайте, если нет доступных модераторов.
            </span>
          </span>
        </label>
      </div>
    </div>
  );
}
