import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { EventModule } from "@/components/ui/EventModule";

const BASE = {
  numeral: "07",
  category: "Фестивали",
  title: "Летний фестиваль медиаискусства",
  venue: "Музей «Гараж»",
  date: "25–26.07",
  price: "FREE",
  href: "/events/evt-1",
} as const;

describe("EventModule covers", () => {
  it("renders the desktop band at 5:2 and the mobile thumbnail when a cover is given", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} cover="/covers/festival.jpg" />);
    expect(html).toContain("aspect-[5/2]");
    expect(html).toContain('sizes="(max-width: 639px) 1px, (max-width: 1023px) 340px, 460px"');
    expect(html).toContain('sizes="(min-width: 640px) 1px, 44px"');
    expect(html.match(/%2Fcovers%2Ffestival.jpg/g)?.length).toBeGreaterThanOrEqual(2);
  });

  it("keeps every sizes string free of viewport units so the srcset ladder starts at 32w", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} cover="/covers/festival.jpg" />);
    for (const m of html.matchAll(/sizes="([^"]+)"/g)) {
      expect(m[1]).not.toMatch(/\d(vw|vh|vmin|vmax)/);
    }
    expect(html).toContain("w=32&");
  });

  it("renders the numeral plate instead of a photo when no cover is given", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} />);
    expect(html).toContain("bg-cell-blank");
    expect(html).not.toContain("<img");
    // numeral appears in the desktop plate, the mobile plate and the mobile caption
    expect(html.match(/07/g)?.length).toBeGreaterThanOrEqual(3);
    expect(html).toContain("Фестивали");
  });

  it("moves the numeral into the mobile caption and keeps the 44px column", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} cover="/covers/festival.jpg" />);
    expect(html).toContain("grid-cols-[44px_1fr_auto]");
    expect(html).not.toContain("grid-cols-[22px_1fr_auto]");
    expect(html).toContain("07 · Музей «Гараж» · 25–26.07");
  });

  it("still renders the U3 match reason under the content column", () => {
    const html = renderToStaticMarkup(
      <EventModule {...BASE} cover="/covers/festival.jpg" matchReason="маленькая группа" />,
    );
    expect(html).toContain("Совпало: маленькая группа");
  });

  it("wraps everything in exactly one anchor", () => {
    const html = renderToStaticMarkup(<EventModule {...BASE} cover="/covers/festival.jpg" />);
    expect(html.match(/<a /g)).toHaveLength(1);
  });
});
