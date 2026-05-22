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

Start PostgreSQL:

```powershell
.\scripts\db-up.ps1
```

Run the API:

```powershell
Set-Location apps\api
go run .\cmd\server
```

The API listens on `http://localhost:8090` by default.

Run the web app:

```powershell
Set-Location apps\web
pnpm install
pnpm dev
```

## Scripts

Start local database:

```powershell
.\scripts\db-up.ps1
```

Show development commands:

```powershell
.\scripts\dev.ps1
```

Generate sqlc code:

```powershell
.\scripts\sqlc.ps1
```

Build frontend and API:

```powershell
.\scripts\build.ps1
```
