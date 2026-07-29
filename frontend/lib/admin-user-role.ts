/** A4 role chip label. Derived, not stored: the backend only knows
 * users.role ("common" | "admin") and whether an organizers row exists.
 * Priority: test account → staff → organizer → viewer. */
export function adminUserRoleLabel(
  user: { role: string; is_organizer: boolean },
  opts: { test?: boolean } = {},
): string {
  if (opts.test) return "Тестовый";
  if (user.role === "admin") return "Админ";
  if (user.is_organizer) return "Организатор";
  return "Зритель";
}
