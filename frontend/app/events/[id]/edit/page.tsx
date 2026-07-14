"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";

import { CreateEventForm, toDatetimeLocalValue, type FormValues } from "@/components/CreateEventForm";
import { fetchEventWithAuth } from "@/lib/api";

/**
 * Edit page for an owner's event (draft or published). Reuses <CreateEventForm>
 * in "edit" mode: fetches the event with the caller's token (so the owner sees
 * their own drafts, which the anonymous GET hides), maps it to the form's
 * `initial` shape, and lets the form PATCH it on submit.
 *
 * `fetchEventWithAuth` returns null both on 404 and when the caller isn't
 * signed in; the backend also 404s (not 403) a non-owner's request to keep the
 * "does this event exist" signal from leaking, so both cases render the same
 * "not found or inaccessible" message.
 */
export default function EditEventPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;

  const { data: event, isLoading, isError } = useQuery({
    queryKey: ["event-edit", id],
    queryFn: () => fetchEventWithAuth(id),
  });

  if (isLoading) {
    return <div className="min-h-screen bg-bg-grouped" />;
  }

  if (isError || !event) {
    return (
      <main className="mx-auto max-w-2xl px-5 py-16 text-center">
        <p className="text-[17px] text-label-secondary">
          Событие не найдено или недоступно.
        </p>
      </main>
    );
  }

  const initial: Partial<FormValues> & { coverFileId?: string; coverPreviewUrl?: string } = {
    title: event.title,
    description: event.description,
    categoryIds: event.categories.map((c) => c.id),
    format: event.format,
    venueId: event.venue?.id ?? "",
    startsAt: toDatetimeLocalValue(event.startsAt),
    endsAt: event.endsAt ? toDatetimeLocalValue(event.endsAt) : undefined,
    isFree: event.priceType === "free",
    priceMin: event.priceMin,
    status: event.status === "published" ? "published" : "draft",
    signupMode: event.signupMode ?? "open",
    capacity: event.capacity,
    curatorQuestion: event.curatorQuestion,
    externalRegistrationUrl: event.externalRegistrationUrl,
    coverPreviewUrl: event.coverUrl,
  };

  return <CreateEventForm mode="edit" eventId={event.id} initial={initial} />;
}
