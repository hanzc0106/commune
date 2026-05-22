[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

$root = Split-Path -Parent $PSScriptRoot
Push-Location "$root\apps\api"
try {
    go run .\cmd\migrate
} finally {
    Pop-Location
}
