-- name: GetRepositoryByID :one
SELECT * FROM repositories WHERE user_id = ? AND id = ?;

-- name: ListRepositories :many
SELECT * FROM repositories WHERE user_id = ? ORDER BY full_name;

-- name: UpsertRepository :one
INSERT INTO repositories (
    user_id, github_id, node_id, name, full_name, owner_login, owner_id,
    private, description, html_url, fork, visibility, default_branch,
    archived, disabled, pushed_at, created_at, updated_at, raw,
    owner_avatar_url, owner_html_url
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, full_name) DO UPDATE SET
    github_id = excluded.github_id,
    node_id = excluded.node_id,
    name = excluded.name,
    owner_login = excluded.owner_login,
    owner_id = excluded.owner_id,
    private = excluded.private,
    description = excluded.description,
    html_url = excluded.html_url,
    fork = excluded.fork,
    visibility = excluded.visibility,
    default_branch = excluded.default_branch,
    archived = excluded.archived,
    disabled = excluded.disabled,
    pushed_at = excluded.pushed_at,
    updated_at = excluded.updated_at,
    raw = excluded.raw,
    owner_avatar_url = excluded.owner_avatar_url,
    owner_html_url = excluded.owner_html_url
RETURNING *;
