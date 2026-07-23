import type { LiaEvent } from "@/lib/types";

/** Google Calendar wants "YYYYMMDDTHHMMSSZ" in UTC. */
function gcalStamp(iso: string): string {
  return new Date(iso).toISOString().replace(/[-:]/g, "").replace(/\.\d{3}/, "");
}

/**
 * Deep-link that opens a pre-filled event in the user's Google Calendar.
 * No OAuth, no backend — closes the "не интегрируется с Google" gap (QA 4a).
 * .ics (existing "В календарь") still covers Apple/Outlook.
 */
export function googleCalendarUrl(event: LiaEvent): string {
  const start = new Date(event.startsAt);
  const end = event.endsAt
    ? new Date(event.endsAt)
    : new Date(start.getTime() + 2 * 60 * 60 * 1000);
  const location = [event.venue?.name, event.venue?.address].filter(Boolean).join(", ");
  const params = new URLSearchParams({
    action: "TEMPLATE",
    text: event.title,
    dates: `${gcalStamp(start.toISOString())}/${gcalStamp(end.toISOString())}`,
  });
  if (event.description) params.set("details", event.description);
  if (location) params.set("location", location);
  return `https://calendar.google.com/calendar/render?${params.toString()}`;
}
