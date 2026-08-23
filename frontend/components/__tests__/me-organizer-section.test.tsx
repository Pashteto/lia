import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { OrganizerSection } from "@/components/MeOrganizerSection";

describe("OrganizerSection", () => {
  it("always offers create-event", () => {
    const html = renderToStaticMarkup(
      <OrganizerSection hasOrganizer={false} eventsCount={0} pendingApplications={0} />,
    );
    expect(html).toContain("/events/new");
    expect(html).toContain("Создать событие");
    expect(html).not.toContain("/organizer/applications");
  });

  it("shows cabinet links and applications counter for an organizer", () => {
    const html = renderToStaticMarkup(
      <OrganizerSection hasOrganizer={true} eventsCount={3} pendingApplications={2} />,
    );
    expect(html).toContain("/organizer");
    expect(html).toContain("/events/mine");
    expect(html).toContain("Заявки");
    expect(html).toContain(">2<");
    expect(html).toContain("text-signal");
  });
});
