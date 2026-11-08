# ---- Build Stage ----
FROM node:20-alpine AS builder

WORKDIR /build

COPY package.json package-lock.json* ./
RUN npm ci

COPY . .
RUN npm run build

# ---- Runtime Stage ----
FROM node:20-alpine

WORKDIR /app

COPY --from=builder /build/package.json .
COPY --from=builder /build/node_modules ./node_modules
COPY --from=builder /build/.next ./.next
COPY --from=builder /build/public ./public

EXPOSE 3000

ENV NODE_ENV=production

CMD ["node_modules/.bin/next", "start"]
