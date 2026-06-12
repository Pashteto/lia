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
  /** Optional: the backend events model has no category yet. */
  category?: EventCategory;
  format: EventFormat;
  status: EventStatus;
  startsAt: string; // ISO 8601
  endsAt?: string;
  priceType: PriceType;
  priceMin?: number; // RUB
  capacity?: number;
  attendeeCount?: number;
  coverUrl?: string;
  /** Optional: the backend exposes organizer_id only (no profile join yet). */
  organizer?: Organizer;
  venue?: Venue;
}

/** Shape returned by the backend `GET /api/v1/events` (Lia API Event model). */
export interface ApiEvent {
  id: string;
  organizer_id?: string;
  venue_id?: string;
  title: string;
  description?: string;
  category?: string;
  venue_name?: string;
  venue_metro?: string;
  status: EventStatus;
  format?: EventFormat;
  price_type?: PriceType;
  price_min?: number;
  price_max?: number;
  external_ticket_url?: string;
  starts_at: string;
  ends_at?: string;
  published_at?: string;
  created_at?: string;
  updated_at?: string;
}
