import { describe, expect, it } from "vitest";
import { isActiveParticipant } from "@/components/feedback-visibility";

/**
 * «Как прошла встреча?» rendered for any signed-in user on a past event,
 * including one who never attended. The backend rejects with 403
 * ErrNotParticipant (feedback/service.go), so the form was an invitation to
 * hit an error. Mirrors the server rule: status IN ('going','accepted').
 */
describe("isActiveParticipant", () => {
  it("accepts a confirmed attendee", () => {
    expect(isActiveParticipant("going")).toBe(true);
  });

  it("accepts an accepted applicant", () => {
    expect(isActiveParticipant("accepted")).toBe(true);
  });

  it("rejects someone who never signed up — the reported bug", () => {
    expect(isActiveParticipant("")).toBe(false);
    expect(isActiveParticipant(undefined)).toBe(false);
  });

  it("rejects every non-attending rsvp state the backend also rejects", () => {
    for (const status of ["applied", "waitlist", "declined", "withdrawn", "cancelled"]) {
      expect(isActiveParticipant(status)).toBe(false);
    }
  });
});
