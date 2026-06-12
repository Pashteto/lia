import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  images: {
    // Scaffold-only: mock covers are served from Unsplash. Replace with the
    // real S3/CDN image host when wiring the API.
    remotePatterns: [
      { protocol: "https", hostname: "images.unsplash.com" },
    ],
  },
};

export default nextConfig;
