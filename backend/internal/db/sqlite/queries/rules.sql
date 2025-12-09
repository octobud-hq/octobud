-- name: GetRule :one
SELECT * FROM rules WHERE user_id = ? AND id = ?;

-- name: ListRules :many
SELECT * FROM rules WHERE user_id = ? ORDER BY display_order, name;

-- name: ListEnabledRulesOrdered :many
SELECT * FROM rules WHERE user_id = ? AND enabled = 1 ORDER BY display_order;

-- name: GetRulesByViewID :many
SELECT * FROM rules WHERE user_id = ? AND view_id = ? ORDER BY display_order;

-- name: CreateRule :one
INSERT INTO rules (user_id, name, description, query, view_id, enabled, actions, display_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
RETURNING *;

-- name: UpdateRule :one
UPDATE rules SET
    name = COALESCE(?, name),
    description = COALESCE(?, description),
    query = COALESCE(?, query),
    view_id = ?,
    enabled = COALESCE(?, enabled),
    actions = COALESCE(?, actions),
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE user_id = ? AND id = ?
RETURNING *;

-- name: DeleteRule :exec
DELETE FROM rules WHERE user_id = ? AND id = ?;

-- name: UpdateRuleOrder :exec
UPDATE rules SET display_order = ? WHERE user_id = ? AND id = ?;
