import type { EventStatus } from "@/lib/types";

export type OwnerPanelMode = "draft" | "pending" | null;

/** Which owner panel the event detail page shows. Draft/pending events are
 * only ever served to their owner (anonymous fetches 404), so status alone is
 * a safe ownership proxy here — the backend still authorises every action. */
export function ownerPanelMode(status: EventStatus): OwnerPanelMode {
  switch (status) {
    case "draft":
      return "draft";
    case "pending_review":
      return "pending";
    default:
      return null;
  }
}
