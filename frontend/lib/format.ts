import type { LiaEvent } from "./types";

// Pin the timezone: without it, formatting uses the runtime zone, so the SSR
// container (UTC) and the visitor's browser (local zone) produce different text
// and React throws a hydration mismatch (#418). Moscow events → Europe/Moscow.
const dateFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "Europe/Moscow",
});

/** "13 июня, 19:00" */
export function formatEventDate(iso: string): string {
  return dateFmt.format(new Date(iso));
}

/** "Бесплатно" | "2 500 ₽" | "от 500 ₽" */
export function formatPrice(event: LiaEvent): string {
  if (event.priceType === "free") return "Бесплатно";
  const amount = (event.priceMin ?? 0).toLocaleString("ru-RU");
  return event.priceType === "from" ? `от ${amount} ₽` : `${amount} ₽`;
}

/** "11 / 18 участников" | "64 участника" */
export function formatAttendance(event: LiaEvent): string | null {
  if (event.attendeeCount == null) return null;
  return event.capacity != null
    ? `${event.attendeeCount} / ${event.capacity} участников`
    : `${event.attendeeCount} участников`;
}
