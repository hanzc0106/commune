# Commune

Commune is a self-hosted family bookkeeping application.

## Current Status

The project is in foundation setup. The initial target stack is:

- Go API
- PostgreSQL
- React/Vite frontend
- Docker Compose

## Local Development

Copy the example environment file:

```powershell
Copy-Item .env.example .env
```

Install the local developer tools once if they are missing:

```powershell
scoop install task air
```

Start the database and run migrations:

```powershell
task dev
```

Run the API with Go hot reload:

```powershell
task api
```

Run the web app:

```powershell
task web
```

For the Windows one-click workflow:

```powershell
.\scripts\start-dev.ps1
```

This starts PostgreSQL, runs migrations, opens an API PowerShell window running `task api`, opens a Web PowerShell window running `task web`, and opens `http://localhost:5173`.

Start PostgreSQL:

```powershell
task db
```

Run the API without hot reload:

```powershell
task api:run
```

The API listens on `http://localhost:8090` by default.

## Scripts

Show all canonical development commands:

```powershell
task --list
```

Start local database:

```powershell
task db
```

Show development commands:

```powershell
.\scripts\dev.ps1
```

Start API and Web dev servers in separate PowerShell windows:

```powershell
.\scripts\start-dev.ps1
```

Reset development auth data:

```powershell
task reset-db
```

Generate sqlc code:

```powershell
.\scripts\sqlc.ps1
```

Build frontend and API:

```powershell
task build
```
