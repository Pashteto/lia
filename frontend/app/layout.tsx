import type { Metadata } from "next";
import { Golos_Text, Manrope, JetBrains_Mono } from "next/font/google";
import { Providers } from "./providers";
import { TabBar } from "@/components/ui/TabBar";
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

export const metadata: Metadata = {
  title: "PRESENCE — События",
  description:
    "Медиации, лекции и разговоры об искусстве. Участливые культурные события Москвы.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      lang="ru"
      className={`h-full antialiased ${golos.variable} ${manrope.variable} ${jbmono.variable}`}
    >
      <body className="min-h-full bg-paper font-ui text-ink">
        <Providers>
          <VerifyEmailBanner />
          {children}
        </Providers>
        <TabBar />
      </body>
    </html>
  );
}
