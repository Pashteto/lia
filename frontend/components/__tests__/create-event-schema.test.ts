import { describe, it, expect } from "vitest";
import { eventFormSchema } from "@/components/CreateEventForm";

const base = {
  title: "Тур",
  format: "offline" as const,
  startsAt: "2026-08-01T10:00",
  isFree: true,
  status: "published" as const,
  signupMode: "open" as const,
};

describe("eventFormSchema signup rules", () => {
  it("open with no capacity passes", () => {
    expect(eventFormSchema.safeParse(base).success).toBe(true);
  });

  it("application requires a curator question", () => {
    const r = eventFormSchema.safeParse({ ...base, signupMode: "application" });
    expect(r.success).toBe(false);
  });

  it("application with a question passes", () => {
    const r = eventFormSchema.safeParse({
      ...base, signupMode: "application", curatorQuestion: "Над чем работаете?",
    });
    expect(r.success).toBe(true);
  });

  it("external requires a valid url", () => {
    const bad = eventFormSchema.safeParse({ ...base, signupMode: "external" });
    expect(bad.success).toBe(false);
    const ok = eventFormSchema.safeParse({
      ...base, signupMode: "external", externalRegistrationUrl: "https://t.me/x",
    });
    expect(ok.success).toBe(true);
  });

  it("capacity must be a positive integer when set", () => {
    expect(eventFormSchema.safeParse({ ...base, capacity: 0 }).success).toBe(false);
    expect(eventFormSchema.safeParse({ ...base, capacity: 10 }).success).toBe(true);
  });
});
