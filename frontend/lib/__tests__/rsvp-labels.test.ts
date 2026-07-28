import { describe, expect, it } from "vitest";
import { rsvpStatusLabel } from "../rsvp-labels";
import { statusChipVariant } from "../status-chip";
import type { RsvpStatus } from "../types";

describe("rsvpStatusLabel", () => {
  it("confirmed states share the Подтверждено label", () => {
    expect(rsvpStatusLabel("going")).toEqual({ long: "Подтверждено", short: "ОК" });
    expect(rsvpStatusLabel("accepted")).toEqual({ long: "Подтверждено", short: "ОК" });
  });
  it("pending states", () => {
    expect(rsvpStatusLabel("applied")).toEqual({ long: "Ожидает", short: "ЖДЁМ" });
    expect(rsvpStatusLabel("waitlist")).toEqual({ long: "В листе ожидания", short: "ЛИСТ" });
  });
  it("closed states", () => {
    expect(rsvpStatusLabel("declined")).toEqual({ long: "Отклонена", short: "НЕТ" });
    expect(rsvpStatusLabel("withdrawn")).toEqual({ long: "Отозвана", short: "ОТОЗВ" });
    expect(rsvpStatusLabel("cancelled")).toEqual({ long: "Отменено", short: "ОТМ" });
  });
  it("unknown status degrades to em dashes", () => {
    expect(rsvpStatusLabel("nonsense" as RsvpStatus)).toEqual({ long: "—", short: "—" });
  });
});

describe("rsvp label → chip tone", () => {
  it("only «Ожидает» is red; confirmed is ink-filled; the rest are outlines", () => {
    expect(statusChipVariant(rsvpStatusLabel("applied").long)).toBe("signal");
    expect(statusChipVariant(rsvpStatusLabel("going").long)).toBe("active");
    expect(statusChipVariant(rsvpStatusLabel("accepted").long)).toBe("active");
    for (const s of ["waitlist", "declined", "withdrawn", "cancelled"] as RsvpStatus[]) {
      expect(statusChipVariant(rsvpStatusLabel(s).long)).toBe("default");
    }
  });
});
