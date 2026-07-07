# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`notification-api` (formerly `tg-vpn-bot`) is a Go service with two responsibilities:

1. A Telegram bot flow (`internal/tg/bot`) that lets users provision VPN profiles
   (WireGuard/Xray) via chat, backed by `vpn-handler`.
2. A notification module (`internal/notification`) that owns every outbound
   message the service sends — bot replies, approval webhooks, bulk
   announcements, and battery alerts all go through it instead of calling the
   Telegram API directly.

## Build & Run

```bash
make build            # linux/amd64 binary -> ./app (from cmd/main.go)
make build-arm        # linux/arm64 binary
make docker           # buildx build (amd64), push to registry
make docker-arm       # build (arm64), push to registry
make release          # docker-arm + kubectl rollout restart deployment -n util notification-api
```

No lint/test tooling is configured (`go test ./...` works if tests exist, but
there is no Makefile target for it).

## Configuration

Loaded via `internal/config.Load()` from environment variables:

| Var | Purpose |
|---|---|
| `BOT_TOKEN` | Telegram bot API token |
| `HOST_USA`, `HOST_FIN` | `vpn-handler`/3x-ui backend URLs per region |
| `PASSWORD` | Backend auth password |
| `MINI_APP_URL` | URL of the `vpn-front` Telegram mini-app |
| `NODE_EXPORTER_HOST` | Prometheus node_exporter host for battery monitoring |
| `JOBS_FILE_PATH` | Path to the scheduled-announcement jobs JSON file (default `./jobs.json`); skipped entirely if the file doesn't exist |

Note: `cmd/main.go` also configures a hardcoded outbound proxy for Telegram API
traffic — check there if bot connectivity behaves unexpectedly in a new
environment.

## Architecture

```
Telegram Updates (long polling)
  └─> cmd/main.go                     — wiring: config, bot client, notification stack, HTTP server, BatteryService
        └─> internal/tg/bot/router.go — Route(ctx, update): initializes per-user state, dispatches by message text
              └─> internal/tg/bot/handler.go       — flow handlers (region -> platform -> name); calls Notifier, never the Telegram API directly
              └─> internal/tg/bot/stateMachine.go  — in-memory per-user step state (not persisted)
              └─> internal/tg/bot/keyboard.go      — reply keyboards
                    └─> internal/vpnprofile/profile.go — ProfileService.Create(region, platform, name)
                          └─> internal/tg/client/vpnClient.go — HTTP client: session auth, config creation/download
                                └─> vpn-handler / 3x-ui backend (HOST_USA / HOST_FIN)

Every outbound message (bot replies, approvals, announcements, battery alerts):
  └─> internal/notification/service   — NotificationService: composes a message (via template) and dispatches it (via sender)
        ├─> internal/notification/template — Go text/template strings rendered in-process (instruction, approve, battery alert, profile-ready caption)
        └─> internal/notification/sender   — TelegramSender: the only code that calls the Telegram Bot API (SendText/SendHTML/SendPhotoBytes/SendPhotoURL/SendDocument)

HTTP entry point into the notification stack:
  └─> internal/notification/controller — REST handlers, registered on the same mux as everything else
        ├─> POST/GET /api/v1/notifications/approve   — approve/reject webhook (moved from /approve)
        └─> POST     /api/v1/notifications/announce  — bulk announcement endpoint (moved from /api/v1/send/announce)

Scheduled announcements (no HTTP call needed):
  └─> internal/notification/job — Scheduler: reads JOBS_FILE_PATH once at startup,
        registers each task's cron expression via robfig/cron, and calls
        NotificationService.SendAnnounce when it fires. See jobs.example.json
        for the file format: [{ "text", "user_ids": [...], "image_link"?, "cron" }].
        Adding/editing a job requires a service restart — the file is not re-read.
```

- **`internal/vpnprofile`** holds `ProfileService` (VPN profile creation) and
  `BatteryService` (polls `NODE_EXPORTER_HOST` on an interval). `BatteryService`
  only decides *when* to alert — the actual send is delegated to
  `NotificationService.SendBatteryAlert`.
- **User state** (`internal/tg/bot`) is an in-memory map keyed by chat/user ID;
  it does not survive restarts.
- **Mini app integration**: the instruction message links out to `MINI_APP_URL`
  (the `vpn-front` Telegram mini-app) via an inline web_app button.

### Key directories

| Dir | Purpose |
|---|---|
| `cmd/` | Entry point: config load, bot init, notification stack wiring, HTTP server, BatteryService startup |
| `internal/config/` | Env var loading |
| `internal/tg/bot/` | Update routing, flow handlers, keyboards, state machine — decision logic only, no direct Telegram API calls |
| `internal/tg/client/` | `vpnClient` — HTTP client for the VPN backend (session, config CRUD) |
| `internal/vpnprofile/` | `ProfileService` (VPN profile creation), `BatteryService` (battery polling) |
| `internal/notification/controller/` | HTTP handlers for `/api/v1/notifications/*` |
| `internal/notification/service/` | `NotificationService` — orchestrates template + sender per notification type |
| `internal/notification/template/` | Go-template message bodies (instruction, approve/reject, battery alert, profile-ready caption) |
| `internal/notification/sender/` | `TelegramSender` — the only place that calls the Telegram Bot API |
| `internal/notification/job/` | `Scheduler` + `LoadTasks` — cron-driven announcements read from a JSON file at startup |
