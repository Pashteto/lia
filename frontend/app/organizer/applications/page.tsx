import { OrganizerApplications } from "@/components/OrganizerApplications";
import { AppHeader, BackToFeedLink, ORG_NAV } from "@/components/ui/AppHeader";

export default function OrganizerApplicationsPage() {
  return (
    <>
      <AppHeader nav={ORG_NAV} mobileCaption="ЗАЯВКИ" actions={<BackToFeedLink />} />
      <OrganizerApplications />
    </>
  );
}
