import { Suspense } from "react";
import Link from "next/link";

import { MyEventsBrowse } from "@/components/MyEventsBrowse";
import { AppHeader, BackToFeedLink, ORG_NAV } from "@/components/ui/AppHeader";

// MyEventsBrowse reads ?status= via useSearchParams, which needs a Suspense
// boundary in the App Router.
export default function MyEventsPage() {
  return (
    <>
      <AppHeader
        nav={ORG_NAV}
        mobileCaption="МОИ СОБЫТИЯ"
        actions={
          <>
            <BackToFeedLink />
            <Link
              href="/events/new"
              className="swiss-focus text-[11px] font-bold uppercase tracking-[0.07em] sm:hidden"
              aria-label="Создать событие"
            >
              +
            </Link>
          </>
        }
      />
      <Suspense fallback={null}>
        <MyEventsBrowse />
      </Suspense>
    </>
  );
}
