import Link from "next/link";
import { EmptyState } from "@/components/ui/EmptyState";

export default function AdminUsersStub() {
  return (
    <EmptyState
      numeral="—"
      title="Пользователи"
      text="Реестр и гигиена контента появятся в следующей фазе."
      actions={
        <Link
          href="/admin"
          className="swiss-focus bg-paper px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink"
        >
          К ОБЗОРУ
        </Link>
      }
    />
  );
}
