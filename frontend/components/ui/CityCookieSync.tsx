"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { CITY_COOKIE, type CitySlug } from "@/lib/city";
import { writeCityCookie } from "@/lib/city-context";

/** Rendered by pages that honor a ?city= override: persists the override into
 * the cookie so the choice sticks on navigation, then re-renders the header
 * (which is cookie-driven) once. No-op when the cookie already matches. */
export function CityCookieSync({ slug }: { slug: CitySlug }) {
  const router = useRouter();
  useEffect(() => {
    const current = document.cookie
      .split("; ")
      .find((c) => c.startsWith(`${CITY_COOKIE}=`))
      ?.split("=")[1];
    if (current !== slug) {
      writeCityCookie(slug);
      router.refresh();
    }
  }, [slug, router]);
  return null;
}
