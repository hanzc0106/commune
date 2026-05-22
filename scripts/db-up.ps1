[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

docker compose --env-file .env.example -f deploy/compose.yml up -d db
