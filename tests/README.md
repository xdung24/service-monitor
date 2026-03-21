# Tests

This directory contains functional tests (Bruno) and performance tests (k6).

```
tests/
  functional-tests/     Bruno collection — API correctness checks
  performance-tests/    k6 scripts — load and performance tests
```

---

## Prerequisites

### Bruno CLI (`bru`)

```bash
# npm
npm install -g @usebruno/cli

# Homebrew (macOS/Linux)
brew install bruno
```

Verify: `bru --version`

### k6

```bash
# macOS/Linux — Homebrew
brew install k6

# Windows — winget
winget install k6 --source winget

# Windows — Chocolatey
choco install k6

# Docker (no install needed)
docker pull grafana/k6
```

Verify: `k6 version`

---

## Configuration

### Functional tests — Bruno environment

Environments live in `functional-tests/environments/`. The default `local.bru` targets `http://localhost:3001`.

To override, edit `functional-tests/environments/local.bru`:

```
vars {
  baseUrl: http://localhost:3001
}
```

Or add a new environment file (e.g. `staging.bru`) with the same structure and pass `--env staging` when running.

### Performance tests — k6 environment

k6 reads variables from the OS environment via `__ENV`. Pass them with the `-e` flag:

```bash
k6 run -e BASE_URL=http://localhost:3001 performance-tests/get-healthz.k6
```

To load from a `.env` file first:

```bash
# Linux/macOS
set -a && source .env && set +a && k6 run performance-tests/get-healthz.k6

# PowerShell (Windows)
Get-Content .env | ForEach-Object { $k,$v = $_ -split '=',2; [System.Environment]::SetEnvironmentVariable($k,$v) }
k6 run performance-tests/get-healthz.k6
```

---

## Running tests

Start the server before running any tests:

```bash
go run ./cmd/server
# or
docker compose up -d
```

### 1. Functional tests (Bruno)

```bash
# Run all requests in the collection against the local environment
bru run --env local

# Run a single request file
bru run get-healthz.bru --env local
```

A passing run exits with code `0`. Any failed assertion exits with a non-zero code.

### 2. Performance tests (k6)

```bash
# Run the healthz load test
k6 run -e BASE_URL=http://localhost:3001 get-healthz.k6
```

The script ramps up to 10 virtual users over 10 s, holds for 30 s, then ramps down.
It fails if the error rate exceeds 1% or the p95 latency exceeds 200 ms.
