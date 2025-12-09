-- name: DeleteAllGitHubData :exec
-- Delete all GitHub data for a user (notifications, pull requests, repositories, sync state, tag assignments)
-- Preserves: tags, views, rules, users
DELETE FROM tag_assignments WHERE user_id = ?;
DELETE FROM notifications WHERE user_id = ?;
DELETE FROM pull_requests WHERE user_id = ?;
DELETE FROM repositories WHERE user_id = ?;
DELETE FROM sync_state WHERE user_id = ?;

