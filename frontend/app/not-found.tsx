import Link from "next/link";

/** U8-3: 404, inverted to ink. Mono numeral, one sentence, one action. */
export default function NotFound() {
  return (
    <div
      data-surface="ink"
      className="flex min-h-screen flex-col items-start justify-center gap-[10px] bg-surface px-[20px] text-on-surface"
    >
      <span className="font-mono text-[44px] font-bold leading-none tracking-[-0.04em]">
        404
      </span>
      <h1 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">
        Страница не найдена
      </h1>
      <p className="max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim-dark-2">
        Проверьте адрес: возможно, ссылка устарела или страницу убрали.
      </p>
      <Link
        href="/"
        className="swiss-focus mt-[6px] bg-paper px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink hover:opacity-90"
      >
        Вернуться к ленте
      </Link>
    </div>
  );
}
