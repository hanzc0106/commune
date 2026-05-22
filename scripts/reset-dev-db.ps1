[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot

Push-Location $root
try {
    docker compose --env-file .env.example -f deploy/compose.yml up -d db
} finally {
    Pop-Location
}

docker exec commune-db psql -U commune -d commune -c "TRUNCATE sessions, members, app_settings RESTART IDENTITY CASCADE;"

Write-Host "Development auth data has been reset."
Write-Host "Refresh http://localhost:5173 to see the initialization screen."
