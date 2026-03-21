# Copilot Instructions for conductor

## Repository Overview

**conductor** is a self-hosted uptime monitoring tool written entirely in Go.
It uses server-side rendered HTML templates (no frontend build step) and SQLite for storage.

- **Language**: Go 1.25+
- **Module**: `github.com/xdung24/conductor`
- **HTTP Framework**: Gin (`github.com/gin-gonic/gin`)
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Migrations**: `github.com/golang-migrate/migrate/v4` with embedded SQL files
- **Auth**: bcrypt passwords + HMAC-signed session cookies
- **Templates**: Go `html/template` embedded via `embed.FS` — parsed at startup, baked into the binary

## Build & Run Commands

```bash
go build -o conductor ./cmd/server   # compile
go run ./cmd/server                         # run without building
go build ./...                              # check all packages compile
go vet ./...                                # lint
go test ./...                               # run tests
```

## Project Structure

```
cmd/server/              Entry point (main.go) — opens DBs, starts schedulers, HTTP server + graceful shutdown
internal/
  config/                Env-based config (LISTEN_ADDR, DATA_DIR, SECRET_KEY)
  database/              SQLite open/close + migration runners + per-user Registry
    migrations_users/    Embedded SQL for shared users.db  (users + push_tokens tables)
    migrations_user/     Embedded SQL for per-user data.db (monitors, heartbeats, notifications, …; 0001_init + 0002_db_monitor)
    registry.go          Registry — lazily opens & caches per-user *sql.DB connections
  models/                Data types (Monitor, Heartbeat, User) + DB stores
  monitor/               Monitor checker implementations (HTTP, TCP, Ping, DNS, SMTP, Push, MySQL, PostgreSQL, Redis, MongoDB)
  scheduler/             Goroutine-per-monitor periodic check scheduler + MultiScheduler
  notifier/              Notification providers (Slack, Discord, ntfy, Telegram, Email, Webhook)
  web/
    router.go            Gin router setup + embedded template loading
    embed.go             //go:embed directive for templates/*.html
    router_test.go       Single TestTemplatesParse test (parses all templates)
    handlers/
      dashboard.go       Auth middleware, setup/login/logout, dashboard, per-request context helpers
      monitors.go        Monitor CRUD handlers + import/export
      notifications.go   Notification provider CRUD + test + log handlers
      users.go           User management handlers (list, create, change-password, delete)
      account.go         2FA (TOTP) setup/disable for the current user
      api_keys.go        API key list/create/delete (stored in users.db)
      maintenance.go     Maintenance window CRUD handlers
      register.go        Public self-registration + invite-token flow; also settingsStore helper
      settings.go        Theme toggle + admin settings page (registration enable/disable)
      status_pages.go    Status page CRUD + public read-only view
      tags.go            Tag CRUD handlers
      auth_token.go      HMAC token sign/verify
    templates/           HTML templates (SSR, dark theme)
      partials.html      Shared CSS (styles) and navbar defines
      dashboard.html     Monitor list page
      monitor_form.html  Create/edit monitor form
      monitor_detail.html Monitor heartbeat history
      notification_list.html  Notification providers list
      notification_form.html  Create/edit notification form
      notification_log.html   Notification send history
      users.html         User management list page
      user_form.html     Create user / change password form
      account_2fa.html   2FA status + TOTP setup QR code page
      admin_settings.html Admin settings page (registration enable/disable)
      api_keys.html      API key list/create/delete page
      login.html         Login page
      login_2fa.html     TOTP second-factor prompt
      maintenance_form.html  Create/edit maintenance window form
      maintenance_list.html  Maintenance windows list
      register.html      Public self-registration / invite-token page
      setup.html         First-run setup wizard
      status_page_form.html  Create/edit status page form
      status_page_list.html  Status pages management list
      status_page_public.html Public read-only status page at /status/:username/:slug
      tags.html          Tag management page (list + inline create form)
      error.html         Error page
Dockerfile               Multi-stage, non-root, alpine-based
compose.yaml             Docker Compose
Makefile                 build, run, dev, test, lint, clean, docker-build
```

## Database Architecture

The app uses **two tiers of SQLite databases** to eliminate write-lock contention between users.

### Shared users database (`DATA_DIR/users.db`)
- Migration source: `internal/database/migrations_users/`
- Tables: `users`, `push_tokens`
- `push_tokens` maps `token → username` so the unauthenticated `GET /push/:token` endpoint can resolve the owning user without touching any per-user DB.
- Opened once at startup by `database.Open` + `database.MigrateUsersDB`.

### Per-user data database (`DATA_DIR/users/<username>/data.db`)
- Migration source: `internal/database/migrations_user/`
- Tables: `monitors`, `heartbeats`, `notifications`, `monitor_notifications`, `notification_logs`, `tags`, `monitor_tags`, `maintenance_windows`, `maintenance_monitors`, `status_pages`, `status_page_monitors`, `docker_hosts`
- Opened lazily on first request via `database.Registry.Get(username)`.
- Each user gets exactly one writer connection (`SetMaxOpenConns(1)`).

### Registry (`internal/database/registry.go`)
- `Registry.Get(username)` opens + migrates + caches the per-user DB.
- `Registry.Remove(username)` closes and evicts a cached connection (used on user deletion).
- `Registry.Close()` closes all connections on shutdown.

### MultiScheduler (`internal/scheduler/scheduler.go`)
- Wraps `map[username]*Scheduler`.
- `StartForUser(username, db)` creates and starts a Scheduler for one user (no-op if already running).
- `ForUser(username)` returns the per-user Scheduler.
- `StopUser(username)` stops and removes a single user's Scheduler.

### Auth middleware (`AuthRequired()` in `dashboard.go`)
After validating the session cookie it calls `registry.Get(username)` and injects two keys into the Gin context:
- `"sm_user_db"` → `*sql.DB` (per-user DB)
- `"sm_username"` → `string`

All protected handlers access stores via context helpers defined on `Handler`:
- `h.monitorStore(c)`, `h.heartbeatStore(c)`, `h.notifStore(c)`, `h.notifLogStore(c)`, `h.schedFor(c)` — per-user stores that require a Gin context
- `h.tagStore(c)`, `h.maintenanceStore(c)`, `h.statusPageStore(c)` — per-user stores for tags, maintenance windows, and status pages
- `h.apiKeyStore()`, `h.settingsStore()`, `h.regTokenStore()` — shared-DB stores (no context arg; read from `h.usersDB` directly)

## Key Conventions

### Timezone Convention

All datetimes are stored and transmitted in **UTC** throughout the stack. Follow these rules consistently:

- **Go / database layer**: always call `.UTC()` before passing a `time.Time` to any SQL query parameter. SQLite stores datetimes as text; passing a non-UTC value produces a string with a `+HH:MM` offset that breaks lexicographic comparisons against the stored UTC strings.
- **JSON API responses**: all timestamp fields are formatted with `time.RFC3339` after `.UTC()` — they always carry a `Z` suffix (e.g. `"2026-03-21T10:00:00Z"`).
- **Authenticated pages (dashboard, chart modal)**: JavaScript receives UTC ISO strings. Use `new Date(tsString)` and then `toLocaleString()` / `toLocaleTimeString()` **without** a `timeZone` option — the browser automatically converts to the visitor's local timezone.
- **Public status pages**: JavaScript also receives UTC ISO strings, but must display in UTC. Use `toLocaleTimeString([], {timeZone: 'UTC'})` / `toLocaleString([], {timeZone: 'UTC'})` and append `' UTC'` to the label so the timezone is visible.
- **Server-side rendered template values** (e.g. heartbeat history, maintenance window times): format with `time.RFC3339` in the handler; the template displays the raw string. If local display is needed, convert client-side in JS.

---

- **Module path**: always `github.com/xdung24/conductor`
- **Indentation**: tabs (Go standard)
- **Naming**: Go idiomatic — camelCase for Go, snake_case for SQL columns
- **Error handling**: always wrap errors with `fmt.Errorf("context: %w", err)`
- **Error checking**: always check every error return value — never discard with `_` unless the call genuinely cannot fail or the error is intentionally ignored (document why with a comment). This applies to all store methods (`Create`, `Update`, `Delete`, `SetActive`, etc.) and any other function that returns an `error`.
- **No CGO**: all dependencies must be pure Go (no CGO required)
- **Templates**: each page is a `{{ define "filename.html" }}` block in its own file, pulling in `{{ template "styles" }}` and `{{ template "navbar" }}` from `partials.html`
- **SQL migrations**: filename format `NNNN_description.up.sql` / `NNNN_description.down.sql`; embedded via `//go:embed` in `database.go`
- **Never edit existing migration files** — always add a new numbered migration
- **Database context (noctx)**: never call `(*sql.DB).Exec`, `(*sql.DB).Query`, or `(*sql.DB).QueryRow` directly — always use the `Context` variants (`ExecContext`, `QueryContext`, `QueryRowContext`) passing `context.Background()` at minimum

## Web UI Conventions

### Template data contract
Every protected page is rendered via `c.HTML(status, "page.html", h.pageData(c, gin.H{...}))`.

`pageData()` merges the caller-supplied map with exactly **one** common key:
- `IsAdmin bool` — `true` when the logged-in user has the admin role.

There is **no** `Username` or `Theme` key in the template data.  The theme is read from the
`sm_theme` cookie by a tiny inline `<script>` inside `{{ define "styles" }}` in `partials.html`,
which sets `data-theme` on `<html>`.  Usernames are stored only in the Gin context (key
`"sm_username"`), not passed to templates.

Before editing a template, **always read the corresponding handler** in
`internal/web/handlers/` to confirm the exact keys it puts in `gin.H{}`.

Common keys used by individual pages (set by the handler, not by `pageData`):

| Template | Extra keys |
|---|---|
| `monitor_form.html` | `Monitor *models.Monitor`, `IsNew bool`, `Error string`, `AllNotifs`, `LinkedNotifIDs`, `NotifSummaries`, `AllTags`, `LinkedTagIDs` |
| `notification_form.html` | `Notif *models.Notification`, `IsNew bool`, `Error string` |
| `dashboard.html` | `Monitors []models.Monitor`, `Stats` |
| `users.html` | `Users []models.User`, `CurrentUser string` |
| `account_2fa.html` | `Enabled bool`, `Flash string`, `Error string`; optionally `SetupMode bool`, `QRDataURI template.URL`, `TOTPSecret string` |
| `admin_settings.html` | `Flash string`, `Error string`, `RegistrationEnabled bool` |
| `api_keys.html` | `Keys []models.APIKey`, `Flash string`; optionally `NewToken string` (shown once after creation) |
| `maintenance_form.html` | `Window *models.MaintenanceWindow`, `IsNew bool`, `AllMonitors []models.Monitor`, `LinkedMonitorIDs map[int64]bool`, `Error string` |
| `maintenance_list.html` | `Windows []models.MaintenanceWindow`, `Flash string` |
| `register.html` | `Token string`, `Disabled bool`, `Error string` |
| `status_page_form.html` | `Page *models.StatusPage`, `IsNew bool`, `AllMonitors []models.Monitor`, `LinkedMonitorIDs map[int64]bool`, `Error string` |
| `status_page_list.html` | `Pages []models.StatusPage`, `Flash string` |
| `tags.html` | `Tags []*models.Tag`, `Flash string`; optionally `NewForm bool`, `Tag *models.Tag`, `Error string` |

### Dual-theme CSS
The dark theme is the base.  Every new CSS rule that is colour- or background-sensitive
**must** have a `[data-theme="light"]` override added in the same `{{ define "styles" }}`
block (in `partials.html` for shared classes, or in the page template for page-specific ones).

```css
/* dark (base) */
.my-class { background: #1e293b; color: #e2e8f0; }
/* light override — required for every colour/background rule */
[data-theme="light"] .my-class { background: #ffffff; color: #0f172a; }
```

### Monitor-type field show/hide pattern
Fields specific to one monitor type live in a `<div id="TYPE-fields" ...>` with an inline
`style` that sets the initial visibility from the template data:

```html
<div id="foo-fields" style="display:{{ if eq .Monitor.Type "foo" }}block{{ else }}none{{ end }};">
    <!-- foo-specific fields -->
</div>
```

After adding the div, add a corresponding line to `toggleTypeFields()` at the bottom of
`monitor_form.html`:

```js
document.getElementById('foo-fields').style.display = (t === 'foo') ? 'block' : 'none';
```

If the new type has no URL/address field, also add it to the `noURL` list in the same
function so the shared URL input is hidden.

### Notification-provider field show/hide pattern
Same idea, but in `notification_form.html` using `showFields(type)`.  Each provider's
fields are in a `<div id="fields-PROVIDER">` and toggled by adding a `case` to the switch
inside `showFields()`.

### Template validation
After any template change run:

```bash
go test ./internal/web/...
```

`TestTemplatesParse` in `internal/web/router_test.go` parses every template and will fail
immediately on syntax errors.  The server itself also panics during startup when a template
fails to parse, so issues are caught before serving any request.

### General HTML/template rules
- Always use `html/template` (auto-escaping).  Never use `text/template` for HTML output.
- Each page template must `{{ define "filename.html" }}` and call
  `{{ template "styles" . }}` and `{{ template "navbar" . }}`.
- For user-supplied content embedded in JS strings, use `{{ .Value | js }}`.

---

## Feature Roadmap

All planned, in-progress, and completed features are tracked in **`FEATURES.md`** at the repository root.

**Rules:**
- Whenever a feature is implemented, update its status in `FEATURES.md` from `⬜ Planned` / `🚧 In Progress` to `✅ Done`.
- Whenever a new feature is identified (gap analysis, user request, etc.), add a row to the appropriate section of `FEATURES.md`.
- Before starting any new feature work, mark it `🚧 In Progress` in `FEATURES.md`.

## Adding a New Monitor Type

1. Add a constant to `internal/models/models.go` (`MonitorTypeFoo MonitorType = "foo"`)
2. If the type needs extra DB columns, create migration `NNNN_foo_fields.up.sql` / `.down.sql` in `migrations_user/`; add the column to **all five** queries in `store.go` (`List`, `Get`, `Create`, `Update`, `GetByPushToken`)
3. Update the `Monitor` struct in `internal/models/models.go` with the new field(s)
4. Implement `Checker` interface in a new `internal/monitor/checker_<type>.go` file
5. Register it in `checkerFor()` switch in `checker.go`
6. Add the `<option>` to `monitor_form.html`; add any type-specific form fields with JS show/hide
7. Update `monitorFromForm()` in `internal/web/handlers/monitors.go` to parse the new fields; update export/import structs too
8. Add a sample import file `examples/monitor-<type>.json` covering the new type's key fields
9. Mark the feature `✅ Done` in `FEATURES.md`

## Adding a New Notification Provider

1. Create `internal/notifier/foo.go` implementing the `Provider` interface
2. Register it in `internal/notifier/notifier.go` (`Registry["foo"] = FooProvider{}`)
3. Add a fieldset to `notification_form.html` and toggle it in the existing `showFields()` JS
4. Mark the feature `✅ Done` in `FEATURES.md`

## User Management Routes

All routes are under the `AuthRequired` middleware group:

| Method | Path | Handler |
|--------|------|---------|
| GET | `/admin/users` | `UserList` |
| GET | `/admin/users/new` | `UserNew` |
| POST | `/admin/users` | `UserCreate` |
| GET | `/admin/users/:username/password` | `UserPasswordPage` |
| POST | `/admin/users/:username/password` | `UserChangePassword` |
| POST | `/admin/users/:username/delete` | `UserDelete` |

Deleting: cannot delete own account or last user. Deletion also calls `UnregisterAllPushTokens`, `StopUser`, `Registry.Remove`.

## Testing Philosophy

- **Prefer minimal tests** — do not add unit tests for every function or handler.
- **Write tests only for logic that has real failure modes**: parsers, calculators, token sign/verify, data transformations.
- **Do not write tests for**: HTTP handlers, template rendering, DB queries, or glue code — these are covered by integration/manual testing.
- **Templates**: a single `TestTemplatesParse` in `internal/web/router_test.go` covers all templates at once. No per-template tests needed. Templates are also embedded via `embed.FS` and parsed by `mustParseTemplates()` at startup — a bad template panics immediately before the server accepts requests.
- **Do not suggest adding tests** unless explicitly asked or the code is pure logic (no I/O, no framework deps).

## Database

- SQLite with WAL mode enabled, `_foreign_keys=ON`
- Single writer per DB file (`SetMaxOpenConns(1)`)
- Migrations are embedded in the binary — never edit existing migration files, always add new ones
- Two embed FSes in `database.go`: `usersMigrationsFS` (`migrations_users/*.sql`) and `userMigrationsFS` (`migrations_user/*.sql`)

## Communication Style

- Do not add emojis or icons to responses unless explicitly asked.

## Security Notes

- Passwords hashed with `bcrypt.DefaultCost`
- Session tokens: HMAC-SHA256 signed, stored in `HttpOnly` cookies
- `SECRET_KEY` env var must be set to a strong random value in production
- Templates use `html/template` (auto-escaping) — never use `text/template` for HTML
- Push tokens are registered in `users.db` so they can be resolved without exposing per-user DB paths
