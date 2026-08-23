import type { EventStatus } from "@/lib/types";

/** Owners never see the signup controls on their own event — they get the
 * management strip instead (the backend also rejects owner self-signup). */
export function showSignupControls(event: { isOwner?: boolean }): boolean {
  return !event.isOwner;
}

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
