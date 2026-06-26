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
    <div className="space-y-6">
      <h1 className="text-2xl font-bold tracking-[-0.022em]">Настройки</h1>
      {error && <p className="text-sm text-red-600">{error}</p>}
      <div className="glass space-y-3 rounded-card p-6">
        <label className="flex items-start gap-3">
          <input
            type="checkbox"
            checked={!!settings[AUTO_VERIFY_ALL]}
            disabled={busy}
            onChange={() => toggle(AUTO_VERIFY_ALL)}
            className="mt-1"
          />
          <span>
            <span className="font-medium">Авто-подтверждение всех организаторов</span>
            <span className="block text-sm text-label-secondary">
              Когда включено, каждая отправленная заявка организатора подтверждается автоматически,
              минуя очередь модерации. Включайте, если нет доступных модераторов.
            </span>
          </span>
        </label>
      </div>
    </div>
  );
}
