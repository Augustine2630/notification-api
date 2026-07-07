# Build stage
FROM golang:1.25.11 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=arm64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /app/app ./cmd

# Runtime stage
FROM debian:bookworm-slim
# rsvg-convert + корневые сертификаты
RUN apt-get update && apt-get install -y --no-install-recommends \
      librsvg2-bin \
      ca-certificates \
      curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/app /app/app
COPY jobs.json /app/jobs.json

# каталог для файлов, доступный пользователю nobody
RUN mkdir -p /app/data && chown -R nobody:nogroup /app/data

USER nobody

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:80/health || exit 1

ENTRYPOINT ["/app/app"]
