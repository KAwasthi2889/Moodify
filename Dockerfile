# syntax=docker/dockerfile:1
# ── Stage 1: Build Go Backend Binary ───────────────────────────
FROM golang:alpine AS builder

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
    PYTHON_BIN=python3 \
    DEBIAN_FRONTEND=noninteractive \
    TZ=UTC

# Install system audio DSP & fingerprinting utilities with apt cache mounts
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    ffmpeg \
    libchromaprint-tools \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /app

# Optimization 1: Pre-install CPU-only PyTorch wheel
# This completely bypasses NVIDIA CUDA 12/13 dependencies, saving ~8 GB of dead weight!
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir torch --index-url https://download.pytorch.org/whl/cpu

# Optimization 2: Install remaining Python libraries (cached unless requirements.txt changes)
COPY python/requirements.txt ./python/requirements.txt
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir -r python/requirements.txt

# Optimization 3: Application source layers (change frequently, cached after heavy deps)
COPY python/ ./python/
COPY --from=builder /app/server /app/server
RUN mkdir -p /app/uploads

EXPOSE 8080

ENTRYPOINT ["/app/server"]
