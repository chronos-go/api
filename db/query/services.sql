-- name: CreateService :one
INSERT INTO services (provider_id, name, description, price_cents, duration_minutes)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetServiceByID :one
SELECT * FROM services
WHERE id = $1;

-- name: ListServices :many
SELECT * FROM services
ORDER BY created_at DESC;

-- name: ListServicesByProviderID :many
SELECT * FROM services
WHERE provider_id = $1
ORDER BY created_at DESC;

-- name: UpdateService :one
UPDATE services
SET name = $2, description = $3, price_cents = $4, duration_minutes = $5
WHERE id = $1
RETURNING *;

-- name: DeleteService :exec
DELETE FROM services
WHERE id = $1;
