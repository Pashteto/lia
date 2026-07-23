import type { EventStatus } from "@/lib/types";

/** RU reason the signup CTA is unavailable, or null when signup is open. */
export function signupClosedLabel(status: EventStatus): string | null {
  switch (status) {
    case "published":
      return null;
    case "cancelled":
      return "Событие отменено";
    case "rejected":
      return "Событие снято модератором";
    case "draft":
    case "pending_review":
      return "Событие ещё не опубликовано";
  }
}
