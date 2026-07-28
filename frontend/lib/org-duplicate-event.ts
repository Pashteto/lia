import { createEvent, type CreateEventInput } from "@/lib/api";
import type { LiaEvent } from "@/lib/types";

/** Maps an existing event into a create payload forced to draft (no cover id). */
export function eventToDuplicateDraftInput(event: LiaEvent): CreateEventInput {
  const endsOk = !!event.endsAt && new Date(event.endsAt).getUTCFullYear() > 1970;
  return {
    title: event.title,
    description: event.description,
    category_ids: event.categories.length > 0 ? event.categories.map((c) => c.id) : undefined,
    venue_id: event.venue?.id,
    status: "draft",
    format: event.format,
    price_type: event.priceType,
    price_min: event.priceType === "free" ? undefined : event.priceMin,
    starts_at: event.startsAt,
    ends_at: endsOk ? event.endsAt : undefined,
    signup_mode: event.signupMode,
    capacity: event.capacity,
    curator_question: event.curatorQuestion,
    external_registration_url: event.externalRegistrationUrl,
  };
}

/** Create a draft copy via POST /events; returns the new event id. */
export async function duplicateAsDraft(event: LiaEvent): Promise<string> {
  const created = await createEvent(eventToDuplicateDraftInput(event));
  return created.id;
}
