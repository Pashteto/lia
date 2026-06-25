import type { NextConfig } from "next";

// Cover images are served by the Go backend at `${API}/api/v1/files/{key}`.
// next/image refuses any origin not listed here, so derive the API host from
// the same env the API client uses and allow it. Unsplash stays for mock data.
const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

function apiPattern() {
  try {
    const u = new URL(apiUrl);
    return [
      {
        protocol: u.protocol.replace(":", "") as "http" | "https",
        hostname: u.hostname,
        ...(u.port ? { port: u.port } : {}),
      },
    ];
  } catch {
    return [];
  }
}

const nextConfig: NextConfig = {
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "images.unsplash.com" },
      // Production API host (covers served from there).
      { protocol: "https", hostname: "api.lia.pashteto.com" },
      // Local dev backend.
      { protocol: "http", hostname: "localhost", port: "8080" },
      // Whatever NEXT_PUBLIC_API_URL points at (covers the above + any override).
      ...apiPattern(),
    ],
  },
};

export default nextConfig;
