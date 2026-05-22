-- name: CreateMember :one
INSERT INTO members (name, pin_hash, role, active)
VALUES ($1, $2, $3, TRUE)
RETURNING id, name, pin_hash, role, active, created_at, updated_at;

-- name: GetMemberByID :one
SELECT id, name, pin_hash, role, active, created_at, updated_at
FROM members
WHERE id = $1;

-- name: ListActiveLoginMembers :many
SELECT id, name
FROM members
WHERE active = TRUE
ORDER BY lower(name);
