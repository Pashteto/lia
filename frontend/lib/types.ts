// Domain types for the Presence.Tarski frontend. These mirror the backend domain model
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

export type SignupMode = "open" | "application" | "external";

export type RsvpStatus =
  | "going"
  | "waitlist"
  | "applied"
  | "accepted"
  | "declined"
  | "withdrawn"
  | "cancelled";

export interface Organizer {
  id: string;
  name: string;
  /** Short role/affiliation line, e.g. "Музей современного искусства". */
  affiliation?: string;
  avatarUrl?: string;
  /** True when the organizer has been verified by an admin. */
  verified?: boolean;
  /** Organizer profile id for linking to /organizers/{profile_id}. */
  profile_id?: string;
}

export interface Venue {
  id: string;
  name: string;
  /** Метро / district label, e.g. "Парк культуры". */
  metro?: string;
  address?: string;
  district?: string;
  lat?: number;
  lon?: number;
}

export interface EventCategory {
  /** Stable category id (uuid) from the backend. */
  id: string;
  /** Stable slug used for filtering. */
  slug: string;
  /** Russian display label, e.g. "Медиации". */
  label: string;
}

export interface LiaEvent {
  id: string;
  title: string;
  description?: string;
  /** Categories from the curated taxonomy (many-to-many). */
  categories: EventCategory[];
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
  /** Distance from the user's location in metres; set only for nearby results. */
  distanceM?: number;
  signupMode?: SignupMode;
  seatsRemaining?: number;
  /** Empty string means no active RSVP. */
  myRsvpStatus?: RsvpStatus | "";
  curatorQuestion?: string;
  externalRegistrationUrl?: string;
}

export interface Rsvp {
  id: string;
  eventId: string;
  status: RsvpStatus;
  applicationAnswer?: string;
  createdAt: string;
  event?: LiaEvent;
}

/** Shape returned by the backend `GET /api/v1/events` (Presence.Tarski API Event model). */
export interface ApiEvent {
  id: string;
  organizer_id?: string;
  /** Public organizer display data (creator). No email — public surface. */
  organizer?: { uuid?: string; name?: string; avatar_url?: string; verified?: boolean; profile_id?: string };
  venue_id?: string;
  title: string;
  description?: string;
  categories?: { id: string; slug: string; label: string }[];
  venue?: { id: string; name: string; address?: string; metro?: string; district?: string; lat?: number; lon?: number };
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
  /** Present on `/events/nearby` responses; distance from requested coordinates. */
  distance_m?: number;
  cover_url?: string;
  signup_mode?: SignupMode;
  capacity?: number;
  seats_remaining?: number;
  my_rsvp_status?: RsvpStatus | "";
  curator_question?: string;
  external_registration_url?: string;
}
