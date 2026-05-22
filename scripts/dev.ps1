[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

docker compose --env-file .env.example -f deploy/compose.yml up -d db

Write-Host "PostgreSQL is starting or already running."
Write-Host "Run the API in one terminal:"
Write-Host "  Set-Location apps\api; go run .\cmd\server"
Write-Host "Run the web app in another terminal:"
Write-Host "  Set-Location apps\web; pnpm dev"
