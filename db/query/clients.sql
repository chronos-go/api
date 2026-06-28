-- name: CreateClient :one
INSERT INTO clients (name, email, birth_date, password)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetClientByID :one
SELECT * FROM clients
WHERE id = $1;

-- name: GetClientByEmail :one
SELECT * FROM clients
WHERE email = $1;

-- name: ListClients :many
SELECT * FROM clients
ORDER BY created_at DESC;

-- name: UpdateClient :one
UPDATE clients
SET name = $2, email = $3, birth_date = $4, password = $5
WHERE id = $1
RETURNING *;

-- name: DeleteClient :execrows
DELETE FROM clients
WHERE id = $1;
