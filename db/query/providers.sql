-- name: CreateProvider :one
INSERT INTO providers (name, email, document, password)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetProviderByID :one
SELECT * FROM providers
WHERE id = $1;

-- name: GetProviderByEmail :one
SELECT * FROM providers
WHERE email = $1;

-- name: ListProviders :many
SELECT * FROM providers
ORDER BY created_at DESC;
