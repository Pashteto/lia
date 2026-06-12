# Lia frontend — demo deploy image (Next.js 16, pnpm).
# Runs `next start` on :3001; image optimization via sharp works on Debian/arm64.
FROM node:22-slim

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

RUN pnpm build

EXPOSE 3001
CMD ["pnpm", "start", "-p", "3001"]
