"use client";

import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { useAuth } from "@/lib/auth-context";

// Nav auth control: "Войти" when signed out (opens a demo-login modal), or the
// signed-in email + "Выйти" when authed. Demo-login takes just an email — no
// password (see lib/auth.ts).
export function AuthButton() {
  const { email, isAuthed, ready, logout, role } = useAuth();
  const [open, setOpen] = useState(false);

  // Before the stored session is read, render a stable placeholder so the
  // button doesn't flicker between states on hydration.
  if (!ready) {
    return (
      <Button variant="plain" disabled>
        Войти
      </Button>
    );
  }

  if (isAuthed) {
    return (
      <div className="flex items-center gap-2">
        <Link
          href="/me/calendar"
          className="text-[14px] font-medium text-accent"
        >
          Календарь
        </Link>
        <Link
          href="/organizer"
          className="text-[14px] font-medium text-accent"
        >
          Организаторам
        </Link>
        <Link
          href="/me/invitations"
          className="text-[14px] font-medium text-accent"
        >
          Приглашения
        </Link>
        {role === "admin" && (
          <Link
            href="/admin"
            className="text-[14px] font-medium text-accent"
          >
            Админ
          </Link>
        )}
        <span
          className="hidden max-w-[12rem] truncate text-[14px] text-label-secondary sm:block"
          title={email ?? undefined}
        >
          {email}
        </span>
        <Button variant="plain" onClick={logout}>
          Выйти
        </Button>
      </div>
    );
  }

  return (
    <>
      <Button variant="plain" onClick={() => setOpen(true)}>
        Войти
      </Button>
      {open && <LoginModal onClose={() => setOpen(false)} />}
    </>
  );
}

export function LoginModal({ onClose }: { onClose: () => void }) {
  const { register, loginPassword } = useAuth();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [registeredEmail, setRegisteredEmail] = useState<string | null>(null);

  const isRegister = mode === "register";
  const canSubmit =
    email.trim().length > 0 &&
    password.length >= (isRegister ? 8 : 1) &&
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
        setRegisteredEmail(addr); // show the confirmation instead of closing
      } else {
        await loginPassword(email.trim(), password);
        onClose();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось войти. Попробуйте ещё раз.");
    } finally {
      setBusy(false);
    }
  };

  const inputClass =
    "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

  if (registeredEmail) {
    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
        onClick={onClose}
      >
        <div className="w-full max-w-sm rounded-card bg-bg p-5" onClick={(e) => e.stopPropagation()}>
          <h2 className="mb-1 text-[17px] font-semibold">Проверьте почту</h2>
          <p className="mb-4 text-[13px] text-label-secondary">
            Мы отправили 6-значный код на {registeredEmail}. Он действует 24 часа.
          </p>
          <div className="flex gap-2">
            <Link href="/auth/verify" className="rounded-capsule bg-accent px-4 py-2 text-white">
              Ввести код
            </Link>
            <button onClick={onClose} className="rounded-capsule bg-fill px-4 py-2 text-label">
              Позже
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <form
        className="w-full max-w-sm rounded-card bg-bg p-5"
        onClick={(e) => e.stopPropagation()}
        onSubmit={submit}
      >
        <h2 className="mb-1 text-[17px] font-semibold">
          {isRegister ? "Регистрация" : "Вход"}
        </h2>
        <p className="mb-4 text-[13px] text-label-secondary">
          {isRegister
            ? "Создайте аккаунт с email и паролем, чтобы публиковать события."
            : "Войдите с email и паролем."}
        </p>
        <label className="mb-3 block">
          <span className="mb-1.5 block text-[13px] text-label-secondary">Email</span>
          <input
            type="email"
            required
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            className={inputClass}
          />
        </label>
        {isRegister && (
          <label className="mb-3 block">
            <span className="mb-1.5 block text-[13px] text-label-secondary">
              Имя (необязательно)
            </span>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Ваше имя"
              className={inputClass}
            />
          </label>
        )}
        <label className="mb-4 block">
          <span className="mb-1.5 block text-[13px] text-label-secondary">
            Пароль{isRegister ? " (минимум 8 символов)" : ""}
          </span>
          <input
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            minLength={isRegister ? 8 : undefined}
            className={inputClass}
          />
        </label>
        {error && <p className="mb-3 text-[14px] text-red-500">{error}</p>}
        <div className="flex items-center justify-between gap-2">
          <button
            type="button"
            className="text-[13px] text-accent"
            onClick={() => {
              setMode(isRegister ? "login" : "register");
              setError(null);
            }}
          >
            {isRegister ? "Уже есть аккаунт? Войти" : "Нет аккаунта? Регистрация"}
          </button>
          <div className="flex gap-2">
            <button
              type="button"
              className="px-3 py-2 text-[15px] text-label"
              onClick={onClose}
            >
              Отмена
            </button>
            <Button type="submit" disabled={!canSubmit}>
              {busy
                ? isRegister
                  ? "Создаём…"
                  : "Вход…"
                : isRegister
                  ? "Создать"
                  : "Войти"}
            </Button>
          </div>
        </div>
      </form>
    </div>
  );
}
