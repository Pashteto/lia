import { describe, expect, it } from "vitest";
import { orgVerificationStep } from "../org-verification";

describe("orgVerificationStep", () => {
  it("maps statuses", () => {
    expect(orgVerificationStep("draft")).toBe(0);
    expect(orgVerificationStep("rejected")).toBe(0);
    expect(orgVerificationStep("pending")).toBe(1);
    expect(orgVerificationStep("verified")).toBe(2);
  });
});
