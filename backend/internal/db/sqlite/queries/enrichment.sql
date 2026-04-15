-- name: ListNotificationsForEnrichment :many
-- Returns notifications that need enrichment (version < target).
-- Scoped by subject_type when provided, otherwise all types.
SELECT id, github_id, subject_type, subject_url, subject_raw, enrichment_version
FROM notifications
WHERE user_id = ?
  AND enrichment_version < sqlc.arg(target_version)
  AND subject_raw IS NOT NULL
ORDER BY effective_sort_date DESC
LIMIT sqlc.arg(batch_limit);

-- name: UpdateNotificationEnrichmentVersion :exec
UPDATE notifications SET enrichment_version = ? WHERE id = ?;
