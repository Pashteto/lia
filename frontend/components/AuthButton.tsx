"use client";

import Link from "next/link";
import { useState } from "react";

import { AuthForm } from "@/components/AuthForm";
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
      <Button variant="ghost" disabled>
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
        <Button variant="ghost" onClick={logout}>
          Выйти
        </Button>
      </div>
    );
  }

  return (
    <>
      <Button variant="ghost" onClick={() => setOpen(true)}>
        Войти
      </Button>
      {open && <LoginModal onClose={() => setOpen(false)} />}
    </>
  );
}

export function LoginModal({ onClose }: { onClose: () => void }) {
  const [mode, setMode] = useState<"login" | "register">("login");
  const isRegister = mode === "register";
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm border border-ink bg-paper p-[20px]"
        onClick={(e) => e.stopPropagation()}
      >
        <p className="cap">{isRegister ? "Регистрация" : "Вход"}</p>
        <h2 className="mb-[14px] mt-[6px] text-[22px] font-black tracking-[-0.02em]">
          {isRegister ? "Создайте аккаунт" : "С возвращением"}
        </h2>
        <AuthForm mode={mode} onSuccess={onClose} />
        <div className="mt-[14px] flex items-center justify-between">
          <button
            type="button"
            className="swiss-focus text-[11.5px] underline underline-offset-2"
            onClick={() => setMode(isRegister ? "login" : "register")}
          >
            {isRegister ? "Уже есть аккаунт? Войти" : "Нет аккаунта? Регистрация"}
          </button>
          <button type="button" className="cap swiss-focus" onClick={onClose}>
            Закрыть
          </button>
        </div>
      </div>
    </div>
  );
}
