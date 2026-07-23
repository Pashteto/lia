import { describe, expect, it } from "vitest";
import { signupClosedLabel } from "@/lib/signup-availability";

describe("signupClosedLabel", () => {
  it("open for published", () => {
    expect(signupClosedLabel("published")).toBeNull();
  });
  it("cancelled → отменено", () => {
    expect(signupClosedLabel("cancelled")).toBe("Событие отменено");
  });
  it("rejected → снято модератором", () => {
    expect(signupClosedLabel("rejected")).toBe("Событие снято модератором");
  });
  it("draft / pending_review are not signup-able", () => {
    expect(signupClosedLabel("draft")).toBe("Событие ещё не опубликовано");
    expect(signupClosedLabel("pending_review")).toBe("Событие ещё не опубликовано");
  });
});
