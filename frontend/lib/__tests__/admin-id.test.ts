import { describe, expect, it } from "vitest";
import { adminShortId } from "../admin-id";

describe("adminShortId", () => {
  it("prefixes EV- and uses first 4 hex", () => {
    expect(adminShortId("a1b2c3d4-5678-4012-8000-000000000001")).toBe("EV-A1B2");
  });
});
