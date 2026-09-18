# Lia frontend — deploy image (Next.js 16, pnpm), two stages.
#
# Stage 1 builds with the full dev toolchain; stage 2 ships ONLY Next's traced
# standalone server. Result: ~200-300 MB instead of the ~1.06 GB single-stage
# image. Reason (2026-09-16): the Selectel box has a 20 GB disk and every
# frontend redeploy adds another image until the old ones are pruned.
#
# Requires `output: "standalone"` in next.config.ts — keep the two in sync.
# Runs `node server.js` on :3001 (standalone has its own server; `next start`
# would need the whole node_modules tree and is NOT used here).

FROM node:22-slim AS builder

WORKDIR /app
RUN npm install -g pnpm@10

# Install deps (incl. devDeps — needed by `next build`).
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

# App source.
COPY . .

# Baked into the client bundle + used by SSR. Points at a dead port so the app
# cleanly falls back to its built-in mock data (lib/mock-events.ts) for the demo.
ARG NEXT_PUBLIC_API_URL=http://127.0.0.1:9
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# Yandex Maps JS API v2.1 key (public, Referer-restricted). Empty default is
# fine: without a key the build still succeeds and the map falls back to the
# "Карта недоступна" placeholder.
ARG NEXT_PUBLIC_YANDEX_MAPS_KEY=
ENV NEXT_PUBLIC_YANDEX_MAPS_KEY=$NEXT_PUBLIC_YANDEX_MAPS_KEY

RUN pnpm build

FROM node:22-slim AS runner

WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3001
ENV HOSTNAME=0.0.0.0

# next/image optimisation needs sharp at runtime. It is only a transitive
# optional dep of Next (see pnpm-workspace.yaml onlyBuiltDependencies), so the
# standalone trace may not carry it — install it explicitly instead of finding
# out in prod that every cover 500s.
RUN npm install --no-save --omit=dev sharp \
    && npm cache clean --force

# Static assets are NOT part of the standalone bundle: they must be copied next
# to it, keeping the .next/static path.
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3001
CMD ["node", "server.js"]
