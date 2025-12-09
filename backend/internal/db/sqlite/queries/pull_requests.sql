-- name: UpsertPullRequest :one
INSERT INTO pull_requests (
    user_id, repository_id, github_id, node_id, number, title, state,
    draft, merged, author_login, author_id,
    created_at, updated_at, closed_at, merged_at, raw
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, repository_id, number) DO UPDATE SET
    github_id = excluded.github_id,
    node_id = excluded.node_id,
    title = excluded.title,
    state = excluded.state,
    draft = excluded.draft,
    merged = excluded.merged,
    author_login = excluded.author_login,
    author_id = excluded.author_id,
    updated_at = excluded.updated_at,
    closed_at = excluded.closed_at,
    merged_at = excluded.merged_at,
    raw = excluded.raw
RETURNING *;

-- name: DeleteOrphanedPullRequests :execrows
-- Delete pull_requests that have no associated notifications for a user
-- Note: user_id passed twice - once for outer query, once for subquery
DELETE FROM pull_requests
WHERE pull_requests.user_id = ?
  AND pull_requests.id NOT IN (
    SELECT DISTINCT notifications.pull_request_id 
    FROM notifications
    WHERE notifications.user_id = ?
      AND notifications.pull_request_id IS NOT NULL
);
