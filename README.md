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

Run the web app:

```powershell
Set-Location apps\web
pnpm install
pnpm dev
```
