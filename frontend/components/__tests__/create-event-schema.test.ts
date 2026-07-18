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

  // Review #2 finding #5: "leave empty = unlimited" must hold.
  it("empty or absent capacity is allowed (unlimited)", () => {
    expect(eventFormSchema.safeParse({ ...base, capacity: "" }).success).toBe(true);
    expect(eventFormSchema.safeParse({ ...base, capacity: undefined }).success).toBe(true);
    const noKey = { ...base } as Record<string, unknown>;
    delete noKey.capacity;
    expect(eventFormSchema.safeParse(noKey).success).toBe(true);
  });

  // Review #2 finding #4: a blank date must yield the human RU message, not the
  // raw "Invalid input: expected string, received undefined".
  it("missing date reports the Russian message, never the raw Zod string", () => {
    const undef = eventFormSchema.safeParse({ ...base, startsAt: undefined });
    expect(undef.success).toBe(false);
    if (!undef.success) {
      const msg = undef.error.issues.find((i) => i.path[0] === "startsAt")?.message;
      expect(msg).toBe("Укажите дату и время");
      expect(msg).not.toContain("Invalid input");
    }
    const empty = eventFormSchema.safeParse({ ...base, startsAt: "" });
    expect(empty.success).toBe(false);
    if (!empty.success) {
      expect(empty.error.issues.find((i) => i.path[0] === "startsAt")?.message).toBe(
        "Укажите дату и время",
      );
    }
  });
});
