# Conductor Test Plan

This plan defines what to test, how to test it, and which automation type to use:
- Go unit tests
- Bruno functional API tests
- k6 performance tests

The plan is risk-based. It does not attempt to automate every possible behavior.

## 1. Objectives

- Catch logic regressions early with fast unit tests.
- Validate user-facing and API workflows with Bruno.
- Protect key SLO-sensitive paths with k6.
- Keep maintenance cost reasonable by avoiding low-value automation.

## 2. Test Pyramid For This App

- Unit tests (largest): pure logic, parsers, token/signature logic, provider payload generation, monitor-checker logic that does not require real external systems.
- Functional tests (medium): critical HTTP routes and end-to-end flows through Gin handlers and database state.
- Performance tests (smallest): public and authenticated hot paths, cache-backed endpoints, and scheduler-scale smoke checks.

## 3. Environments

- Local dev:
  - Server: `go run ./cmd/server`
  - SQLite in temporary or local `DATA_DIR`
  - Bruno env: `tests/functional/environments/local.bru`
  - k6 env: `tests/performance/environments/local.json`
- CI:
  - Run all unit tests on every pull request.
  - Run Bruno smoke suite on every pull request.
  - Run k6 smoke profile on merge to main and nightly.
- Pre-release:
  - Full Bruno regression suite.
  - Full k6 baseline + stress profiles.

## 4. Coverage Strategy By Feature Family

## 4.1 Authentication, Sessions, Security

Features:
- Login/logout, 2FA, API keys
- CSRF, rate limiting, security headers
- Session expiry and key signing

Automation:
- Unit (Go): token sign/verify, session-age checks, CSRF helper logic, rate-limit window logic.
- Functional (Bruno): login success/fail, protected route access, admin-only route denied for normal user, API-key auth success/fail, CSRF rejection on mutating endpoints.
- Performance (k6): light smoke for login and a read-only authenticated endpoint under concurrency.

Not automated deeply:
- Browser-specific cookie/UI behavior (manual spot checks are sufficient).

## 4.2 Monitor Checkers (All Types)

Features:
- Core monitor types and extended HTTP capabilities.
- Niche protocol monitors (Steam, SIP, Radius, Kafka, gRPC, SNMP, MQTT, etc.).

Automation:
- Unit (Go): checker logic with mocks/fakes where feasible, response parsing, status mapping, timeout/error behavior.
- Functional (Bruno): CRUD monitor endpoints + simple check trigger/readback for representative types only.
- Performance (k6): monitor list/read endpoints only; checker internals are not k6 targets.

Representative functional subset (high value):
- HTTP
- TCP
- DNS
- Push
- One database monitor (MySQL or PostgreSQL)
- One message/protocol monitor (MQTT or gRPC)

Not automated deeply:
- Exhaustive matrix for every protocol against real third-party infra in CI.

## 4.3 Notifications And Providers

Features:
- Provider registry and payload delivery for many providers.
- Notification logs, badge count, browser notification polling.

Automation:
- Unit (Go): provider payload composition and required-field validation using httptest servers.
- Functional (Bruno): notification CRUD, test-send endpoint behavior, notification logs/count endpoint contract.
- Performance (k6): notification logs/count endpoint concurrency smoke.

Not automated deeply:
- Real network delivery to each external provider in CI (use mock HTTP servers only).

## 4.4 Status Pages, Summary, Charts, Caching

Features:
- Public status page HTML
- Public summary JSON
- Chart-data endpoints (public cached + private realtime)

Automation:
- Unit (Go): chart data shaping and downtime aggregation logic.
- Functional (Bruno): summary endpoint schema and status codes, chart endpoint auth/public access rules.
- Performance (k6): summary and public chart endpoints with cache-aware thresholds.

Not automated deeply:
- Pixel-perfect HTML rendering checks.

## 4.5 Scheduler, Downtime, Maintenance, Multi-user Isolation

Features:
- Per-user scheduler lifecycle
- Downtime event tracking
- Maintenance suppression
- Per-user DB isolation

Automation:
- Unit (Go): scheduler state transitions and downtime event lifecycle where logic is isolated.
- Functional (Bruno): end-to-end maintenance window create/update/delete and effect on monitor alert state; multi-user isolation on key list/read/write routes.
- Performance (k6): smoke load on monitor list and summary endpoints with multiple users.

Not automated deeply:
- Long-running time-based scheduler soak in every PR (run nightly only).

## 4.6 Admin/User Management, Registration, Password Reset

Features:
- Admin user CRUD, role updates, disable/enable account
- Invite/registration and password-reset flow

Automation:
- Unit (Go): token creation/validation logic for reset/invite where pure.
- Functional (Bruno): complete admin and self-service workflows with positive and negative cases.
- Performance (k6): not a primary target, only optional smoke.

Not automated deeply:
- Full email rendering/visual validation (assert API contract and send invocation only).

## 5. What We Intentionally Do Not Automate

- Duplicate tests for framework behavior already covered by Gin/Go stdlib.
- Per-template visual snapshots for each page.
- Live tests against every external notification/service provider.
- Full protocol compatibility matrices in PR CI.

These are covered by:
- Targeted unit tests
- Representative functional flows
- Manual release checklist where needed

## 6. Automation Inventory To Build

## 6.1 Unit Test Buckets (Go)

- Config parsing and secret-key derivation behavior.
- Auth token/signature and expiry behavior.
- Monitor checker parser and status logic.
- Notification provider payload/required fields.
- Data transformation logic (chart, downtime, summary shaping).

Target:
- Fast suite under 2 minutes in CI.

## 6.2 Functional Suite (Bruno)

Organize under folders:
- `tests/functional/auth/`
- `tests/functional/monitors/`
- `tests/functional/notifications/`
- `tests/functional/status-pages/`
- `tests/functional/admin-users/`
- `tests/functional/maintenance/`

Each folder should include:
- Happy-path flow
- Auth/permission negative case
- Validation/400-level negative case

Target:
- Smoke subset under 3 minutes per PR.
- Full suite under 10 minutes nightly.

## 6.3 Performance Suite (k6)

Profiles:
- Smoke: low VUs, short duration for CI merge checks.
- Baseline: moderate VUs for release comparisons.
- Stress: high VUs/nightly to detect saturation.

Required scripts:
- health endpoint
- summary endpoint
- public chart endpoint
- authenticated monitor list endpoint

Core thresholds (starting point):
- `http_req_failed < 1%`
- p95 latency:
  - health: < 200 ms
  - summary/chart cached endpoints: < 150 ms
  - authenticated list endpoints: < 300 ms

## 7. CI Execution Plan

Two GitHub Actions workflows drive test automation:

### 7.1 `test.yml` — Pull Request & Push to main

File: [.github/workflows/test.yml](../.github/workflows/test.yml)

Triggers:
- `pull_request` targeting `main` / `master`
- `push` to `main` / `master`
- `workflow_dispatch`

Jobs:

| Job | Trigger | What it runs |
|---|---|---|
| `unit` | PR + push | `go test -race -timeout 5m ./...` |
| `functional` | PR + push, needs unit | Bruno smoke collection against `--env ci` |
| `performance` | push + dispatch only | k6 smoke — 2 VUs × 20 s per script |

Server lifecycle per job:
1. Build binary: `go build -o conductor ./cmd/server`
2. Start in background with `SECRET_KEY`, `DATA_DIR=/tmp/conductor-ci`
3. Health-check poll max 30 s (`/healthz`)
4. Run tests
5. Kill server in `if: always()` step

Artifacts produced:
- `bruno-results/` — `bruno.json` + `bruno.log`

### 7.2 `test-nightly.yml` — Nightly Regression

File: [.github/workflows/test-nightly.yml](../.github/workflows/test-nightly.yml)

Trigger: daily cron `0 2 * * *` (02:00 UTC) + `workflow_dispatch` with `k6_profile` input (`baseline` | `stress`).

Jobs:

| Job | What it runs |
|---|---|
| `unit` | Same unit suite, 10 min timeout |
| `functional-full` | All Bruno files under `tests/functional/` |
| `performance-full` | Health + summary k6 scripts with profile VUs/duration |

k6 profiles:

| Profile | VUs | Duration |
|---|---|---|
| baseline (default) | 20 | 1 min per script |
| stress | 100 | 3 min per script |

Artifacts produced:
- `bruno-nightly-results/`
- `k6-nightly-results/` — `k6-health.json`, `k6-summary.json`

### 7.3 Pre-release gate

Run via `workflow_dispatch` on `test-nightly.yml` with `k6_profile=baseline` against the release candidate branch before any tag is pushed.

## 8. Traceability To Roadmap

- Use feature groups from [FEATURES.md](../FEATURES.md).
- For each new completed feature, add at least one of:
  - Unit test case
  - Bruno request/assertion
  - k6 script update (only if on a hot path)

Rule of thumb:
- Pure logic: unit test.
- User/API contract: functional test.
- Throughput/latency risk: k6 test.

## 9. Automated Testing — Feature Coverage Table

Key:
- ✅ Exists — test file(s) already written
- 📋 Planned — in the automation backlog, not yet written
- ⬜ Skip — intentionally not automated (manual / not worth the cost)

### 9.1 Platform / Auth / Security

| Feature | Unit (Go) | Functional (Bruno) | k6 |
|---|---|---|---|
| Health endpoint | ⬜ | ✅ `get-healthz.bru` | ✅ `get-healthz.k6` |
| Config: secret-key derivation | ✅ `config_test.go` | ⬜ | ⬜ |
| Auth: session token sign/verify | ✅ `auth_token_test.go` | ⬜ | ⬜ |
| Auth: session expiry enforcement | ✅ `auth_token_test.go` | 📋 `auth/session-expiry.bru` | ⬜ |
| Auth: login success / bad password | ⬜ | 📋 `auth/login.bru` | ⬜ |
| Auth: logout clears session | ⬜ | 📋 `auth/logout.bru` | ⬜ |
| Auth: API key create / use / delete | ✅ `api_key.go` model | 📋 `auth/api-key.bru` | ⬜ |
| Auth: protected route rejects no-auth | ⬜ | 📋 `auth/protected-no-auth.bru` | ⬜ |
| Auth: admin-only route denies normal user | ⬜ | 📋 `auth/admin-required.bru` | ⬜ |
| CSRF: rejects mutating request without token | ⬜ | 📋 `auth/csrf-reject.bru` | ⬜ |
| Rate limiter: window reset logic | ✅ (rate_limit.go stale cleanup) | ⬜ | ⬜ |

### 9.2 Monitor CRUD & Checker Logic

| Feature | Unit (Go) | Functional (Bruno) | k6 |
|---|---|---|---|
| Monitor create / list / edit / delete | ⬜ | 📋 `monitors/crud.bru` | 📋 monitor list endpoint |
| Monitor import / export JSON | ⬜ | 📋 `monitors/import-export.bru` | ⬜ |
| HTTP checker: status code / keyword / JSON path | ✅ `checker_http_test.go` | ⬜ | ⬜ |
| HTTP checker: TLS cert expiry | ✅ `checker_response_test.go` | ⬜ | ⬜ |
| DNS checker | ✅ `checker_test.go` | ⬜ | ⬜ |
| Push / heartbeat endpoint | ⬜ | 📋 `monitors/push-heartbeat.bru` | ⬜ |
| SMTP checker | ✅ `checker_smtp_test.go` | ⬜ | ⬜ |
| Steam / GameDig checker | ✅ `checker_steam_test.go` | ⬜ | ⬜ |
| Niche protocol checkers (SIP, Radius, SNMP, Kafka, gRPC, MQTT) | ✅ per-checker unit tests | ⬜ | ⬜ |

### 9.3 Notifications

| Feature | Unit (Go) | Functional (Bruno) | k6 |
|---|---|---|---|
| All provider payload / required-field validation | ✅ `*_test.go` in `notifier/` | ⬜ | ⬜ |
| Notification create / list / edit / delete | ⬜ | 📋 `notifications/crud.bru` | ⬜ |
| Notification test-send endpoint | ⬜ | 📋 `notifications/test-send.bru` | ⬜ |
| Notification log list | ⬜ | 📋 `notifications/logs.bru` | ⬜ |
| Notification log count (badge API) | ✅ `CountSince` in `store.go` | 📋 `notifications/log-count.bru` | 📋 log-count.k6 |

### 9.4 Status Pages, Summary & Chart

| Feature | Unit (Go) | Functional (Bruno) | k6 |
|---|---|---|---|
| Status page create / list / edit / delete | ⬜ | 📋 `status-pages/crud.bru` | ⬜ |
| Public status page HTML (unauthenticated) | ⬜ | 📋 `status-pages/public-view.bru` | ⬜ |
| Public summary JSON contract | ⬜ | ✅ `get-status-summary.bru` | ✅ `get-status-summary.k6` |
| Chart data endpoint (authenticated) | ⬜ | 📋 `status-pages/chart-auth.bru` | 📋 chart-auth.k6 |
| Chart data endpoint (public, cached) | ⬜ | 📋 `status-pages/chart-public.bru` | 📋 chart-public.k6 |
| Downtime event tracking logic | ✅ `downtime.go` model | ⬜ | ⬜ |

### 9.5 Maintenance Windows

| Feature | Unit (Go) | Functional (Bruno) | k6 |
|---|---|---|---|
| Maintenance window create / list / edit / delete | ⬜ | 📋 `maintenance/crud.bru` | ⬜ |
| Monitor linked to window suppresses alerts | ⬜ | 📋 `maintenance/suppress-alert.bru` | ⬜ |

### 9.6 Admin / User Management / Registration

| Feature | Unit (Go) | Functional (Bruno) | k6 |
|---|---|---|---|
| User list / create / delete / change password | ⬜ | 📋 `admin-users/crud.bru` | ⬜ |
| Disable / enable user account | ⬜ | 📋 `admin-users/disable-enable.bru` | ⬜ |
| Admin generate password-reset link | ⬜ | 📋 `admin-users/reset-link.bru` | ⬜ |
| Open registration flow | ⬜ | 📋 `admin-users/register.bru` | ⬜ |
| Invite-token registration | ⬜ | 📋 `admin-users/invite.bru` | ⬜ |
| Role promote / demote | ⬜ | 📋 `admin-users/role.bru` | ⬜ |

### 9.7 Tags & Proxies

| Feature | Unit (Go) | Functional (Bruno) | k6 |
|---|---|---|---|
| Tag CRUD | ⬜ | 📋 `tags/crud.bru` | ⬜ |
| Proxy CRUD | ⬜ | 📋 `proxies/crud.bru` | ⬜ |

---

## 10. Automation Backlog — Priority Order

**Priority 0 — immediate (unblock CI smoke coverage):**
- Add `auth/login.bru` + `auth/protected-no-auth.bru`
- Add `notifications/log-count.bru`
- Add k6 script for authenticated monitor-list endpoint

**Priority 1 — core workflow confidence:**
- `monitors/crud.bru` — full CRUD + import/export
- `notifications/crud.bru` + `notifications/test-send.bru`
- `status-pages/crud.bru` + `status-pages/chart-public.bru`
- k6 `chart-public.k6`

**Priority 2 — admin / edge-case coverage:**
- Full `admin-users/` folder (CRUD, disable, invite, reset-link, role)
- `maintenance/crud.bru` + `maintenance/suppress-alert.bru`
- `auth/csrf-reject.bru` + `auth/api-key.bru`

**Priority 3 — supplementary / nightly only:**
- `tags/crud.bru`, `proxies/crud.bru`
- k6 stress scripts for chart and summary endpoints
