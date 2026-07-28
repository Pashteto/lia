import Link from "next/link";
import { AuthForm } from "@/components/AuthForm";

export const metadata = { title: "Регистрация — PRESENCE" };

export default function SignupPage() {
  return (
    <main className="grid min-h-screen grid-cols-2 max-md:grid-cols-1">
      <div
        data-surface="ink"
        className="flex flex-col justify-between bg-surface p-[20px] text-on-surface max-md:min-h-[220px]"
      >
        <span className="text-[13px] font-black tracking-[-0.01em]">PRESENCE</span>
        <div className="flex flex-col gap-[10px]">
          <h1 className="max-w-[16ch] text-[34px] font-black leading-[0.94] tracking-[-0.03em] max-md:text-[22px]">
            Участвуйте, а не только смотрите
          </h1>
          <p className="cap">Аккаунт за минуту</p>
        </div>
      </div>
      <div className="flex flex-col justify-center gap-[14px] p-[20px] md:px-[48px]">
        <div>
          <p className="cap">Регистрация</p>
          <h2 className="mt-[6px] text-[22px] font-black tracking-[-0.02em]">
            Создайте аккаунт
          </h2>
        </div>
        <AuthForm mode="register" />
        <p className="mt-auto pt-[14px] text-[11.5px] text-text-dim">
          Уже есть аккаунт?{" "}
          <Link href="/login" className="swiss-focus underline underline-offset-2">
            Войти
          </Link>
        </p>
      </div>
    </main>
  );
}
