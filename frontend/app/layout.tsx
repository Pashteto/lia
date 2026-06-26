import type { Metadata } from "next";
import { Providers } from "./providers";
import { TabBar } from "@/components/ui/TabBar";
import "./globals.css";

// No webfont import: the design system uses the system font stack only
// (SF Pro on Apple devices, OS font elsewhere) — see ../../design/DESIGN.md.

export const metadata: Metadata = {
  title: "Presence.Tarski — События",
  description:
    "Discovery участливых культурных практик: медиации, мастер-классы, лекции, читательские группы.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ru" className="h-full antialiased" suppressHydrationWarning>
      <head>
        {/* Apply the saved/OS theme before first paint to avoid a flash and to
            keep the explicit choice winning over prefers-color-scheme. */}
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){try{var t=localStorage.getItem("theme");if(t!=="light"&&t!=="dark"){t=matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light";}document.documentElement.classList.add(t);}catch(e){}})();`,
          }}
        />
      </head>
      <body className="min-h-full bg-bg-grouped text-label">
        <Providers>{children}</Providers>
        <TabBar />
      </body>
    </html>
  );
}
