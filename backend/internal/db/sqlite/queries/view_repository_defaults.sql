-- name: ListViewRepositoryDefaults :many
SELECT * FROM view_repository_defaults WHERE user_id = ?;

-- name: GetViewRepositoryDefault :one
SELECT * FROM view_repository_defaults WHERE user_id = ? AND view_key = ?;

-- name: UpsertViewRepositoryDefault :exec
INSERT INTO view_repository_defaults (user_id, view_key, repository_ids, updated_at)
VALUES (?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
ON CONFLICT(user_id, view_key) DO UPDATE SET
    repository_ids = excluded.repository_ids,
    updated_at = excluded.updated_at;

-- name: DeleteViewRepositoryDefault :exec
DELETE FROM view_repository_defaults WHERE user_id = ? AND view_key = ?;
