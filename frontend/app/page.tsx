import { AuthButton } from "@/components/AuthButton";
import { DiscoveryFeed } from "@/components/DiscoveryFeed";
import { BrandLogo } from "@/components/ui/BrandLogo";
import { GlassNav } from "@/components/ui/GlassNav";
import { ThemeSwitch } from "@/components/ui/ThemeSwitch";
import { fetchPublishedEvents } from "@/lib/api";
import { MOCK_EVENTS } from "@/lib/mock-events";

// Discovery screen — the worked example for the frontend scaffold.
// Built from design/screens/discovery.html on the Apple-HIG token system.
//
// Fetches published events from the backend server-side (SSR, good for SEO per
// the tech-stack doc). Falls back to mock data when the backend is unreachable
// so the page still renders during local frontend-only development.
export default async function DiscoveryPage() {
  const initialEvents = await fetchPublishedEvents().catch(() => MOCK_EVENTS);

  return (
    <>
      <GlassNav
        title={<BrandLogo />}
        actions={
          <>
            <ThemeSwitch />
            <AuthButton />
          </>
        }
      />
      <DiscoveryFeed initialEvents={initialEvents} />
    </>
  );
}
