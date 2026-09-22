-- name: GetSyncState :one
SELECT * FROM sync_state WHERE user_id = ?;

-- name: UpsertSyncState :one
INSERT INTO sync_state (
    user_id, last_successful_poll, last_notification_etag, 
    latest_notification_at, initial_sync_completed_at, oldest_notification_synced_at,
    created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
)
ON CONFLICT(user_id) DO UPDATE SET
    last_successful_poll = COALESCE(excluded.last_successful_poll, sync_state.last_successful_poll),
    last_notification_etag = COALESCE(excluded.last_notification_etag, sync_state.last_notification_etag),
    latest_notification_at = COALESCE(excluded.latest_notification_at, sync_state.latest_notification_at),
    initial_sync_completed_at = COALESCE(excluded.initial_sync_completed_at, sync_state.initial_sync_completed_at),
    -- The oldest-synced marker only ever moves backwards: a sync window that ends later
    -- than the current oldest (e.g. a re-sync of the last 30 days) must not advance it.
    oldest_notification_synced_at = CASE
        WHEN excluded.oldest_notification_synced_at IS NULL THEN sync_state.oldest_notification_synced_at
        WHEN sync_state.oldest_notification_synced_at IS NULL THEN excluded.oldest_notification_synced_at
        ELSE MIN(excluded.oldest_notification_synced_at, sync_state.oldest_notification_synced_at)
    END,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
RETURNING *;
