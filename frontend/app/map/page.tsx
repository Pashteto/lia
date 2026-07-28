import { MapBrowse } from "@/components/MapBrowse";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";

export const metadata = { title: "Карта — PRESENCE" };

// U5 · Карта. Public route — no auth gate.
export default function MapPage() {
  return (
    <>
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="КАРТА" />
      <MapBrowse />
    </>
  );
}
