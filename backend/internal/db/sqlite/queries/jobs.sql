-- name: EnqueueJob :one
INSERT INTO jobs (queue, payload, max_attempts, scheduled_at, created_at, updated_at)
VALUES (?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
RETURNING *;

-- name: DequeueJob :one
-- Atomically claim the next available job from a queue
-- Uses a subquery to find the job, then UPDATE with RETURNING to claim it
UPDATE jobs
SET status = 'processing',
    started_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    attempts = attempts + 1,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = (
    SELECT j.id FROM jobs j
    WHERE j.queue = ?
      AND j.status = 'pending'
      AND j.scheduled_at <= strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
    ORDER BY j.scheduled_at ASC
    LIMIT 1
)
RETURNING *;

-- name: AckJob :exec
-- Mark job as completed and delete it
DELETE FROM jobs WHERE id = ?;

-- name: NackJobRetry :exec
-- Mark job for retry with exponential backoff
-- scheduled_at is passed as parameter (calculated in Go)
UPDATE jobs
SET status = 'pending',
    started_at = NULL,
    scheduled_at = ?,
    last_error = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?;

-- name: NackJobFailed :exec
-- Mark job as permanently failed (dead letter)
UPDATE jobs
SET status = 'failed',
    last_error = ?,
    completed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = ?;

-- name: ResetStaleJobs :execrows
-- Reset jobs that have been processing for too long (crashed workers)
-- cutoff_time is passed as parameter
UPDATE jobs
SET status = 'pending',
    started_at = NULL,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE status = 'processing'
  AND started_at < ?;

-- name: GetQueueStats :one
-- Get statistics for observability
SELECT
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_count,
    COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing_count,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count
FROM jobs
WHERE queue = ?;

-- name: GetAllQueueStats :one
-- Get overall statistics
SELECT
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_count,
    COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing_count,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count
FROM jobs;

-- name: ListFailedJobs :many
-- List failed jobs for debugging/retry
SELECT * FROM jobs
WHERE queue = ? AND status = 'failed'
ORDER BY updated_at DESC
LIMIT ?;

-- name: RetryFailedJob :exec
-- Manually retry a failed job
UPDATE jobs
SET status = 'pending',
    attempts = 0,
    scheduled_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    last_error = NULL,
    completed_at = NULL,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ? AND status = 'failed';

-- name: DeleteOldCompletedJobs :execrows
-- Cleanup old completed jobs (if we ever keep them instead of deleting on ack)
DELETE FROM jobs
WHERE status = 'completed'
  AND completed_at < ?;

-- name: DeleteOldFailedJobs :execrows
-- Cleanup old failed jobs after retention period
DELETE FROM jobs
WHERE status = 'failed'
  AND completed_at < ?;

-- name: CountPendingByQueue :one
-- Count pending jobs in a specific queue
SELECT COUNT(*) as count FROM jobs
WHERE queue = ? AND status = 'pending';

