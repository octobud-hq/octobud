// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/sqlite"
)

// Jitter configuration for exponential backoff
const (
	// jitterFactor is the range of randomization applied to backoff.
	// 0.5 means the actual delay will be between 50% and 150% of the base backoff.
	jitterFactor = 0.5
)

// Queue names for different job types
const (
	QueueProcessNotification      = "process_notification"
	QueueApplyRule                = "apply_rule"
	QueueSyncOlder                = "sync_older"
	QueueApplyRulesToNotification = "apply_rules_to_notification"
)

// Default configuration
const (
	DefaultMaxAttempts       = 5
	DefaultVisibilityTimeout = 5 * time.Minute
)

// ErrNoJobAvailable is returned when Dequeue finds no jobs to process
var ErrNoJobAvailable = errors.New("no job available")

// Job represents a queued job with its metadata
type Job struct {
	ID          int64
	Queue       string
	Payload     []byte
	Attempts    int
	MaxAttempts int
	CreatedAt   time.Time
	ScheduledAt time.Time
}

// EnqueueParams contains parameters for enqueueing a new job
type EnqueueParams struct {
	Queue       string
	Payload     []byte
	MaxAttempts int           // Default: 5
	Delay       time.Duration // For delayed/scheduled jobs
}

// QueueStats contains statistics about the job queue
type QueueStats struct {
	Pending    int64
	Processing int64
	Failed     int64
}

// JobQueue provides persistent job queue operations
type JobQueue interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, params EnqueueParams) (int64, error)

	// Dequeue atomically claims the next available job from a queue
	// Returns ErrNoJobAvailable if no jobs are available
	Dequeue(ctx context.Context, queue string) (*Job, error)

	// Ack marks a job as successfully completed (deletes it)
	Ack(ctx context.Context, jobID int64) error

	// Nack marks a job as failed - will retry with backoff or dead-letter
	Nack(ctx context.Context, jobID int64, err error) error

	// NackWithDelay marks a job as failed with a minimum retry delay.
	// This is useful when the server provides a Retry-After header.
	// The actual delay will be max(minDelay, calculatedBackoff).
	NackWithDelay(ctx context.Context, jobID int64, err error, minDelay time.Duration) error

	// ResetStale reclaims jobs stuck in "processing" state (crashed workers)
	ResetStale(ctx context.Context, timeout time.Duration) (int64, error)

	// Stats returns queue statistics for observability
	Stats(ctx context.Context, queue string) (QueueStats, error)

	// AllStats returns overall statistics across all queues
	AllStats(ctx context.Context) (QueueStats, error)
}

// SQLiteJobQueue implements JobQueue using SQLite
type SQLiteJobQueue struct {
	queries *sqlite.Queries
}

// NewSQLiteJobQueue creates a new SQLite-backed job queue
func NewSQLiteJobQueue(sqlDB *sql.DB) *SQLiteJobQueue {
	return &SQLiteJobQueue{
		queries: sqlite.New(sqlDB),
	}
}

// Enqueue adds a job to the queue
func (q *SQLiteJobQueue) Enqueue(ctx context.Context, params EnqueueParams) (int64, error) {
	maxAttempts := params.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}

	scheduledAt := time.Now().UTC()
	if params.Delay > 0 {
		scheduledAt = scheduledAt.Add(params.Delay)
	}

	// Retry on SQLITE_BUSY to handle concurrent access
	job, err := db.RetryOnBusy(ctx, func() (sqlite.Job, error) {
		return q.queries.EnqueueJob(ctx, sqlite.EnqueueJobParams{
			Queue:       params.Queue,
			Payload:     string(params.Payload),
			MaxAttempts: int64(maxAttempts),
			ScheduledAt: scheduledAt.Format(time.RFC3339),
		})
	})
	if err != nil {
		return 0, err
	}

	return job.ID, nil
}

// Dequeue atomically claims the next available job from a queue
func (q *SQLiteJobQueue) Dequeue(ctx context.Context, queue string) (*Job, error) {
	// Retry on SQLITE_BUSY to handle concurrent access
	sqliteJob, err := db.RetryOnBusy(ctx, func() (sqlite.Job, error) {
		return q.queries.DequeueJob(ctx, queue)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoJobAvailable
		}
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, sqliteJob.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse createdAt: %w", err)
	}
	scheduledAt, err := time.Parse(time.RFC3339, sqliteJob.ScheduledAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scheduledAt: %w", err)
	}

	return &Job{
		ID:          sqliteJob.ID,
		Queue:       sqliteJob.Queue,
		Payload:     []byte(sqliteJob.Payload),
		Attempts:    int(sqliteJob.Attempts),
		MaxAttempts: int(sqliteJob.MaxAttempts),
		CreatedAt:   createdAt,
		ScheduledAt: scheduledAt,
	}, nil
}

// Ack marks a job as successfully completed
func (q *SQLiteJobQueue) Ack(ctx context.Context, jobID int64) error {
	// Retry on SQLITE_BUSY - critical to ensure job is marked complete
	return db.RetryVoidOnBusy(ctx, func() error {
		return q.queries.AckJob(ctx, jobID)
	})
}

// Nack marks a job as failed - will retry with backoff or dead-letter
func (q *SQLiteJobQueue) Nack(ctx context.Context, jobID int64, jobErr error) error {
	return q.NackWithDelay(ctx, jobID, jobErr, 0)
}

// NackWithDelay marks a job as failed with an optional minimum retry delay.
// If minDelay is provided (> 0), the actual delay will be max(minDelay, calculatedBackoff).
// This is useful when the server provides a Retry-After header that should be respected.
func (q *SQLiteJobQueue) NackWithDelay(
	ctx context.Context,
	jobID int64,
	jobErr error,
	minDelay time.Duration,
) error {
	// Retry all operations on SQLITE_BUSY - critical to ensure job state is updated
	// If Nack fails, the job would be stuck in "processing" state until stale job cleanup

	// Get the job to check attempt count
	job, err := db.RetryOnBusy(ctx, func() (sqlite.Job, error) {
		return q.queries.GetJob(ctx, jobID)
	})
	if err != nil {
		return err
	}

	errStr := sql.NullString{}
	if jobErr != nil {
		errStr = sql.NullString{String: jobErr.Error(), Valid: true}
	}

	// If we've exhausted all attempts, mark as failed (dead letter)
	if job.Attempts >= job.MaxAttempts {
		return db.RetryVoidOnBusy(ctx, func() error {
			return q.queries.NackJobFailed(ctx, sqlite.NackJobFailedParams{
				ID:        jobID,
				LastError: errStr,
			})
		})
	}

	// Calculate exponential backoff with jitter to prevent thundering herd
	backoffDelay := calculateBackoffWithJitter(int(job.Attempts))

	// Use the larger of calculated backoff or server-specified minimum delay
	if minDelay > backoffDelay {
		backoffDelay = minDelay
	}

	nextSchedule := time.Now().UTC().Add(backoffDelay)

	return db.RetryVoidOnBusy(ctx, func() error {
		return q.queries.NackJobRetry(ctx, sqlite.NackJobRetryParams{
			ID:          jobID,
			ScheduledAt: nextSchedule.Format(time.RFC3339),
			LastError:   errStr,
		})
	})
}

// calculateBackoffWithJitter computes an exponential backoff delay with random jitter.
// Base formula: 2^attempts seconds, with ±50% jitter.
// This prevents the thundering herd problem when multiple workers fail at the same time.
func calculateBackoffWithJitter(attempts int) time.Duration {
	// Base exponential backoff: 2s, 4s, 8s, 16s, 32s...
	// Starting at 2s gives a reasonable initial delay for API failures.
	baseBackoffSeconds := math.Pow(2, float64(attempts))
	if baseBackoffSeconds < 2 {
		baseBackoffSeconds = 2
	}

	// Cap at 5 minutes before applying jitter
	const maxBackoffSeconds = 300.0
	if baseBackoffSeconds > maxBackoffSeconds {
		baseBackoffSeconds = maxBackoffSeconds
	}

	// Apply jitter: multiply by random factor between (1-jitterFactor) and (1+jitterFactor)
	// This spreads retries across time to avoid synchronized retry storms
	// #nosec G404 -- math/rand is sufficient for jitter; no security requirement
	jitterMultiplier := 1.0 - jitterFactor + (rand.Float64() * 2 * jitterFactor)
	backoffSeconds := baseBackoffSeconds * jitterMultiplier

	return time.Duration(backoffSeconds * float64(time.Second))
}

// ResetStale reclaims jobs stuck in "processing" state
func (q *SQLiteJobQueue) ResetStale(ctx context.Context, timeout time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-timeout)
	// Retry on SQLITE_BUSY
	return db.RetryOnBusy(ctx, func() (int64, error) {
		return q.queries.ResetStaleJobs(ctx, sql.NullString{
			String: cutoff.Format(time.RFC3339),
			Valid:  true,
		})
	})
}

// Stats returns queue statistics for a specific queue
func (q *SQLiteJobQueue) Stats(ctx context.Context, queue string) (QueueStats, error) {
	stats, err := q.queries.GetQueueStats(ctx, queue)
	if err != nil {
		return QueueStats{}, err
	}
	return QueueStats{
		Pending:    stats.PendingCount,
		Processing: stats.ProcessingCount,
		Failed:     stats.FailedCount,
	}, nil
}

// AllStats returns overall statistics across all queues
func (q *SQLiteJobQueue) AllStats(ctx context.Context) (QueueStats, error) {
	stats, err := q.queries.GetAllQueueStats(ctx)
	if err != nil {
		return QueueStats{}, err
	}
	return QueueStats{
		Pending:    stats.PendingCount,
		Processing: stats.ProcessingCount,
		Failed:     stats.FailedCount,
	}, nil
}
