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
