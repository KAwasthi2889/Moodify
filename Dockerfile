# ── Stage 1: Build Go Backend Binary ───────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# ── Stage 2: Minimal Runtime with Audio Fingerprinting ─────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata chromaprint ffmpeg

WORKDIR /app
COPY --from=builder /app/server /app/server
RUN mkdir -p /app/uploads

EXPOSE 8080

ENTRYPOINT ["/app/server"]
