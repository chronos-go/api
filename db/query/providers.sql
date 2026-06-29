-- name: CreateProvider :one
INSERT INTO providers (id, name, email, document, password)
VALUES ($1, $2, $3, $4, $5)
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

-- name: UpdateProvider :one
UPDATE providers
SET name = $2, email = $3, document = $4, password = $5
WHERE id = $1
RETURNING *;

-- name: DeleteProvider :execrows
DELETE FROM providers
WHERE id = $1;
