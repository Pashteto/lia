import { DiscoveryFeed } from "@/components/DiscoveryFeed";
import { ThemeToggle } from "@/components/ThemeToggle";
import { Button } from "@/components/ui/Button";
import { GlassNav } from "@/components/ui/GlassNav";
import { TabBar } from "@/components/ui/TabBar";
import { fetchPublishedEvents } from "@/lib/api";
import { MOCK_EVENTS } from "@/lib/mock-events";
import Link from "next/link";

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
        title="Lia"
        actions={
          <>
            <ThemeToggle />
            <Link href="/events/new" className="hidden sm:block">
              <Button variant="tinted">Создать событие</Button>
            </Link>
            <Button variant="plain">Войти</Button>
          </>
        }
      />
      <DiscoveryFeed initialEvents={initialEvents} />
      <div className="sm:hidden">
        <TabBar />
      </div>
    </>
  );
}
