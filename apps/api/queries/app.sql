-- name: GetAppSettings :one
SELECT id, household_name, default_currency, initialized_at, created_at, updated_at
FROM app_settings
WHERE id = TRUE;

-- name: AppSettingsExist :one
SELECT EXISTS (SELECT 1 FROM app_settings WHERE id = TRUE)::boolean AS exists;

-- name: CreateAppSettings :one
INSERT INTO app_settings (id, household_name, default_currency)
VALUES (TRUE, $1, 'CNY')
RETURNING id, household_name, default_currency, initialized_at, created_at, updated_at;
