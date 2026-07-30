import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { EventCover } from "@/components/ui/EventCover";

describe("EventCover", () => {
  it("renders a lazy next/image at the requested aspect when src is given", () => {
    const html = renderToStaticMarkup(
      <EventCover src="/covers/festival.jpg" aspect="aspect-[5/2]" sizes="460px" />,
    );
    expect(html).toContain("aspect-[5/2]");
    expect(html).toContain('data-nimg="fill"');
    expect(html).toContain('loading="lazy"');
    expect(html).toContain('sizes="460px"');
    expect(html).toContain("%2Fcovers%2Ffestival.jpg");
  });

  it("renders the fallback node on cell-blank when src is missing", () => {
    const html = renderToStaticMarkup(
      <EventCover aspect="aspect-[5/2]" sizes="460px" fallback={<span>07</span>} />,
    );
    expect(html).toContain("bg-cell-blank");
    expect(html).toContain("<span>07</span>");
    expect(html).not.toContain("<img");
  });

  it("renders a bare cell-blank box when neither src nor fallback is given", () => {
    const html = renderToStaticMarkup(<EventCover sizes="460px" />);
    expect(html).toContain("bg-cell-blank");
    expect(html).not.toContain("<img");
  });

  it("keeps the photo box on paper so a transparent PNG reads as paper", () => {
    const html = renderToStaticMarkup(<EventCover src="/covers/film.jpg" sizes="460px" />);
    expect(html).toContain("bg-paper");
  });
});
