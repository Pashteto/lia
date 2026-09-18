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
  // Ship only the traced server bundle instead of the whole node_modules tree:
  // the deploy image drops from ~1.06 GB to a few hundred MB. That matters on
  // the Selectel box (2026-09-15 migration): a 10 GB disk cannot hold two or
  // three 1 GB frontend images while old ones wait to be pruned.
  // The Dockerfile's runner stage copies .next/standalone + .next/static +
  // public and starts `node server.js` (NOT `next start`).
  output: "standalone",

  // Intuitive-but-wrong URLs people actually type (QA-23-aug №12).
  async redirects() {
    return [
      { source: "/auth/signup", destination: "/signup", permanent: false },
      { source: "/events/create", destination: "/events/new", permanent: false },
    ];
  },
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "images.unsplash.com" },
      // Production API host (covers served from there).
      { protocol: "https", hostname: "api.lia.pashteto.com" },
      // Every host covers may be served from. This list must stay a SUPERSET of
      // whatever STORAGE_PUBLIC_BASE points at: the whitelist is baked at build
      // time, so switching the backend to a new host without rebuilding the
      // frontend makes next/image answer 400 and every cover disappears
      // (hit in prod 2026-09-02 during the presencehq.ru migration).
      { protocol: "https", hostname: "presencehq.ru" },
      { protocol: "https", hostname: "api.tarski.ru" },
      { protocol: "https", hostname: "p.tarski.ru" },
      { protocol: "https", hostname: "presence.tarski.ru" },
      { protocol: "https", hostname: "api.presence.tarski.ru" },
      // Local dev backend.
      { protocol: "http", hostname: "localhost", port: "8080" },
      // Whatever NEXT_PUBLIC_API_URL points at (covers the above + any override).
      ...apiPattern(),
    ],
  },
};

export default nextConfig;
