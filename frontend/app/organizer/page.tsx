"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { fetchMyEvents, getMyOrganizer } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { LiaEvent } from "@/lib/types";

const VERIFY_LABEL: Record<string, string> = {
  draft: "Профиль не отправлен",
  pending: "На проверке",
  verified: "Подтверждённый организатор",
  rejected: "Отклонён — отредактируйте профиль",
};

const VERIFY_CLASS: Record<string, string> = {
  draft: "text-label-secondary",
  pending: "text-amber-600",
  verified: "text-green-600",
  rejected: "text-red-500",
};

export default function OrganizerHubPage() {
  const { isAuthed, ready } = useAuth();

  const { data: organizer } = useQuery({
    queryKey: ["my-organizer"],
    queryFn: getMyOrganizer,
    enabled: ready && isAuthed,
  });
  const { data: events = [] } = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: ready && isAuthed,
  });

  const drafts = events.filter((e: LiaEvent) => e.status === "draft").length;
  const published = events.filter((e: LiaEvent) => e.status === "published").length;
  const status = organizer?.verification_status ?? "draft";

  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[28px] font-bold tracking-[-0.022em]">Организаторам</h1>
          <p className="mt-1 text-[14px] text-label-secondary">
            Создавайте и ведите свои события
          </p>
        </div>
        <Link href="/events/new">
          <Button variant="tinted">+ Создать событие</Button>
        </Link>
      </div>

      {isAuthed && (
        <Link
          href="/me/organizer"
          className="mt-5 flex items-center gap-2 rounded-card bg-bg p-3.5 shadow-card-subtle"
        >
          <span className={`text-[14px] font-medium ${VERIFY_CLASS[status] ?? ""}`}>
            {VERIFY_LABEL[status] ?? "Профиль организатора"}
          </span>
          <span className="ml-auto text-[13px] text-label-secondary">
            Профиль организатора →
          </span>
        </Link>
      )}

      <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <HubCard
          href="/events/mine"
          title="Мои события"
          subtitle="Черновики, на модерации, опубликованные"
        >
          {isAuthed && (
            <div className="mt-2 flex gap-2 text-[11px]">
              <span className="rounded-control bg-amber-100 px-2 py-0.5 text-amber-800">
                {drafts} черновиков
              </span>
              <span className="rounded-control bg-green-100 px-2 py-0.5 text-green-800">
                {published} опубликовано
              </span>
            </div>
          )}
        </HubCard>

        <HubCard
          href="/organizer/applications"
          title="Заявки участников"
          subtitle="Подтвердить или отклонить запись"
        />

        <HubCard
          href="/me/organizer"
          title="Профиль организатора"
          subtitle="Название, описание, логотип, верификация"
        />

        <div className="rounded-card bg-bg p-4 opacity-50 shadow-card-subtle">
          <h3 className="text-[16px] font-semibold">Подписчики</h3>
          <p className="mt-0.5 text-[13px] text-label-secondary">
            Кто следит за вашими событиями (позже)
          </p>
        </div>
      </div>
    </main>
  );
}

function HubCard({
  href,
  title,
  subtitle,
  children,
}: {
  href: string;
  title: string;
  subtitle: string;
  children?: React.ReactNode;
}) {
  return (
    <Link href={href} className="rounded-card bg-bg p-4 shadow-card-subtle transition hover:shadow-card">
      <h3 className="text-[16px] font-semibold">{title}</h3>
      <p className="mt-0.5 text-[13px] text-label-secondary">{subtitle}</p>
      {children}
    </Link>
  );
}
