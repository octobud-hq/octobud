-- name: GetUser :one
SELECT * FROM users WHERE id = 1;

-- name: CreateUser :one
-- Creates the single user record (id is always 1)
INSERT INTO users (id, created_at, updated_at)
VALUES (1, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
RETURNING *;

-- name: UpdateUserGitHubIdentity :one
-- Updates the GitHub identity (user ID and username from GitHub)
UPDATE users SET 
    github_user_id = ?, 
    github_username = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') 
WHERE id = 1 RETURNING *;

-- name: UpdateUserSyncSettings :one
UPDATE users SET sync_settings = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1 RETURNING *;

-- name: UpdateUserGitHubToken :one
UPDATE users SET github_token_encrypted = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1 RETURNING *;

-- name: ClearUserGitHubToken :one
UPDATE users SET 
    github_token_encrypted = NULL, 
    github_user_id = NULL,
    github_username = NULL,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') 
WHERE id = 1 RETURNING *;

-- name: UpdateUserRetentionSettings :one
UPDATE users SET retention_settings = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1 RETURNING *;

-- name: UpdateUserMutedUntil :one
UPDATE users SET muted_until = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1 RETURNING *;

-- name: UpdateUserUpdateSettings :one
UPDATE users SET update_settings = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1 RETURNING *;
