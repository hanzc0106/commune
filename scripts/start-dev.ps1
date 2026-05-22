[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot

Push-Location $root
try {
    task db
    task migrate
} finally {
    Pop-Location
}

$apiCommand = @"
[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > `$null
Set-Location "$root"
task api
"@

$webCommand = @"
[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > `$null
Set-Location "$root"
task web
"@

Start-Process powershell -ArgumentList @("-NoExit", "-Command", $apiCommand) -WindowStyle Normal
Start-Process powershell -ArgumentList @("-NoExit", "-Command", $webCommand) -WindowStyle Normal

Start-Sleep -Seconds 2
Start-Process "http://localhost:5173"

Write-Host "Commune development environment is starting."
Write-Host "API: http://localhost:8090"
Write-Host "Web: http://localhost:5173"
Write-Host "Close the API and Web PowerShell windows to stop the dev servers."
