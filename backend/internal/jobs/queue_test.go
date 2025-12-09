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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

var queueTestDBCounter int64

// setupQueueTestDB creates an in-memory SQLite database with the jobs table.
// Each test gets a unique database name to prevent conflicts when running in parallel.
func setupQueueTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use a unique database name per test to avoid conflicts
	dbNum := atomic.AddInt64(&queueTestDBCounter, 1)
	dsn := fmt.Sprintf("file:queuetestdb%d?mode=memory&cache=shared", dbNum)

	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			queue TEXT NOT NULL,
			payload TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 5,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			scheduled_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			started_at TEXT,
			completed_at TEXT,
			last_error TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_jobs_poll ON jobs(queue, status, scheduled_at);
	`)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestSQLiteJobQueue_EnqueueDequeue(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	payload := []byte(`{"test": "data"}`)
	jobID, err := queue.Enqueue(ctx, EnqueueParams{
		Queue:   QueueProcessNotification,
		Payload: payload,
	})
	require.NoError(t, err)
	require.Greater(t, jobID, int64(0))

	job, err := queue.Dequeue(ctx, QueueProcessNotification)
	require.NoError(t, err)
	require.NotNil(t, job)
	require.Equal(t, jobID, job.ID)
	require.Equal(t, QueueProcessNotification, job.Queue)
	require.Equal(t, payload, job.Payload)
	require.Equal(t, 1, job.Attempts)
	require.Equal(t, DefaultMaxAttempts, job.MaxAttempts)
}

func TestSQLiteJobQueue_DequeueEmpty(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	job, err := queue.Dequeue(ctx, QueueProcessNotification)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoJobAvailable))
	require.Nil(t, job)
}

func TestSQLiteJobQueue_Ack(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	jobID, err := queue.Enqueue(ctx, EnqueueParams{
		Queue:   QueueProcessNotification,
		Payload: []byte("test"),
	})
	require.NoError(t, err)

	job, err := queue.Dequeue(ctx, QueueProcessNotification)
	require.NoError(t, err)
	require.Equal(t, jobID, job.ID)

	err = queue.Ack(ctx, job.ID)
	require.NoError(t, err)

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE id = ?", job.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestSQLiteJobQueue_NackRetry(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	jobID, err := queue.Enqueue(ctx, EnqueueParams{
		Queue:       QueueProcessNotification,
		Payload:     []byte("test"),
		MaxAttempts: 3,
	})
	require.NoError(t, err)

	job, err := queue.Dequeue(ctx, QueueProcessNotification)
	require.NoError(t, err)
	require.Equal(t, jobID, job.ID)
	require.Equal(t, 1, job.Attempts)

	err = queue.Nack(ctx, job.ID, errors.New("temporary failure"))
	require.NoError(t, err)

	var status string
	err = db.QueryRowContext(ctx, "SELECT status FROM jobs WHERE id = ?", job.ID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "pending", status)
}

func TestSQLiteJobQueue_NackDeadLetter(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	jobID, err := queue.Enqueue(ctx, EnqueueParams{
		Queue:       QueueProcessNotification,
		Payload:     []byte("test"),
		MaxAttempts: 1,
	})
	require.NoError(t, err)

	job, err := queue.Dequeue(ctx, QueueProcessNotification)
	require.NoError(t, err)
	require.Equal(t, jobID, job.ID)

	err = queue.Nack(ctx, job.ID, errors.New("permanent failure"))
	require.NoError(t, err)

	var status string
	err = db.QueryRowContext(ctx, "SELECT status FROM jobs WHERE id = ?", job.ID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "failed", status)
}

func TestSQLiteJobQueue_ResetStale(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		INSERT INTO jobs (queue, payload, status, started_at, scheduled_at)
		VALUES (?, ?, 'processing', datetime('now', '-10 minutes'), datetime('now'))
	`, QueueProcessNotification, "stale-job")
	require.NoError(t, err)

	count, err := queue.ResetStale(ctx, 5*time.Minute)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	var status string
	err = db.QueryRowContext(ctx, "SELECT status FROM jobs WHERE payload = 'stale-job'").
		Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "pending", status)
}

func TestSQLiteJobQueue_Stats(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		INSERT INTO jobs (queue, payload, status, scheduled_at) VALUES
		(?, 'p1', 'pending', datetime('now')),
		(?, 'p2', 'pending', datetime('now')),
		(?, 'proc1', 'processing', datetime('now')),
		(?, 'f1', 'failed', datetime('now'))
	`, QueueProcessNotification, QueueProcessNotification, QueueProcessNotification, QueueProcessNotification)
	require.NoError(t, err)

	stats, err := queue.Stats(ctx, QueueProcessNotification)
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Pending)
	require.Equal(t, int64(1), stats.Processing)
	require.Equal(t, int64(1), stats.Failed)
}

func TestSQLiteJobQueue_EnqueueWithDelay(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	jobID, err := queue.Enqueue(ctx, EnqueueParams{
		Queue:   QueueProcessNotification,
		Payload: []byte("delayed"),
		Delay:   1 * time.Hour,
	})
	require.NoError(t, err)

	job, err := queue.Dequeue(ctx, QueueProcessNotification)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoJobAvailable))
	require.Nil(t, job)

	var scheduledAt string
	err = db.QueryRowContext(ctx, "SELECT scheduled_at FROM jobs WHERE id = ?", jobID).
		Scan(&scheduledAt)
	require.NoError(t, err)

	scheduled, err := time.Parse(time.RFC3339, scheduledAt)
	require.NoError(t, err)
	require.True(t, scheduled.After(time.Now().Add(30*time.Minute)))
}

func TestSQLiteJobQueue_QueueIsolation(t *testing.T) {
	db := setupQueueTestDB(t)
	queue := NewSQLiteJobQueue(db)
	ctx := context.Background()

	_, err := queue.Enqueue(ctx, EnqueueParams{
		Queue:   QueueProcessNotification,
		Payload: []byte("notification"),
	})
	require.NoError(t, err)

	_, err = queue.Enqueue(ctx, EnqueueParams{
		Queue:   QueueApplyRule,
		Payload: []byte("rule"),
	})
	require.NoError(t, err)

	job, err := queue.Dequeue(ctx, QueueProcessNotification)
	require.NoError(t, err)
	require.Equal(t, []byte("notification"), job.Payload)

	job, err = queue.Dequeue(ctx, QueueApplyRule)
	require.NoError(t, err)
	require.Equal(t, []byte("rule"), job.Payload)
}
