"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Field";
import { useAuth } from "@/lib/auth-context";
import { safeNextPath } from "@/lib/safe-next";

function currentNext(): string | null {
  if (typeof window === "undefined") return null;
  const next = new URLSearchParams(window.location.search).get("next");
  return safeNextPath(next, window.location.origin);
}

export function AuthForm({
  mode,
  onSuccess,
}: {
  mode: "login" | "register";
  onSuccess?: () => void;
}) {
  const { register, loginPassword } = useAuth();
  const router = useRouter();
  const next = currentNext();
  const isRegister = mode === "register";

  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [registeredEmail, setRegisteredEmail] = useState<string | null>(null);

  const mismatch = isRegister && confirm.length > 0 && confirm !== password;
  const canSubmit =
    email.trim().length > 0 &&
    password.length >= (isRegister ? 8 : 1) &&
    (!isRegister || confirm === password) &&
    !busy;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setBusy(true);
    setError(null);
    try {
      if (isRegister) {
        const addr = email.trim();
        await register(addr, name.trim(), password);
        setRegisteredEmail(addr);
      } else {
        await loginPassword(email.trim(), password);
        onSuccess?.();
        if (!onSuccess) router.push(next ?? "/");
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Не удалось войти. Попробуйте ещё раз.",
      );
    } finally {
      setBusy(false);
    }
  };

  if (registeredEmail) {
    return (
      <div className="flex flex-col gap-[10px]">
        <h2 className="text-[22px] font-black tracking-[-0.02em]">Проверьте почту</h2>
        <p className="max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim">
          Мы отправили 6-значный код на {registeredEmail}. Он действует 24 часа.
        </p>
        <Link
          href={next ? `/auth/verify?next=${encodeURIComponent(next)}` : "/auth/verify"}
          className="swiss-focus mt-[6px] self-start bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black"
        >
          Ввести код
        </Link>
      </div>
    );
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-[11px]">
      {isRegister && (
        <Input label="Имя" value={name} onChange={(e) => setName(e.target.value)} placeholder="Как вас представить участникам" autoComplete="name" />
      )}
      <Input label="Почта" type="email" required autoFocus value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" autoComplete="email" />
      <Input label={isRegister ? "Пароль (минимум 8 символов)" : "Пароль"} type="password" required minLength={isRegister ? 8 : undefined} value={password} onChange={(e) => setPassword(e.target.value)} placeholder="••••••••" autoComplete={isRegister ? "new-password" : "current-password"} />
      {isRegister && (
        <Input label="Пароль ещё раз" type="password" required value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder="••••••••" autoComplete="new-password" error={mismatch ? "Пароли не совпадают" : undefined} />
      )}
      {error && <p className="text-[11px] text-signal">{error}</p>}
      <Button type="submit" disabled={!canSubmit} className="mt-[6px]">
        {busy ? (isRegister ? "Создаём…" : "Вход…") : isRegister ? "Создать аккаунт" : "Войти"}
      </Button>
    </form>
  );
}
