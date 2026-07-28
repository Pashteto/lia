import type { RsvpStatus } from "./types";

export interface RsvpLabel {
  /** Desktop chip text; routed through statusChipVariant() for its tone. */
  long: string;
  /** Compact mobile chip text (U6 mobile rows show «ОК» / «ЖДЁМ»). */
  short: string;
}

const LABELS: Record<RsvpStatus, RsvpLabel> = {
  going: { long: "Подтверждено", short: "ОК" },
  accepted: { long: "Подтверждено", short: "ОК" },
  applied: { long: "Ожидает", short: "ЖДЁМ" },
  waitlist: { long: "В листе ожидания", short: "ЛИСТ" },
  declined: { long: "Отклонена", short: "НЕТ" },
  withdrawn: { long: "Отозвана", short: "ОТОЗВ" },
  cancelled: { long: "Отменено", short: "ОТМ" },
};

const UNKNOWN: RsvpLabel = { long: "—", short: "—" };

/** RSVP status → the two Russian labels U6 renders. «Ожидает» is the only one
 * that resolves to a red chip (the organizer owes the user an answer). */
export function rsvpStatusLabel(status: RsvpStatus): RsvpLabel {
  return LABELS[status] ?? UNKNOWN;
}
