-- name: CreateAuthSession :one
INSERT INTO auth_sessions (id, user_id, role, email, family_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetAuthSessionForUpdate :one
SELECT * FROM auth_sessions
WHERE token_hash = $1
FOR UPDATE;

-- name: GetAuthSessionByHash :one
SELECT * FROM auth_sessions
WHERE token_hash = $1;

-- name: MarkAuthSessionUsed :execrows
UPDATE auth_sessions
SET used_at = $2
WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL;

-- name: RevokeAuthSessionFamily :exec
UPDATE auth_sessions
SET revoked_at = COALESCE(revoked_at, $2)
WHERE family_id = $1;

-- name: RevokeAuthSessionByHash :execrows
UPDATE auth_sessions
SET revoked_at = COALESCE(revoked_at, $2)
WHERE token_hash = $1 AND revoked_at IS NULL;
