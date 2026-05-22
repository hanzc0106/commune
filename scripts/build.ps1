[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    pnpm --dir apps/web build

    New-Item -ItemType Directory -Force -Path "$root\dist" | Out-Null

    Push-Location "$root\apps\api"
    try {
        go test ./...
        go build -o "$root\dist\commune-server.exe" .\cmd\server
    } finally {
        Pop-Location
    }

    Write-Host "Build complete: dist\commune-server.exe"
} finally {
    Pop-Location
}
