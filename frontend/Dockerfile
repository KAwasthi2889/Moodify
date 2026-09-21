# syntax=docker/dockerfile:1
# ── Stage 1: Build Production Vite Bundle ──────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --prefer-offline

COPY . .
RUN npm run build

# ── Stage 2: Serve via High-Performance Nginx ──────────────────
FROM nginx:alpine

ENV BACKEND_HOST=backend \
    NGINX_ENVSUBST_FILTER="BACKEND_HOST"

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/templates/default.conf.template

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
    CMD wget -q --spider http://127.0.0.1/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
