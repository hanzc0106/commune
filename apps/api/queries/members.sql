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

-- name: ListMembers :many
SELECT id, name, role, active, created_at, updated_at
FROM members
ORDER BY active DESC, lower(name);

-- name: DisableMember :one
UPDATE members
SET active = FALSE, updated_at = now()
WHERE id = $1
RETURNING id, name, pin_hash, role, active, created_at, updated_at;

-- name: UpdateMemberPIN :one
UPDATE members
SET pin_hash = $2, updated_at = now()
WHERE id = $1
RETURNING id, name, pin_hash, role, active, created_at, updated_at;

-- name: CountActiveAdmins :one
SELECT count(*)::bigint
FROM members
WHERE active = TRUE AND role = 'admin';
