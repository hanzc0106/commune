# Commune Dev Tooling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the local development workflow from ad hoc PowerShell windows to a repeatable developer toolchain with a unified task runner, Go hot reload, structured server logging, and graceful shutdown.

**Architecture:** Keep PowerShell scripts as Windows-friendly wrappers, but introduce `Taskfile.yml` as the canonical command surface. Use Air for API hot reload in development, keep Docker Compose responsible for PostgreSQL, and update the Go server entry point to use `log/slog` plus signal-aware graceful shutdown.

**Tech Stack:** Go `log/slog`, Chi, pgx, Air, Taskfile, Docker Compose, pnpm, PowerShell.

---

## Scope

This plan implements:

- `Taskfile.yml` with canonical `task dev`, `task api`, `task web`, `task db`, `task migrate`, `task test`, `task build`, and `task reset-db` commands.
- `.air.toml` for Go API hot reload.
- PowerShell wrappers that delegate to Task where appropriate.
- Go server graceful shutdown.
- Go server structured logging with `slog`.
- README updates for the new workflow.

This plan does not implement:

- Production Docker image.
- CI pipeline.
- Cross-platform shell wrappers beyond Taskfile.
- Process supervisor for production deployment.

## Target Files

Create:

- `Taskfile.yml`
- `.air.toml`

Modify:

- `apps/api/cmd/server/main.go`
- `scripts/start-dev.ps1`
- `scripts/dev.ps1`
- `README.md`

## Task 1: Add Taskfile Command Surface

- [ ] Create `Taskfile.yml` with these tasks:

```yaml
version: "3"

tasks:
  db:
    desc: Start local PostgreSQL
    cmds:
      - docker compose --env-file .env.example -f deploy/compose.yml up -d db

  migrate:
    desc: Run database migrations
    dir: apps/api
    cmds:
      - go run ./cmd/migrate

  api:
    desc: Run API with hot reload
    dir: apps/api
    cmds:
      - air

  api:run:
    desc: Run API without hot reload
    dir: apps/api
    cmds:
      - go run ./cmd/server

  web:
    desc: Run Vite dev server
    dir: apps/web
    cmds:
      - pnpm dev

  dev:
    desc: Start database and run migrations, then print dev server commands
    deps: [db, migrate]
    cmds:
      - task --list

  test:
    desc: Run backend tests and frontend type/build checks
    cmds:
      - cmd: go test ./...
        dir: apps/api
      - pnpm --dir apps/web build

  build:
    desc: Build frontend and API binary
    cmds:
      - pnpm --dir apps/web build
      - powershell -NoProfile -ExecutionPolicy Bypass -File scripts/build.ps1

  reset-db:
    desc: Reset local auth development data
    cmds:
      - docker exec commune-db psql -U commune -d commune -c "TRUNCATE sessions, members, app_settings RESTART IDENTITY CASCADE;"
```

- [ ] Run `task --list`.
- [ ] Expected: task list includes `api`, `web`, `dev`, `test`, and `build`.
- [ ] Commit with `chore: add taskfile dev commands`.

## Task 2: Add Air Hot Reload

- [ ] Create `.air.toml`:

```toml
root = "apps/api"
tmp_dir = "../../tmp/air"

[build]
cmd = "go build -o ../../tmp/air/commune-api.exe ./cmd/server"
bin = "../../tmp/air/commune-api.exe"
include_ext = ["go", "sql"]
exclude_dir = ["tmp", "vendor", "internal/db/queries"]
delay = 1000
stop_on_error = true

[log]
time = true

[screen]
clear_on_rebuild = false
```

- [ ] Run `air -v`.
- [ ] Run `task api` briefly and verify it starts the API on `:8090`.
- [ ] Stop the Air process.
- [ ] Commit with `chore: add api hot reload config`.

## Task 3: Add Structured Logging and Graceful Shutdown

- [ ] Modify `apps/api/cmd/server/main.go` to:
  - use `log/slog`;
  - create `http.Server`;
  - listen for `os.Interrupt` and `syscall.SIGTERM`;
  - call `server.Shutdown` with a 10 second timeout;
  - log `server_starting`, `server_stopped`, and fatal startup errors with structured fields.

- [ ] Run:

```powershell
Set-Location apps\api
go test ./...
go build .\cmd\server
```

- [ ] Expected: tests and build pass.
- [ ] Commit with `feat: add graceful api shutdown`.

## Task 4: Update Windows Wrappers and Docs

- [ ] Modify `scripts/start-dev.ps1` to:
  - run `task db`;
  - run `task migrate`;
  - start one PowerShell window with `task api`;
  - start one PowerShell window with `task web`;
  - open `http://localhost:5173`.

- [ ] Keep `scripts/dev.ps1` delegating to `start-dev.ps1`.
- [ ] Update README to make `task dev`, `task api`, and `task web` the preferred commands.
- [ ] Run `.\scripts\build.ps1`.
- [ ] Run `task test`.
- [ ] Commit with `docs: document task based dev workflow`.

## Final Verification

- [ ] Run `task test`.
- [ ] Run `.\scripts\build.ps1`.
- [ ] Run `git status --short --branch`.
- [ ] Push `feature/commune-auth-foundation`.

## Self-Review

This plan upgrades development tooling only. It does not change product behavior, database shape, or frontend flows. The plan keeps Windows convenience while adding a more standard command surface for future contributors and CI.
