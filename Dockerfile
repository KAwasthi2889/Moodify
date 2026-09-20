# syntax=docker/dockerfile:1
# ── Stage 1: Build Go Backend Binary ───────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd/ cmd/
COPY internal/ internal/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# ── Stage 2: Runtime with Audio DSP, Neural Models & Fingerprinting
FROM python:3.11-slim-bookworm

ENV PYTHONUNBUFFERED=1 \
    DEBIAN_FRONTEND=noninteractive \
    TZ=UTC

# Install system audio DSP & fingerprinting utilities
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    ffmpeg \
    libchromaprint-tools \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /app

# Optimize layer caching: Install Python dependencies before copying source
COPY python/requirements.txt ./python/requirements.txt
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir -r python/requirements.txt

# Copy Python scripts and compiled Go server
COPY python/ ./python/
COPY --from=builder /app/server /app/server
RUN mkdir -p /app/uploads

EXPOSE 8080

ENTRYPOINT ["/app/server"]
