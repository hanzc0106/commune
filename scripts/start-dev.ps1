[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot
$apiDir = Join-Path $root "apps\api"
$webDir = Join-Path $root "apps\web"

Push-Location $root
try {
    docker compose --env-file .env.example -f deploy/compose.yml up -d db
} finally {
    Pop-Location
}

Push-Location $apiDir
try {
    go run .\cmd\migrate
} finally {
    Pop-Location
}

$apiCommand = @"
[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > `$null
Set-Location "$apiDir"
go run .\cmd\server
"@

$webCommand = @"
[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > `$null
Set-Location "$webDir"
pnpm dev
"@

Start-Process powershell -ArgumentList @("-NoExit", "-Command", $apiCommand) -WindowStyle Normal
Start-Process powershell -ArgumentList @("-NoExit", "-Command", $webCommand) -WindowStyle Normal

Start-Sleep -Seconds 2
Start-Process "http://localhost:5173"

Write-Host "Commune development environment is starting."
Write-Host "API: http://localhost:8090"
Write-Host "Web: http://localhost:5173"
Write-Host "Close the API and Web PowerShell windows to stop the dev servers."
