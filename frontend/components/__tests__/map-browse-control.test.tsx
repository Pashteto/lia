import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { ResponsiveMapRegion } from "@/components/map/ResponsiveMapRegion";
import { createLatestRequestGate } from "@/lib/latest-request-gate";

describe("ResponsiveMapRegion", () => {
  it("renders exactly one desktop map beside the rail", () => {
    const calls: boolean[] = [];
    const html = renderToStaticMarkup(
      <ResponsiveMapRegion
        mobile={false}
        listRail={<div data-region="rail" />}
        renderMap={(mobile) => {
          calls.push(mobile);
          return <div data-region="map" />;
        }}
      />,
    );

    expect(calls).toEqual([false]);
    expect(html).toContain('data-region="rail"');
    expect(html.match(/data-region="map"/g)).toHaveLength(1);
  });

  it("renders exactly one mobile map without the desktop rail", () => {
    const calls: boolean[] = [];
    const html = renderToStaticMarkup(
      <ResponsiveMapRegion
        mobile
        listRail={<div data-region="rail" />}
        renderMap={(mobile) => {
          calls.push(mobile);
          return <div data-region="map" />;
        }}
      />,
    );

    expect(calls).toEqual([true]);
    expect(html).not.toContain('data-region="rail"');
    expect(html.match(/data-region="map"/g)).toHaveLength(1);
  });
});

describe("createLatestRequestGate", () => {
  it("invalidates an earlier request when a later request starts", () => {
    const gate = createLatestRequestGate();
    const earlier = gate.begin();
    const later = gate.begin();

    expect(gate.isLatest(earlier)).toBe(false);
    expect(gate.isLatest(later)).toBe(true);
  });
});
