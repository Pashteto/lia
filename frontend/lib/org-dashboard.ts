import { seatsFill } from "./org-seats";

export interface OrgDashboardStats {
  published: number;
  pendingReview: number;
  drafts: number;
  /** Sum of seatsFill.filled across events where seatsFill != null; null if none measurable */
  totalRegistrations: number | null;
}

export function orgDashboardStats(
  events: ReadonlyArray<{
    status?: string;
    capacity?: number | null;
    seatsRemaining?: number | null;
  }>,
): OrgDashboardStats {
  let published = 0;
  let pendingReview = 0;
  let drafts = 0;
  let totalRegistrations = 0;
  let hasMeasurable = false;

  for (const event of events) {
    switch (event.status) {
      case "published":
        published++;
        break;
      case "pending_review":
        pendingReview++;
        break;
      case "draft":
        drafts++;
        break;
    }

    const fill = seatsFill(event);
    if (fill != null) {
      hasMeasurable = true;
      totalRegistrations += fill.filled;
    }
  }

  return {
    published,
    pendingReview,
    drafts,
    totalRegistrations: hasMeasurable ? totalRegistrations : null,
  };
}

/** Soonest upcoming (startsAt >= now) by startsAt asc; else null. */
export function nextOrganizerEvent<T extends { startsAt: string }>(
  events: ReadonlyArray<T>,
  now: Date = new Date(),
): T | null {
  const nowMs = now.getTime();
  let best: T | null = null;
  let bestMs = Infinity;

  for (const event of events) {
    const ms = new Date(event.startsAt).getTime();
    if (ms >= nowMs && ms < bestMs) {
      best = event;
      bestMs = ms;
    }
  }

  return best;
}
