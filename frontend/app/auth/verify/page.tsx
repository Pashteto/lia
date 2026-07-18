"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  requestVerification,
  verifyEmail,
  VERIFICATION_EXPIRED,
  VERIFICATION_ATTEMPTS_EXCEEDED,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

const inputClass =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

export default function VerifyPage() {
  const { email, refresh } = useAuth();
  const router = useRouter();
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);

  const addr = email ?? "";

  async function onResend() {
    setError("");
    try {
      await requestVerification(addr);
      setSent(true);
    } catch {
      setError("Не удалось отправить код. Попробуйте позже.");
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (code.length !== 6) return;
    setBusy(true);
    setError("");
    try {
      await verifyEmail(addr, code);
      await refresh(); // re-fetch /auth/me so emailVerified updates before navigating
      // Resume the flow that triggered verification (e.g. an /invite/[token]
      // page) via ?next=; fall back to the feed. Only app-internal paths are
      // honored, to avoid an open-redirect.
      const next = new URLSearchParams(window.location.search).get("next");
      const dest = next && next.startsWith("/") && !next.startsWith("//") ? next : "/";
      router.push(dest);
    } catch (err) {
      const m = err instanceof Error ? err.message : "";
      if (m === VERIFICATION_EXPIRED) setError("Код истёк. Запросите новый.");
      else if (m === VERIFICATION_ATTEMPTS_EXCEEDED)
        setError("Код заблокирован после 5 попыток. Запросите новый.");
      else setError("Неверный код.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-md px-4 py-10">
      <h1 className="mb-2 text-[28px] font-bold tracking-[-0.022em]">Подтвердите почту</h1>
      <p className="mb-4 text-[15px] text-label-secondary">
        Мы отправили 6-значный код на {addr || "вашу почту"}.
      </p>
      <form onSubmit={onSubmit}>
        <input
          className={inputClass + " mb-3 text-center text-[22px] tracking-[10px]"}
          inputMode="numeric"
          maxLength={6}
          placeholder="000000"
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
        />
        {error && <p className="mb-3 text-[14px] text-red-500">{error}</p>}
        {sent && <p className="mb-3 text-[14px] text-green-600">Код отправлен.</p>}
        <button
          type="submit"
          disabled={busy || code.length !== 6}
          className="w-full rounded-capsule bg-accent px-4 py-2.5 text-white disabled:opacity-50"
        >
          Подтвердить
        </button>
      </form>
      <button onClick={onResend} className="mt-4 text-[15px] text-accent">
        Отправить код ещё раз
      </button>
    </main>
  );
}
