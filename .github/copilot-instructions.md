# Copilot Instructions for service-monitor

## Repository Overview

**service-monitor** is a self-hosted uptime monitoring tool written entirely in Go.
It uses server-side rendered HTML templates (no frontend build step) and SQLite for storage.

- **Language**: Go 1.25+
- **Module**: `github.com/xdung24/service-monitor`
- **HTTP Framework**: Gin (`github.com/gin-gonic/gin`)
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Migrations**: `github.com/golang-migrate/migrate/v4` with embedded SQL files
- **Auth**: bcrypt passwords + HMAC-signed session cookies
- **Templates**: Go `html/template` embedded via `embed.FS` — parsed at startup, baked into the binary

## Build & Run Commands

```bash
go build -o service-monitor ./cmd/server   # compile
go run ./cmd/server                         # run without building
go build ./...                              # check all packages compile
go vet ./...                                # lint
go test ./...                               # run tests
```

## Project Structure

```
cmd/server/              Entry point (main.go) — HTTP server + graceful shutdown
internal/
  config/                Env-based config (LISTEN_ADDR, DB_PATH, SECRET_KEY)
  database/              SQLite open/close + migration runner
    migrations/          Embedded SQL files (0001_init.up.sql, etc.)
  models/                Data types (Monitor, Heartbeat, User) + DB stores
  monitor/               Monitor checker implementations (HTTP, TCP, Ping)
  scheduler/             Goroutine-per-monitor periodic check scheduler
  notifier/              Notification providers (to be implemented)
  web/
    router.go            Gin router setup + embedded template loading
    embed.go             //go:embed directive for templates/*.html
    router_test.go       Single TestTemplatesParse test (parses all templates)
    handlers/
      dashboard.go       Auth middleware, setup/login/logout, dashboard
      monitors.go        Monitor CRUD handlers
      auth_token.go      HMAC token sign/verify
    templates/           HTML templates (SSR, dark theme)
      partials.html      Shared CSS (styles) and navbar defines
      dashboard.html     Monitor list page
      monitor_form.html  Create/edit monitor form
      monitor_detail.html Monitor heartbeat history
      login.html         Login page
      setup.html         First-run setup wizard
      error.html         Error page
Dockerfile               Multi-stage, non-root, alpine-based
compose.yaml             Docker Compose
Makefile                 build, run, dev, test, lint, clean, docker-build
```

## Key Conventions

- **Module path**: always `github.com/xdung24/service-monitor`
- **Indentation**: tabs (Go standard)
- **Naming**: Go idiomatic — camelCase for Go, snake_case for SQL columns
- **Error handling**: always wrap errors with `fmt.Errorf("context: %w", err)`
- **No CGO**: all dependencies must be pure Go (no CGO required)
- **Templates**: each page is a `{{ define "filename.html" }}` block in its own file, pulling in `{{ template "styles" }}` and `{{ template "navbar" }}` from `partials.html`
- **SQL migrations**: filename format `NNNN_description.up.sql` / `NNNN_description.down.sql`; embedded via `//go:embed` in `database.go`

## Feature Roadmap

All planned, in-progress, and completed features are tracked in **`FEATURES.md`** at the repository root.

**Rules:**
- Whenever a feature is implemented, update its status in `FEATURES.md` from `⬜ Planned` / `🚧 In Progress` to `✅ Done`.
- Whenever a new feature is identified (gap analysis, user request, etc.), add a row to the appropriate section of `FEATURES.md`.
- Before starting any new feature work, mark it `🚧 In Progress` in `FEATURES.md`.

## Adding a New Monitor Type

1. Add a constant to `internal/models/models.go` (`MonitorTypeFoo MonitorType = "foo"`)
2. If the type needs extra DB columns, create migration `NNNN_foo_fields.up.sql` / `.down.sql`
3. Update `Monitor` struct and all SQL queries in `internal/models/store.go`
4. Implement `Checker` interface in `internal/monitor/checker.go` (or a new file)
5. Register it in `checkerFor()` switch in `checker.go`
6. Add the `<option>` to `monitor_form.html`; add any type-specific form fields with JS show/hide
7. Update `monitorFromForm()` in `internal/web/handlers/monitors.go` to parse the new fields
8. Mark the feature `✅ Done` in `FEATURES.md`

## Adding a New Notification Provider

1. Create `internal/notifier/foo.go` implementing the `Provider` interface
2. Register it in `internal/notifier/notifier.go` (`Registry["foo"] = FooProvider{}`)
3. Add a fieldset to `notification_form.html` and toggle it in the existing `showFields()` JS
4. Mark the feature `✅ Done` in `FEATURES.md`

## Testing Philosophy

- **Prefer minimal tests** — do not add unit tests for every function or handler.
- **Write tests only for logic that has real failure modes**: parsers, calculators, token sign/verify, data transformations.
- **Do not write tests for**: HTTP handlers, template rendering, DB queries, or glue code — these are covered by integration/manual testing.
- **Templates**: a single `TestTemplatesParse` in `internal/web/router_test.go` covers all templates at once. No per-template tests needed. Templates are also embedded via `embed.FS` and parsed by `mustParseTemplates()` at startup — a bad template panics immediately before the server accepts requests.
- **Do not suggest adding tests** unless explicitly asked or the code is pure logic (no I/O, no framework deps).

## Database

- SQLite with WAL mode enabled, `_foreign_keys=ON`
- Single writer (`SetMaxOpenConns(1)`)
- Migrations are embedded in the binary — never edit existing migration files, always add new ones

## Security Notes

- Passwords hashed with `bcrypt.DefaultCost`
- Session tokens: HMAC-SHA256 signed, stored in `HttpOnly` cookies
- `SECRET_KEY` env var must be set to a strong random value in production
- Templates use `html/template` (auto-escaping) — never use `text/template` for HTML
