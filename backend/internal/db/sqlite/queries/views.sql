-- name: GetView :one
SELECT * FROM views WHERE user_id = ? AND id = ?;

-- name: ListViews :many
SELECT * FROM views WHERE user_id = ? ORDER BY display_order, name;

-- name: CreateView :one
INSERT INTO views (user_id, name, slug, description, is_default, icon, query, display_order, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
RETURNING *;

-- name: UpdateView :one
UPDATE views SET
    name = COALESCE(?, name),
    slug = COALESCE(?, slug),
    description = COALESCE(?, description),
    icon = COALESCE(?, icon),
    query = COALESCE(?, query),
    is_default = COALESCE(?, is_default)
WHERE user_id = ? AND id = ?
RETURNING *;

-- name: DeleteView :execrows
DELETE FROM views WHERE user_id = ? AND id = ?;

-- name: UpdateViewOrder :exec
UPDATE views SET display_order = ? WHERE user_id = ? AND id = ?;
