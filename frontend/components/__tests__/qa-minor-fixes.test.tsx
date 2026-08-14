import { readFileSync } from "node:fs";
import { join } from "node:path";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import NotFound from "@/app/not-found";
import { CalendarLegend } from "@/components/CalendarView";
import { BackToFeedLink } from "@/components/ui/AppHeader";

describe("404 copy (QA 14.08, finding 8)", () => {
  it("does not claim the page was an unpublished event", () => {
    const html = renderToStaticMarkup(<NotFound />);
    expect(html).not.toContain("событие сняли с публикации");
    expect(html).toContain("Проверьте адрес");
  });
});

describe("Calendar legend", () => {
  it("explains the filled-day encoding", () => {
    const html = renderToStaticMarkup(<CalendarLegend />);
    expect(html).toContain("есть события");
    expect(html).toContain("выбранный день");
  });

  it("is wired into the month view", () => {
    const src = readFileSync(join(__dirname, "../CalendarView.tsx"), "utf8");
    expect(src.match(/<CalendarLegend/g)?.length).toBeGreaterThanOrEqual(1);
  });
});

describe("Organizer area escape hatch", () => {
  it("BackToFeedLink points home", () => {
    const html = renderToStaticMarkup(<BackToFeedLink />);
    expect(html).toContain('href="/"');
    expect(html).toContain("Лента");
  });

  it("organizer hub pages render it in the header", () => {
    for (const page of [
      "../../app/organizer/page.tsx",
      "../../app/organizer/applications/page.tsx",
      "../../app/events/mine/page.tsx",
      "../../app/me/organizer/page.tsx",
    ]) {
      const src = readFileSync(join(__dirname, page), "utf8");
      expect(src, page).toContain("BackToFeedLink");
    }
  });
});
