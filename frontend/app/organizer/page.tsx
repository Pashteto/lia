import { OrganizerHub } from "@/components/OrganizerHub";
import { AppHeader, ORG_NAV } from "@/components/ui/AppHeader";

export default function OrganizerPage() {
  return (
    <>
      <AppHeader nav={ORG_NAV} mobileCaption="КАБИНЕТ" />
      <OrganizerHub />
    </>
  );
}
