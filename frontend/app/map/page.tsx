import Link from "next/link";

import { MapBrowse } from "@/components/MapBrowse";

export default function MapPage() {
  return (
    <main className="mx-auto max-w-3xl px-4 py-6">
      <Link href="/" className="mb-4 inline-flex items-center text-[17px] text-accent">
        ‹ События
      </Link>
      <MapBrowse />
    </main>
  );
}
