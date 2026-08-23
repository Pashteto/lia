import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { AppHeader } from "@/components/ui/AppHeader";
import { CityControl } from "@/components/ui/CityControl";

describe("AppHeader mobileCaption", () => {
  it("wraps a string caption in the cap span (unchanged behaviour)", () => {
    const html = renderToStaticMarkup(<AppHeader mobileCaption="ПОДБОР" />);
    expect(html).toContain('class="cap whitespace-nowrap"');
    expect(html).toContain("ПОДБОР");
  });

  it("renders a ReactNode caption as-is (city control button)", () => {
    const html = renderToStaticMarkup(<AppHeader mobileCaption={<CityControl />} />);
    expect(html).toContain('aria-haspopup="listbox"');
    expect(html).toContain("МСК");
    // The node must not be wrapped in the string-caption span: nesting a
    // button inside the cap span is invalid markup and inherits caption styles.
    expect(html).not.toMatch(/<span class="cap"><div/);
  });
});
