import { describe, expect, it } from "vitest";
import { adminUserRoleLabel } from "@/lib/admin-user-role";

describe("adminUserRoleLabel", () => {
  it("labels a plain account Зритель", () => {
    expect(adminUserRoleLabel({ role: "common", is_organizer: false })).toBe("Зритель");
  });

  it("labels an organizer owner Организатор", () => {
    expect(adminUserRoleLabel({ role: "common", is_organizer: true })).toBe("Организатор");
  });

  it("labels staff Админ, outranking the organizer flag", () => {
    expect(adminUserRoleLabel({ role: "admin", is_organizer: true })).toBe("Админ");
  });

  it("labels test accounts Тестовый, outranking everything", () => {
    expect(adminUserRoleLabel({ role: "admin", is_organizer: true }, { test: true })).toBe("Тестовый");
  });
});
