import { describe, expect, it } from "vitest";

import { parseStatusParam } from "@/components/MyEventsBrowse";

describe("parseStatusParam", () => {
  it("accepts every real filter value", () => {
    expect(parseStatusParam("published")).toBe("published");
    expect(parseStatusParam("pending_review")).toBe("pending_review");
    expect(parseStatusParam("draft")).toBe("draft");
    expect(parseStatusParam("cancelled")).toBe("cancelled");
  });
  it("falls back to all for absent or garbage values", () => {
    expect(parseStatusParam(null)).toBe("all");
    expect(parseStatusParam("")).toBe("all");
    expect(parseStatusParam("hacker")).toBe("all");
    expect(parseStatusParam("ALL")).toBe("all");
  });
});
