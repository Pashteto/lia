import { OrganizerProfileEdit } from "@/components/OrganizerProfileEdit";
import { AppHeader, BackToFeedLink, ORG_NAV } from "@/components/ui/AppHeader";

export const metadata = { title: "Профиль организатора — PRESENCE" };

export default function MyOrganizerPage() {
  return (
    <>
      <AppHeader nav={ORG_NAV} mobileCaption="ПРОФИЛЬ" actions={<BackToFeedLink />} />
      <OrganizerProfileEdit />
    </>
  );
}
