/** Russian label that feeds StatusChip / statusChipVariant. */
export function orgEventStatusLabel(status: string): string {
  switch (status) {
    case "published":
      return "Опубликовано";
    case "draft":
      return "Черновик";
    case "pending_review":
      return "На модерации";
    case "rejected":
      return "Снято модератором";
    case "cancelled":
      return "Отменено";
    default:
      return status;
  }
}
