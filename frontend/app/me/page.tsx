import { Suspense } from "react";
import { MeProfile } from "@/components/MeProfile";
import { pageTitle } from "@/lib/brand";

export const metadata = { title: pageTitle("Профиль") };

// U6 · Мои записи и профиль. MeProfile reads ?tab= via useSearchParams, which
// needs a Suspense boundary in the App Router.
export default function MePage() {
  return (
    <Suspense fallback={null}>
      <MeProfile />
    </Suspense>
  );
}
