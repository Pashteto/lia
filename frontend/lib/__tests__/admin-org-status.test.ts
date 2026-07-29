import { describe, expect, it } from "vitest";
import { adminOrgStatusLabel } from "../admin-org-status";
import { statusChipVariant } from "../status-chip";

describe("adminOrgStatusLabel", () => {
  it("maps and feeds StatusChip", () => {
    expect(adminOrgStatusLabel("pending")).toBe("На проверке");
    expect(statusChipVariant(adminOrgStatusLabel("pending"))).toBe("signal");
    expect(statusChipVariant(adminOrgStatusLabel("verified"))).toBe("active");
    expect(adminOrgStatusLabel("verified", { test: true })).toBe("Тестовый");
  });
});
