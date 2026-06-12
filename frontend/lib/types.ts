// Domain types for the Lia frontend. These mirror the backend domain model
// described in docs/event_discovery_mvp_technical_stack.md (§13). UI copy is
// Russian; identifiers and types stay English.

export type EventFormat = "offline" | "online";

export type PriceType = "free" | "fixed" | "from";

export type EventStatus =
  | "draft"
  | "pending_review"
  | "published"
  | "rejected"
  | "cancelled";

export interface Organizer {
  id: string;
  name: string;
  /** Short role/affiliation line, e.g. "Музей современного искусства". */
  affiliation?: string;
  avatarUrl?: string;
}

export interface Venue {
  id: string;
  name: string;
  /** Метро / district label, e.g. "Парк культуры". */
  metro?: string;
  address?: string;
}

export interface EventCategory {
  /** Stable slug used for filtering. */
  slug: string;
  /** Russian display label, e.g. "Медиации". */
  label: string;
}

export interface LiaEvent {
  id: string;
  title: string;
  description?: string;
  category: EventCategory;
  format: EventFormat;
  status: EventStatus;
  startsAt: string; // ISO 8601
  endsAt?: string;
  priceType: PriceType;
  priceMin?: number; // RUB
  capacity?: number;
  attendeeCount?: number;
  coverUrl?: string;
  organizer: Organizer;
  venue?: Venue;
}
