-- name: GetNotificationByGithubID :one
SELECT * FROM notifications WHERE user_id = ? AND github_id = ?;

-- name: GetNotificationByID :one
SELECT * FROM notifications WHERE user_id = ? AND id = ?;

-- name: MarkNotificationRead :one
UPDATE notifications SET is_read = 1 WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: MarkNotificationUnread :one
UPDATE notifications SET is_read = 0 WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: ArchiveNotification :one
UPDATE notifications 
SET archived = 1,
    snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: UnarchiveNotification :one
UPDATE notifications SET archived = 0 WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: MuteNotification :one
UPDATE notifications 
SET muted = 1,
    snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: UnmuteNotification :one
UPDATE notifications SET muted = 0 WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: SnoozeNotification :one
UPDATE notifications 
SET snoozed_until = ?,
    snoozed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    effective_sort_date = ?
WHERE user_id = ? AND github_id = ? 
RETURNING *;

-- name: UnsnoozeNotification :one
UPDATE notifications 
SET snoozed_until = NULL,
    snoozed_at = NULL,
    effective_sort_date = COALESCE(github_updated_at, imported_at)
WHERE user_id = ? AND github_id = ? 
RETURNING *;

-- name: StarNotification :one
UPDATE notifications SET starred = 1 WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: UnstarNotification :one
UPDATE notifications SET starred = 0 WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: MarkNotificationFiltered :one
UPDATE notifications SET filtered = 1 WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: MarkNotificationUnfiltered :one
UPDATE notifications SET filtered = 0 WHERE user_id = ? AND github_id = ? RETURNING *;

-- name: UpsertNotification :one
INSERT INTO notifications (
    user_id, github_id, repository_id, pull_request_id, subject_type, subject_title, subject_url,
    subject_latest_comment_url, reason, github_unread, github_updated_at,
    github_last_read_at, github_url, github_subscription_url, payload,
    subject_raw, subject_fetched_at, author_login, author_id,
    subject_number, subject_state, subject_merged, subject_state_reason,
    imported_at, effective_sort_date
) VALUES (
    sqlc.arg(user_id),
    sqlc.arg(github_id), 
    sqlc.arg(repository_id), 
    sqlc.narg(pull_request_id),
    sqlc.arg(subject_type), 
    sqlc.arg(subject_title), 
    sqlc.arg(subject_url),
    sqlc.arg(subject_latest_comment_url), 
    sqlc.arg(reason), 
    sqlc.arg(github_unread), 
    sqlc.arg(github_updated_at),
    sqlc.arg(github_last_read_at), 
    sqlc.arg(github_url), 
    sqlc.arg(github_subscription_url), 
    sqlc.arg(payload),
    sqlc.narg(subject_raw),
    sqlc.narg(subject_fetched_at),
    sqlc.narg(author_login),
    sqlc.narg(author_id),
    sqlc.narg(subject_number),
    sqlc.narg(subject_state),
    sqlc.narg(subject_merged),
    sqlc.narg(subject_state_reason),
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), 
    COALESCE(sqlc.arg(effective_sort_date), strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
)
ON CONFLICT(user_id, github_id) DO UPDATE SET
    pull_request_id = excluded.pull_request_id,
    subject_title = excluded.subject_title,
    subject_url = excluded.subject_url,
    subject_latest_comment_url = excluded.subject_latest_comment_url,
    reason = excluded.reason,
    github_unread = excluded.github_unread,
    github_updated_at = excluded.github_updated_at,
    github_last_read_at = excluded.github_last_read_at,
    github_url = excluded.github_url,
    github_subscription_url = excluded.github_subscription_url,
    payload = excluded.payload,
    subject_raw = excluded.subject_raw,
    subject_fetched_at = excluded.subject_fetched_at,
    author_login = excluded.author_login,
    author_id = excluded.author_id,
    subject_number = excluded.subject_number,
    subject_state = excluded.subject_state,
    subject_merged = excluded.subject_merged,
    subject_state_reason = excluded.subject_state_reason,
    -- Preserve snoozed_until as sort date if notification is snoozed, otherwise use new github_updated_at
    effective_sort_date = COALESCE(notifications.snoozed_until, excluded.effective_sort_date)
RETURNING *;

-- name: UpdateNotificationSubject :exec
UPDATE notifications SET
    subject_raw = ?,
    subject_fetched_at = ?,
    pull_request_id = ?,
    subject_number = ?,
    subject_state = ?,
    subject_merged = ?,
    subject_state_reason = ?
WHERE user_id = ? AND github_id = ?;

-- name: ResetNotificationStatusOnSync :exec
-- Used by UpsertNotification to implement smart status updates.
-- When new activity is detected (github_updated_at changed) and notification is not muted,
-- reset archived and is_read to bring the notification back to inbox.
UPDATE notifications 
SET archived = 0, is_read = 0 
WHERE user_id = ? AND github_id = ?;

-- name: GetStorageStats :one
-- Get notification counts by state for storage management UI
SELECT
    COUNT(*) as total_count,
    COUNT(CASE WHEN archived = 1 THEN 1 END) as archived_count,
    COUNT(CASE WHEN starred = 1 THEN 1 END) as starred_count,
    COUNT(CASE WHEN snoozed_until IS NOT NULL THEN 1 END) as snoozed_count,
    COUNT(CASE WHEN is_read = 0 THEN 1 END) as unread_count,
    (SELECT COUNT(DISTINCT ta.entity_id) FROM tag_assignments ta WHERE ta.user_id = ? AND ta.entity_type = 'notification') as tagged_count
FROM notifications n
WHERE n.user_id = ?;

-- name: CountEligibleForCleanup :one
-- Count notifications eligible for cleanup based on retention settings
-- Eligible: (archived OR muted), not starred (if protected), not tagged (if protected)
-- Uses effective_sort_date for cutoff (consistent with UI sorting, handles snoozed items naturally)
SELECT COUNT(*) as count
FROM notifications n
WHERE n.user_id = ?
  AND (n.archived = 1 OR n.muted = 1)
  AND (sqlc.arg(protect_starred) = 0 OR n.starred = 0)
  AND (sqlc.arg(protect_tagged) = 0 OR n.id NOT IN (
      SELECT ta.entity_id FROM tag_assignments ta WHERE ta.user_id = n.user_id AND ta.entity_type = 'notification'
  ))
  AND COALESCE(n.effective_sort_date, n.github_updated_at, n.imported_at) < sqlc.arg(cutoff_date);

-- name: DeleteOldArchivedNotifications :execrows
-- Delete old archived/muted notifications in batches, respecting protection settings
-- Uses effective_sort_date for cutoff (consistent with UI sorting, handles snoozed items naturally)
-- Returns number of deleted rows
DELETE FROM notifications
WHERE id IN (
    SELECT n.id FROM notifications n
    WHERE n.user_id = ?
      AND (n.archived = 1 OR n.muted = 1)
      AND (sqlc.arg(protect_starred) = 0 OR n.starred = 0)
      AND (sqlc.arg(protect_tagged) = 0 OR n.id NOT IN (
          SELECT ta.entity_id FROM tag_assignments ta WHERE ta.user_id = n.user_id AND ta.entity_type = 'notification'
      ))
      AND COALESCE(n.effective_sort_date, n.github_updated_at, n.imported_at) < sqlc.arg(cutoff_date)
    ORDER BY COALESCE(n.effective_sort_date, n.github_updated_at, n.imported_at) ASC
    LIMIT sqlc.arg(batch_size)
);
