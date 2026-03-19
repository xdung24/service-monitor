# conductor

A production-ready, self-hosted uptime monitoring tool written in Go.

## Features

- HTTP / HTTPS, TCP, Ping, DNS, SMTP, Push/Heartbeat monitor types
- HTTP extended checks: keyword match, JSON path, XPath, custom headers/body, bearer auth, redirect control, TLS certificate expiry alerts
- Database monitors: MySQL/MariaDB, PostgreSQL, MongoDB, Redis
- Server-side rendered dashboard (no JavaScript build step)
- **Per-user SQLite databases** — each user's monitors and notifications live in their own database file, eliminating write-lock contention
- Automatic schema migrations (embedded in binary)
- bcrypt password hashing, HMAC-signed session cookies
- **API key authentication** — generate named tokens; `Authorization: Bearer` header accepted alongside session cookies
- **Two-factor authentication (TOTP)** — QR code setup, per-user opt-in, enforced two-step login
- **Role-based access control** — first user (via startup token) becomes admin; admin-only routes for user management and settings
- **Account registration** — open registration or invite-link only; admin controls from settings page
- Notification providers: Slack, Discord, ntfy, Telegram, Email (SMTP), Webhook
- Public status pages — read-only pages showing selected monitors
- Maintenance windows — suppress alerts during scheduled downtime
- Tags / labels on monitors — color-coded, displayed on dashboard
- Monitor import / export (JSON)
- Dark/light theme toggle
- Latency sparkline charts
- User management admin panel — add, promote/demote, delete, change passwords
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

- [ ] WebSocket Upgrade monitor
- [ ] Docker Container monitor
- [ ] More notification providers (PagerDuty, Gotify, Pushover, Matrix)
- [ ] Proxy management
- [ ] SNMP, MQTT, gRPC, SIP, Radius monitor types
- [ ] Real Browser (Chromium) monitor
- [ ] System Service (systemd/SCM) monitor
