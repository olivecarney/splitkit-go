-- name: CreateGroup :one
INSERT INTO groups (id, name, created_by)
VALUES ($1, $2, $3)
RETURNING id, name, created_by, created_at;

-- name: ListGroups :many
SELECT id, name, created_by, created_at
FROM groups
ORDER BY created_at DESC;

-- name: GetGroup :one
SELECT id, name, created_by, created_at
FROM groups
WHERE id = $1;
