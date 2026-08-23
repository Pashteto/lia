import type { EventStatus } from "@/lib/types";

export type OwnerPanelMode = "draft" | "pending" | "published" | null;

/** Which owner panel the event detail page shows. Draft/pending events are
 * only ever served to their owner (anonymous fetches 404), so status alone is
 * a safe ownership proxy there. Published events are public — the panel shows
 * only when the API marked the caller as the owner (is_owner). The backend
 * still authorises every action. */
export function ownerPanelMode(
  status: EventStatus,
  isOwner = false,
): OwnerPanelMode {
  switch (status) {
    case "draft":
      return "draft";
    case "pending_review":
      return "pending";
    case "published":
      return isOwner ? "published" : null;
    default:
      return null;
  }
}
