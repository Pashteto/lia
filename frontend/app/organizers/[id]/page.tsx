import { PublicOrganizerView } from "@/components/PublicOrganizerView";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";

export default function PublicOrganizerPage() {
  return (
    <>
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} />
      <PublicOrganizerView />
    </>
  );
}
