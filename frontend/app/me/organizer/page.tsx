import { OrganizerProfileEdit } from "@/components/OrganizerProfileEdit";
import { AppHeader, BackToFeedLink, ORG_NAV } from "@/components/ui/AppHeader";
import { pageTitle } from "@/lib/brand";

export const metadata = { title: pageTitle("Профиль организатора") };

export default function MyOrganizerPage() {
  return (
    <>
      <AppHeader nav={ORG_NAV} mobileCaption="ПРОФИЛЬ" actions={<BackToFeedLink />} />
      <OrganizerProfileEdit />
    </>
  );
}
