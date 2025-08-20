FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /out/app cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/app /app/app

USER nonroot:nonroot
ENTRYPOINT ["/app/app"]
