import type { Metadata } from "next";
import { cookies } from "next/headers";
import { CITY_COOKIE, cityBySlug } from "@/lib/city";
import { type CityAvailability, CityProvider } from "@/lib/city-context";
import { fetchCities } from "@/lib/api";
import { Golos_Text, Manrope, JetBrains_Mono } from "next/font/google";
import { Providers } from "./providers";
import { TabBarGate } from "@/components/ui/TabBarGate";
import { VerifyEmailBanner } from "@/components/VerifyEmailBanner";
import "./globals.css";

/* Swiss Grid faces. The handoff specifies Archivo / Space Grotesk, which have
 * no Cyrillic on Google Fonts — Golos Text / Manrope are the Cyrillic-complete
 * substitutes (master plan, decision checkpoint 2). Self-hosted by next/font. */
const golos = Golos_Text({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "600", "700", "900"],
  variable: "--font-golos",
  display: "swap",
});
const manrope = Manrope({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "700"],
  variable: "--font-manrope",
  display: "swap",
});
const jbmono = JetBrains_Mono({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "700"],
  variable: "--font-jbmono",
  display: "swap",
});

export async function generateMetadata(): Promise<Metadata> {
  const city = cityBySlug((await cookies()).get(CITY_COOKIE)?.value);
  return {
    title: "PRESENCE — События",
    description: `Медиации, лекции и разговоры об искусстве. Участливые культурные события ${city.genitive}.`,
  };
}

export default async function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const citySlug = (await cookies()).get(CITY_COOKIE)?.value;
  // Availability comes from the backend (cities.spb_available); a failed
  // fetch degrades to the static flags in lib/city (msk-only).
  const availability: CityAvailability | null = await fetchCities()
    .then((list) =>
      Object.fromEntries(list.map((c) => [c.code, c.available])) as CityAvailability,
    )
    .catch(() => null);
  return (
    <html
      lang="ru"
      className={`h-full antialiased ${golos.variable} ${manrope.variable} ${jbmono.variable}`}
    >
      <body className="min-h-full bg-paper font-ui text-ink">
        <Providers>
          <CityProvider initialSlug={citySlug} availability={availability}>
            <VerifyEmailBanner />
            {children}
          </CityProvider>
        </Providers>
        <TabBarGate />
      </body>
    </html>
  );
}
