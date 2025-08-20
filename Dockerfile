FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /out/app cmd/main.go

FROM alpine:3.20 AS runtime

RUN apk add --no-cache librsvg

WORKDIR /app
COPY --from=builder /out/app /app/app

USER nobody
ENTRYPOINT ["/app/app"]
