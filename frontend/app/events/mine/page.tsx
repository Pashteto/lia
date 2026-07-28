import Link from "next/link";

import { MyEventsBrowse } from "@/components/MyEventsBrowse";
import { AppHeader, ORG_NAV } from "@/components/ui/AppHeader";

export default function MyEventsPage() {
  return (
    <>
      <AppHeader
        nav={ORG_NAV}
        mobileCaption="МОИ СОБЫТИЯ"
        actions={
          <Link
            href="/events/new"
            className="swiss-focus text-[11px] font-bold uppercase tracking-[0.07em] sm:hidden"
            aria-label="Создать событие"
          >
            +
          </Link>
        }
      />
      <MyEventsBrowse />
    </>
  );
}
