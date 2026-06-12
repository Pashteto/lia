import type { Metadata } from "next";
import { Providers } from "./providers";
import "./globals.css";

// No webfont import: the design system uses the system font stack only
// (SF Pro on Apple devices, OS font elsewhere) — see ../../design/DESIGN.md.

export const metadata: Metadata = {
  title: "Lia — События",
  description:
    "Discovery участливых культурных практик: медиации, мастер-классы, лекции, читательские группы.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ru" className="h-full antialiased">
      <body className="min-h-full bg-bg-grouped text-label">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
