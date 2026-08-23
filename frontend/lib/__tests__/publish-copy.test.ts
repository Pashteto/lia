import { describe, expect, it } from "vitest";
import { moderationNote, publishDialogCopy } from "@/lib/publish-copy";

describe("publishDialogCopy", () => {
  it("verified → instant-publish copy", () => {
    const c = publishDialogCopy(true);
    expect(c.body).toContain("сразу появится в ленте");
    expect(c.confirmLabel).toBe("Опубликовать");
  });

  it("unverified → moderation copy", () => {
    const c = publishDialogCopy(false);
    expect(c.title).toContain("модерацию");
    expect(c.body).toContain("модерацию");
    expect(c.confirmLabel).toBe("Отправить");
  });

  it("wizard note matches", () => {
    expect(moderationNote(true)).toContain("сразу");
    expect(moderationNote(false)).toContain("Модерация");
  });
});
