# Service Monitor — Feature Roadmap

This document tracks which features from Uptime Kuma are implemented, in progress, or planned.
**Update this file whenever a feature is implemented or a new one is identified.**

---

## Monitor Types

| Type | Status | Notes |
|---|---|---|
| HTTP / HTTPS | ✅ Done | Status code 2xx–3xx check |
| TCP Port | ✅ Done | Dial connect check |
| Ping | ✅ Done | TCP-proxy reachability (port 80/443) |
| DNS | ✅ Done | A/AAAA/CNAME/MX/NS/TXT/PTR; optional expected-value assertion |
| Push / Heartbeat | ⬜ Planned | `GET /push/:token` endpoint; token stored on monitor |
| HTTP — Keyword match | ⬜ Planned | Scan response body for substring |
| HTTP — JSON Query | ⬜ Planned | JSONPath expression + operator on response |
| HTTP — Accepted status codes | ⬜ Planned | Comma-separated list, e.g. `200,404` |
| WebSocket Upgrade | ⬜ Planned | Verify WS handshake succeeds |
| SMTP | ⬜ Planned | Connect + optional EHLO |
| SNMP | ⬜ Planned | Get OID value, optional assert |
| MQTT | ⬜ Planned | Subscribe, verify broker responds |
| Docker Container | ⬜ Planned | Container running/healthy via Docker socket or HTTP API |
| MySQL / MariaDB | ⬜ Planned | TCP + `SELECT 1` |
| PostgreSQL | ⬜ Planned | TCP + `SELECT 1` |
| Microsoft SQL Server | ⬜ Planned | TCP + simple query |
| MongoDB | ⬜ Planned | Ping command |
| Redis | ⬜ Planned | `PING` command |
| RabbitMQ | ⬜ Planned | Management API health check |
| gRPC Keyword | ⬜ Planned | gRPC call, keyword match on response |
| SIP Options | ⬜ Planned | SIP OPTIONS request |
| Radius | ⬜ Planned | Authentication request |
| Steam | ⬜ Planned | Steam Web API query |
| GameDig | ⬜ Planned | Game server query protocol |
| Tailscale Ping | ⬜ Planned | `tailscale ping` subprocess |
| Globalping | ⬜ Planned | Globalping API distributed check |
| Kafka Producer | ⬜ Planned | Produce a test message |
| Real Browser (Chromium) | ⬜ Planned | Headless browser via chromedp |
| System Service | ⬜ Planned | Windows SCM / systemd unit status |
| Group / Manual | ⬜ Planned | Logical grouping / manual status |

---

## HTTP Check Capabilities (extensions to HTTP type)

| Feature | Status | Notes |
|---|---|---|
| Status code check | ✅ Done | |
| Custom accepted status codes | ⬜ Planned | e.g. treat 404 as UP |
| Ignore TLS/SSL errors | ⬜ Planned | `InsecureSkipVerify` flag |
| TLS certificate expiry alert | ⬜ Planned | Alert N days before expiry |
| Basic auth | ⬜ Planned | Username + password on monitor |
| Bearer token auth | ⬜ Planned | `Authorization: Bearer …` header |
| Custom request headers | ⬜ Planned | Key-value list |
| Custom request body | ⬜ Planned | POST/PUT body + method select |
| Keyword match in body | ⬜ Planned | (see Monitor Types above) |
| JSON path query | ⬜ Planned | (see Monitor Types above) |
| Redirect follow control | ⬜ Planned | Max redirects / no-follow option |
| Proxy per-monitor | ⬜ Planned | HTTP/SOCKS5 proxy per check |

---

## Platform / Infrastructure Features

| Feature | Status | Notes |
|---|---|---|
| Uptime % (24 h / 30 d) | ✅ Done | |
| Heartbeat history page | ✅ Done | |
| Notification send history | ✅ Done | |
| Custom DNS server per monitor | ✅ Done | |
| Notification providers: Webhook | ✅ Done | |
| Notification providers: Telegram | ✅ Done | |
| Notification providers: Email (SMTP) | ✅ Done | |
| Public status page | ⬜ Planned | Read-only page showing selected monitors |
| Maintenance windows | ⬜ Planned | Suppress alerts during scheduled downtime |
| Tags / labels on monitors | ⬜ Planned | Color-coded labels, filter by tag |
| Proxy management | ⬜ Planned | Shared proxy config referenced by monitors |
| Docker host management | ⬜ Planned | Registered Docker daemons for Docker monitor |
| API keys | ⬜ Planned | Token-based API access |
| 2FA (TOTP) | ⬜ Planned | TOTP QR code + enforced on login |
| Additional notification providers | ⬜ Planned | Slack, Discord, PagerDuty, Gotify, ntfy, etc. |
| Remote browser config | ⬜ Planned | Chromium endpoint for real-browser checks |
| Cloudflare Tunnel integration | ⬜ Planned | Expose via cloudflared without open port |
| Dark/light theme toggle | ⬜ Planned | User preference stored in cookie |
| Latency sparkline charts | ⬜ Planned | Inline SVG trend on dashboard/detail |
| Multi-user support | ⬜ Planned | Per-user monitors and notification settings |
| Import / export monitors | ✅ Done | Export a single monitor's config as JSON (`GET /monitors/:id/export`); import via file upload (`POST /monitors/import`) |
| Backup / restore all config | ⬜ Planned | Export all monitors, notifications, and settings into one JSON or YAML file; restore via upload or CLI flag |

---

## Legend

| Symbol | Meaning |
|---|---|
| ✅ Done | Fully implemented and tested |
| 🚧 In Progress | Currently being worked on |
| ⬜ Planned | Identified, not yet started |
| ❌ Won't do | Out of scope for this project |
