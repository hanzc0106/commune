[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

pnpm --dir apps/web build

New-Item -ItemType Directory -Force -Path "$root\dist" | Out-Null

Set-Location "$root\apps\api"
go test ./...
go build -o "$root\dist\commune-server.exe" .\cmd\server

Write-Host "Build complete: dist\commune-server.exe"
