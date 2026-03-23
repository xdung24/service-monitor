---
applyTo: "**"
---

# conductor — Project Instructions

## Project Overview

A self-hosted uptime monitoring tool written in Go. Monitors HTTP/HTTPS, TCP, Ping, DNS, and Push/Heartbeat endpoints. Built with Gin (web), modernc.org/sqlite (pure-Go SQLite), golang-migrate (DB migrations), and SSR HTML templates with a dark theme.

- **Module:** `github.com/xdung24/conductor`
- **Go version:** 1.25.5
- **Dev server port:** 8080
- **Entry point:** `cmd/server/main.go`

## Build & Test Commands

```bash
go build ./...           # build everything
go test ./...            # run all tests
go run cmd/server/main.go  # dev server
```

Live reload is configured via `.air.toml` — run `air` for hot-reload development.

## Directory Structure

```
cmd/server/          — main entry point
internal/
  config/            — app configuration
  database/          — DB connection, migration runner
    migrations/      — SQL migration files (0001–0008, up+down)
  models/
    models.go        — Monitor, Heartbeat, User, Notification, NotificationLog structs
    store.go         — typed DB stores (MonitorStore, HeartbeatStore, etc.)
  monitor/
    checker.go       — Checker interface, Run(), HTTPChecker, TCPChecker, PingChecker, DNSChecker
    checker_response.go  — response assertion helpers (header, body type, JSONPath, XPath)
    checker_*_test.go    — tests (use httptest.NewServer as test server)
  notifier/
    notifier.go      — Notifier, dispatches to providers
    slack.go, discord.go, ntfy.go, telegram.go, email.go, webhook.go
  scheduler/         — goroutine-per-monitor scheduler, state-change detection
  web/
    router.go        — Gin router and middleware wiring
    embed.go         — embeds templates dir
    handlers/
      monitors.go    — monitor CRUD, export/import, push endpoint
      notifications.go
      dashboard.go
      auth_token.go
    templates/       — SSR HTML templates (dark theme, Bootstrap-like custom CSS)
```

## Database

- SQLite via `modernc.org/sqlite` (pure Go, no CGo)
- Migrations in `internal/database/migrations/` — filenames: `NNNN_description.up.sql` / `.down.sql`
- Current migration: **0008** — next would be **0009**
- Run migrations automatically on startup

## Monitor Struct — All Fields

```go
// Core
ID, Name, Type, URL, IntervalSeconds, TimeoutSeconds, Active, Retries

// DNS
DNSServer, DNSRecordType, DNSExpected

// HTTP extended
HTTPAcceptedStatuses  // comma-separated codes; empty = 2xx/3xx
HTTPIgnoreTLS         // skip TLS verify (user opt-in)
HTTPMethod            // default GET
HTTPKeyword           // body must contain; empty = skip
HTTPKeywordInvert     // invert keyword check
HTTPUsername, HTTPPassword   // basic auth
HTTPBearerToken       // bearer auth (takes priority over basic)
HTTPMaxRedirects      // 0 = no follow; default 10

// Push/Heartbeat
PushToken             // random hex token; endpoint: /push/:token

// Response assertions (migration 0008)
HTTPHeaderName        // response header to assert; empty = skip
HTTPHeaderValue       // expected value; empty = presence-only
HTTPBodyType          // "": any | "json" | "xml" | "text" | "binary"
HTTPJsonPath          // JSONPath e.g. $.status
HTTPJsonExpected      // empty = just check path exists
HTTPXPath             // XPath e.g. //status or //*[local-name()='tag']
HTTPXPathExpected     // empty = just check node exists
// SMTP monitor fields (migration 0009)
SMTPUseTLS            // bool: implicit TLS (SMTPS / port 465)
SMTPIgnoreTLS         // bool: skip TLS certificate verification
SMTPUsername          // string: AUTH PLAIN username (empty = no auth)
SMTPPassword          // string: AUTH PLAIN password
// Notification trigger settings (migration 0010)
NotifyOnFailure       // bool: send notification on DOWN (default true)
NotifyOnSuccess       // bool: send notification on UP/recovery (default true)
NotifyBodyChars       // int: include first N chars of HTTP body in notification (0 = disabled, max 4096)
```

## Store SQL Column Order (37 columns)

All SELECT queries in store.go must follow this order exactly:
```
id, name, type, url, interval_seconds, timeout_seconds, active, retries,
dns_server, dns_record_type, dns_expected,
http_accepted_statuses, http_ignore_tls, http_method, http_keyword, http_keyword_invert,
http_username, http_password, http_bearer_token, http_max_redirects,
push_token,
http_header_name, http_header_value, http_body_type,
http_json_path, http_json_expected, http_xpath, http_xpath_expected,
smtp_use_tls, smtp_ignore_tls, smtp_username, smtp_password,
notify_on_failure, notify_on_success, notify_body_chars,
created_at, updated_at
```

The same order is used in `List`, `Get`, `Create`, `Update`, and `GetByPushToken`.

## Response Assertion Logic (checker_response.go)

- `checkHeaderConstraint(resp, m)` — checks header presence/value
- `checkBodyType(resp, bodyType)` — validates Content-Type header
- `checkJsonPath(body, path, expected)` — pure-Go JSONPath evaluator
- `checkXPath(body, expr, expected)` — uses `github.com/antchfx/xmlquery`
- `compareExpectedValue(actual, expected)` — operators: `~` contains, `!=`, `>`, `<`, `>=`, `<=` (numeric-aware)

Body is read once only when `HTTPKeyword`, `HTTPJsonPath`, or `HTTPXPath` is non-empty.

For namespaced SOAP/XML, use `//*[local-name()='tagname']` in XPath expressions.

## Adding a New Monitor Type

1. Add `MonitorTypeXxx MonitorType = "xxx"` constant to `internal/models/models.go`
2. Add a new `XxxChecker` struct with a `Check(ctx, m) Result` method in `internal/monitor/checker.go`
3. Register it in `checkerFor()` switch in `checker.go`
4. Add a migration if new DB columns are needed
5. Update `monitorFromForm`, `exportDoc`, `importDoc` in `internal/web/handlers/monitors.go`
6. Add UI section in `internal/web/templates/monitor_form.gohtml`

## Adding a New Notification Provider

1. Create `internal/notifier/PROVIDER.go` implementing the `Provider` interface
2. Register in `internal/notifier/notifier.go`
3. Add config UI in `internal/web/templates/notification_form.gohtml`
4. Update the `notifSummaryMap` helper in `internal/web/handlers/monitors.go`

## Code Conventions

- Error strings: lowercase, no trailing period
- DB column names: snake_case; Go struct fields: PascalCase with `db:"..."` tag
- Test servers: always `httptest.NewServer(...)` with `defer srv.Close()`
- Helper function `baseMonitor(url string) *models.Monitor` exists in test files
- `HTTPPassword` and `PushToken` are **excluded** from monitor export JSON
- Imported monitors start with `Active: false` and name suffix `" (imported)"`
- Golang html/template for SSR; no frontend framework; custom CSS with a dark theme, mobile-responsive
- Use .gohtml extension for templates to enable Go syntax highlighting in editors

## Key Dependencies

| Package | Purpose |
|---|---|
| `github.com/gin-gonic/gin v1.12.0` | HTTP router + middleware |
| `modernc.org/sqlite v1.46.1` | Pure-Go SQLite driver |
| `github.com/golang-migrate/migrate/v4 v4.19.1` | DB migrations |
| `golang.org/x/crypto` | bcrypt password hashing |
| `github.com/antchfx/xmlquery v1.5.0` | XPath evaluation for XML/SOAP |
