import { readFileSync } from "node:fs";
import { join } from "node:path";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { AdminModeration } from "@/components/AdminModeration";

describe("AdminModeration on narrow screens (QA 14.08, finding 6)", () => {
  it("collapses the queue/record grid to one column under 900px", () => {
    // SSR pass = loading skeleton; it must carry the same responsive shell.
    const html = renderToStaticMarkup(<AdminModeration />);
    expect(html).toContain("max-[899px]:grid-cols-1");
  });

  it("is no longer gated by AdminDesktopOnly on the moderation page", () => {
    const page = readFileSync(
      join(__dirname, "../../app/admin/moderation/events/page.tsx"),
      "utf8",
    );
    expect(page).not.toContain("AdminDesktopOnly");
  });
});
