import { MyEventsBrowse } from "@/components/MyEventsBrowse";
import { AppHeader, ORG_NAV } from "@/components/ui/AppHeader";

export default function MyEventsPage() {
  return (
    <>
      <AppHeader nav={ORG_NAV} mobileCaption="МОИ СОБЫТИЯ" />
      <MyEventsBrowse />
    </>
  );
}
