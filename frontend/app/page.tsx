import { DiscoveryFeed } from "@/components/DiscoveryFeed";
import { ThemeToggle } from "@/components/ThemeToggle";
import { Button } from "@/components/ui/Button";
import { GlassNav } from "@/components/ui/GlassNav";
import { TabBar } from "@/components/ui/TabBar";
import Link from "next/link";

// Discovery screen — the worked example for the frontend scaffold.
// Built from design/screens/discovery.html on the Apple-HIG token system.
export default function DiscoveryPage() {
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
      <DiscoveryFeed />
      <div className="sm:hidden">
        <TabBar />
      </div>
    </>
  );
}
