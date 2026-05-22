-- name: CreateSession :one
INSERT INTO sessions (member_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, member_id, token_hash, expires_at, created_at;

-- name: GetSessionByTokenHash :one
SELECT
    sessions.id,
    sessions.member_id,
    sessions.token_hash,
    sessions.expires_at,
    sessions.created_at,
    members.name AS member_name,
    members.role AS member_role,
    members.active AS member_active
FROM sessions
JOIN members ON members.id = sessions.member_id
WHERE sessions.token_hash = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= now();
