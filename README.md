# service-monitor

A production-ready, self-hosted uptime monitoring tool written in Go.

## Features

- HTTP / HTTPS, TCP, Ping, DNS, SMTP, Push/Heartbeat monitor types
- HTTP extended checks: keyword match, JSON path, XPath, custom headers/body, bearer auth, redirect control
- Server-side rendered dashboard (no JavaScript build step)
- **Per-user SQLite databases** — each user's monitors and notifications live in their own database file, eliminating write-lock contention
- Automatic schema migrations (embedded in binary)
- bcrypt password hashing, HMAC-signed session cookies
- Notification providers: Slack, Discord, ntfy, Telegram, Email, Webhook
- User management admin panel — add, delete, and change passwords
- Monitor import / export (JSON)
- Graceful shutdown
- Docker + Docker Compose support
- Single compiled binary — no runtime dependencies

## Quick Start

```bash
# Run directly
go run ./cmd/server

# Open http://localhost:3001
# Follow the setup wizard to create your admin account
```

## Configuration

All config is via environment variables:

| Variable      | Default                   | Description                          |
|---------------|---------------------------|--------------------------------------|
| `LISTEN_ADDR` | `:3001`                   | HTTP listen address                  |
| `DATA_DIR`    | `./data`                  | Root data directory                  |
| `SECRET_KEY`  | `change-me-in-production` | HMAC key for sessions                |

### Database layout

```
DATA_DIR/
  users.db                          # shared — user accounts + push token index
  users/<username>/data.db           # per-user — monitors, heartbeats, notifications
```

Each user's data is isolated in their own SQLite file. The shared `users.db` holds only the `users` table and a `push_tokens` index (needed for unauthenticated `/push/:token` routing).

## Docker

```bash
docker compose up --build
```

## Build

```bash
make build   # compile binary
make run     # build + run
make test    # run tests
make lint    # vet + staticcheck
```

## Project Structure

```
cmd/server/          Entry point
internal/
  config/            Environment config
  database/          SQLite open + migrate + per-user Registry
    migrations_users/  SQL migrations for shared users.db
    migrations_user/   SQL migrations for per-user data.db
  models/            Data models + DB stores
  monitor/           Monitor checker implementations
  scheduler/         Periodic check scheduler + MultiScheduler
  notifier/          Notification providers
  web/
    handlers/        HTTP request handlers
    templates/       HTML templates (SSR)
Dockerfile
compose.yaml
Makefile
```

## Roadmap

- [ ] Public status pages
- [ ] Certificate expiry monitoring
- [ ] Latency sparkline charts
- [ ] API keys (token-based access)
- [ ] 2FA (TOTP)
- [ ] Maintenance windows
- [ ] Tags / labels on monitors
