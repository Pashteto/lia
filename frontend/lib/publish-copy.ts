/** Publish-dialog copy that tells the truth about moderation (QA-23-aug №3):
 * verified organizers publish straight to the feed; everyone else goes
 * through pre-moderation (pending_review). */
export function publishDialogCopy(verified: boolean): {
  title: string;
  body: string;
  confirmLabel: string;
} {
  return verified
    ? {
        title: "Опубликовать событие?",
        body: "Событие сразу появится в ленте. Его можно будет отредактировать или вернуть в черновики.",
        confirmLabel: "Опубликовать",
      }
    : {
        title: "Отправить на модерацию?",
        body: "Событие уйдёт на модерацию — обычно это занимает до 24 часов. После одобрения оно появится в ленте.",
        confirmLabel: "Отправить",
      };
}

/** Short wizard-footer note matching the dialog. */
export function moderationNote(verified: boolean): string {
  return verified ? "Публикация — сразу в ленту" : "Модерация до 24 часов";
}
