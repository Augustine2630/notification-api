# ---------- build stage ----------
FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /out/app cmd/main.go

# ---------- runtime ----------
FROM debian:bookworm-slim

# rsvg-convert + корневые сертификаты
RUN apt-get update && apt-get install -y --no-install-recommends \
      librsvg2-bin \
      ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/app /app/app

# каталог для файлов, доступный пользователю nobody
RUN mkdir -p /app/data && chown -R nobody:nogroup /app/data

# (необязательно) если хотите, чтобы /app/data жил как volume:
# VOLUME ["/app/data"]

USER nobody
ENTRYPOINT ["/app/app"]
