import { OrganizerHub } from "@/components/OrganizerHub";
import { AppHeader, BackToFeedLink, ORG_NAV } from "@/components/ui/AppHeader";

export default function OrganizerPage() {
  return (
    <>
      <AppHeader nav={ORG_NAV} mobileCaption="КАБИНЕТ" actions={<BackToFeedLink />} />
      <OrganizerHub />
    </>
  );
}
