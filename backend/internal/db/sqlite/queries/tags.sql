-- name: GetTag :one
SELECT * FROM tags WHERE user_id = ? AND id = ?;

-- name: GetTagByName :one
SELECT * FROM tags WHERE user_id = ? AND name = ?;

-- name: ListAllTags :many
SELECT * FROM tags WHERE user_id = ? ORDER BY display_order, name;

-- name: UpsertTag :one
INSERT INTO tags (user_id, name, slug, color, description, created_at, display_order)
VALUES (?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), 0)
ON CONFLICT(user_id, name) DO UPDATE SET
    slug = excluded.slug,
    color = excluded.color,
    description = excluded.description
RETURNING *;

-- name: UpdateTag :one
UPDATE tags SET
    name = COALESCE(?, name),
    slug = COALESCE(?, slug),
    color = COALESCE(?, color),
    description = COALESCE(?, description)
WHERE user_id = ? AND id = ?
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE user_id = ? AND id = ?;

-- name: UpdateTagDisplayOrder :exec
UPDATE tags SET display_order = ? WHERE user_id = ? AND id = ?;

-- name: ListTagsForEntity :many
SELECT t.* FROM tags t
JOIN tag_assignments ta ON t.id = ta.tag_id
WHERE t.user_id = ? AND ta.entity_type = ? AND ta.entity_id = ?
ORDER BY t.display_order, t.name;

-- name: AssignTagToEntity :one
INSERT INTO tag_assignments (user_id, tag_id, entity_type, entity_id, created_at)
VALUES (?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
ON CONFLICT(user_id, tag_id, entity_type, entity_id) DO UPDATE SET entity_type = excluded.entity_type
RETURNING *;

-- name: RemoveTagAssignment :exec
DELETE FROM tag_assignments 
WHERE user_id = ? AND tag_id = ? AND entity_type = ? AND entity_id = ?;

-- name: GetNotificationTagIDs :many
SELECT tag_id FROM tag_assignments WHERE user_id = ? AND entity_type = 'notification' AND entity_id = ?;
