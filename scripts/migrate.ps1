[Console]::InputEncoding = [Text.Encoding]::UTF8
[Console]::OutputEncoding = [Text.Encoding]::UTF8
chcp 65001 > $null

Write-Host "Migration runner is not implemented in the foundation plan."
Write-Host "Current migration files live in apps\api\migrations."
Write-Host "The auth and ledger plan will choose and wire the migration tool."
