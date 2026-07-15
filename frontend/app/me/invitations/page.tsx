"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  EMAIL_NOT_VERIFIED,
  acceptMyInvitation,
  declineMyInvitation,
  fetchMyInvitations,
  type MyInvitation,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

function InvitationRow({ invitation }: { invitation: MyInvitation }) {
  const router = useRouter();
  const queryClient = useQueryClient();

  const accept = useMutation({
    mutationFn: () => acceptMyInvitation(invitation.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["my-invitations"] }),
    onError: (e) => {
      if (e instanceof Error && e.message === EMAIL_NOT_VERIFIED) {
        router.push("/auth/verify");
      }
    },
  });
  const decline = useMutation({
    mutationFn: () => declineMyInvitation(invitation.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["my-invitations"] }),
  });

  return (
    <li className="flex items-center justify-between gap-3 rounded-card bg-bg-secondary p-4 shadow-card-subtle">
      <Link
        href={`/events/${invitation.event_id}`}
        className="text-[17px] font-semibold text-accent hover:underline"
      >
        Открыть событие
      </Link>
      <div className="flex shrink-0 gap-2">
        <button
          onClick={() => accept.mutate()}
          disabled={accept.isPending || decline.isPending}
          className="rounded-capsule bg-accent px-3 py-1.5 text-[14px] font-medium text-white disabled:opacity-50"
        >
          {accept.isPending ? "…" : "Принять"}
        </button>
        <button
          onClick={() => decline.mutate()}
          disabled={accept.isPending || decline.isPending}
          className="rounded-capsule bg-fill px-3 py-1.5 text-[14px] font-medium text-label disabled:opacity-50"
        >
          {decline.isPending ? "…" : "Отклонить"}
        </button>
      </div>
    </li>
  );
}

export default function MyInvitationsPage() {
  const { isAuthed, ready } = useAuth();

  const { data: invitations = [], isLoading, isError } = useQuery({
    queryKey: ["my-invitations"],
    queryFn: fetchMyInvitations,
    enabled: ready && isAuthed,
  });

  if (!ready) {
    return <div className="min-h-screen bg-bg-grouped" />;
  }

  if (!isAuthed) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16">
        <Link href="/" className="inline-flex items-center text-[17px] text-accent">
          ‹ События
        </Link>
        <div className="mt-8 text-center">
          <h1 className="text-[28px] font-bold tracking-[-0.022em]">Приглашения</h1>
          <p className="mt-3 text-label-secondary">
            Войдите, чтобы увидеть приглашения.
          </p>
        </div>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-8 max-sm:pb-28">
      <Link href="/" className="mb-4 inline-flex items-center text-[17px] text-accent">
        ‹ События
      </Link>
      <h1 className="text-[28px] font-bold tracking-[-0.022em]">Приглашения</h1>

      {isLoading ? (
        <p className="mt-8 text-label-secondary">Загрузка…</p>
      ) : isError ? (
        <p className="mt-8 text-label-secondary">Не удалось загрузить данные.</p>
      ) : invitations.length === 0 ? (
        <p className="mt-8 text-label-secondary">Нет новых приглашений.</p>
      ) : (
        <ul className="mt-6 flex flex-col gap-3">
          {invitations.map((invitation) => (
            <InvitationRow key={invitation.id} invitation={invitation} />
          ))}
        </ul>
      )}
    </main>
  );
}
