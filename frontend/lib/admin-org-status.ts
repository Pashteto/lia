/** Maps org moderation status to a Russian label for StatusChip. */
export function adminOrgStatusLabel(
  status: string,
  opts?: { test?: boolean },
): string {
  if (opts?.test) return "Тестовый";
  switch (status) {
    case "pending":
      return "На проверке";
    case "verified":
      return "Верифицирован";
    case "rejected":
      return "Отклонён";
    case "draft":
      return "Черновик";
    default:
      return status;
  }
}
