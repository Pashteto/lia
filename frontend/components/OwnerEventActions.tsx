"use client";

import Link from "next/link";

import { PublishEventButton } from "@/components/PublishEventButton";
import { ownerPanelMode } from "@/lib/owner-actions";
import type { LiaEvent } from "@/lib/types";

/** Owner strip on the event detail page (QA 14.08, finding 4): a draft used
 * to render with no publish/edit controls at all — they lived only behind the
 * «···» row in «Мои события». Draft/pending events are visible only to their
 * owner, so the strip needs no extra ownership check (the backend authorises
 * every mutation anyway). */
export function OwnerEventActions({ event }: { event: LiaEvent }) {
  const mode = ownerPanelMode(event.status);
  if (!mode) return null;

  return (
    <div className="border-b border-ink bg-cell-blank px-[20px] py-[12px]">
      <p className="cap">
        {mode === "draft"
          ? "Черновик — видно только вам"
          : "На модерации — опубликуется после одобрения"}
      </p>
      <div className="mt-[8px] flex flex-wrap items-center gap-[10px]">
        {mode === "draft" && <PublishEventButton eventId={event.id} onPublished={() => window.location.reload()} />}
        <Link
          href={`/events/${event.id}/edit`}
          className="swiss-focus hover-invert border border-ink px-[11px] py-[9px] text-[10px] font-bold uppercase tracking-[0.07em]"
        >
          Редактировать
        </Link>
        <Link
          href="/events/mine"
          className="swiss-focus text-[11px] underline underline-offset-2"
        >
          Мои события
        </Link>
      </div>
    </div>
  );
}
