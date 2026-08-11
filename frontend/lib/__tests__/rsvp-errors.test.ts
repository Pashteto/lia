import { describe, expect, it } from "vitest";

import { extractHttpStatus, rsvpErrorMessage } from "../rsvp-errors";

describe("extractHttpStatus", () => {
  it("parses the trailing status of an api.ts raw-throw message", () => {
    expect(extractHttpStatus(new Error("sign up failed: 401"))).toBe(401);
    expect(extractHttpStatus(new Error("cancel failed: 500"))).toBe(500);
  });
  it("returns null for messages without a status", () => {
    expect(extractHttpStatus(new Error("not authenticated"))).toBeNull();
    expect(extractHttpStatus(new Error("sign up failed: abc"))).toBeNull();
    expect(extractHttpStatus("boom")).toBeNull();
    expect(extractHttpStatus(null)).toBeNull();
  });
});

describe("rsvpErrorMessage", () => {
  it("never leaks the raw status code text", () => {
    for (const status of [400, 401, 403, 409, 429, 500]) {
      const msg = rsvpErrorMessage(new Error(`sign up failed: ${status}`), "signup");
      expect(msg).not.toContain("failed");
      expect(msg).not.toContain(String(status));
    }
  });
  it("explains a conflict (already signed up)", () => {
    expect(rsvpErrorMessage(new Error("sign up failed: 409"), "signup")).toBe(
      "Похоже, вы уже записаны. Обновите страницу, чтобы увидеть свой статус.",
    );
  });
  it("explains rate limiting", () => {
    expect(rsvpErrorMessage(new Error("sign up failed: 429"), "signup")).toBe(
      "Слишком много попыток. Подождите минуту и попробуйте ещё раз.",
    );
  });
  it("falls back to a generic action-specific message", () => {
    expect(rsvpErrorMessage(new Error("sign up failed: 500"), "signup")).toBe(
      "Не удалось записаться. Попробуйте ещё раз.",
    );
    expect(rsvpErrorMessage(new Error("cancel failed: 500"), "cancel")).toBe(
      "Не удалось отменить запись. Попробуйте ещё раз.",
    );
    expect(rsvpErrorMessage("weird", "signup")).toBe(
      "Не удалось записаться. Попробуйте ещё раз.",
    );
  });
});
