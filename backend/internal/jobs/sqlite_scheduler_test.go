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
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/octobud-hq/octobud/backend/internal/db"
	dbmocks "github.com/octobud-hq/octobud/backend/internal/db/mocks"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/sync"
	syncmocks "github.com/octobud-hq/octobud/backend/internal/sync/mocks"

	_ "modernc.org/sqlite"
)

var testDBCounter int64

// setupMockStore creates a mock store that returns a test user
func setupMockStore(ctrl *gomock.Controller) *dbmocks.MockStore {
	mockStore := dbmocks.NewMockStore(ctrl)
	mockStore.EXPECT().GetUser(gomock.Any()).Return(db.User{
		GithubUserID: sql.NullString{String: "test-user-id", Valid: true},
	}, nil).AnyTimes()
	// ProcessNotification handler may call GetNotificationByGithubID
	mockStore.EXPECT().GetNotificationByGithubID(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(db.Notification{}, sql.ErrNoRows).AnyTimes()
	return mockStore
}

// setupTestDB creates an in-memory SQLite database with the jobs table for testing.
// Each test gets a unique database name to prevent conflicts when running in parallel.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use a unique database name per test to avoid conflicts
	dbNum := atomic.AddInt64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:testdb%d?mode=memory&cache=shared", dbNum)

	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	// Create the jobs table
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

func TestSQLiteScheduler_NewScheduler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)

	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       logger,
		DBConn:       db,
		Store:        nil,
		SyncService:  mockSync,
		SyncInterval: 1 * time.Minute,
	})

	require.NotNil(t, scheduler)
	require.Equal(t, 1*time.Minute, scheduler.syncInterval)
	require.Equal(t, defaultNotificationWorkers, scheduler.notificationWorkers)
	require.NotNil(t, scheduler.jobQueue)
}

func TestSQLiteScheduler_DefaultSyncInterval(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	db := setupTestDB(t)

	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zap.NewNop(),
		DBConn:       db,
		SyncService:  mockSync,
		SyncInterval: 0, // Should default to 30 seconds
	})

	require.Equal(t, 30*time.Second, scheduler.syncInterval)
}

func TestSQLiteScheduler_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().GetSyncContext(gomock.Any(), gomock.Any()).Return(sync.SyncContext{
		IsSyncConfigured: false,
	}, nil).AnyTimes()

	db := setupTestDB(t)
	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zap.NewNop(),
		DBConn:       db,
		SyncService:  mockSync,
		SyncInterval: 1 * time.Hour,
	})

	ctx := context.Background()

	// Start scheduler
	err := scheduler.Start(ctx)
	require.NoError(t, err)
	require.True(t, scheduler.running)

	// Starting again should be a no-op
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
	require.False(t, scheduler.running)
}

func TestSQLiteScheduler_StopWhenNotRunning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	db := setupTestDB(t)

	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:      zap.NewNop(),
		DBConn:      db,
		SyncService: mockSync,
	})

	// Stop without starting should be no-op
	err := scheduler.Stop(context.Background())
	require.NoError(t, err)
}

func TestSQLiteScheduler_EnqueueProcessNotification_Persisted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)

	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:      logger,
		DBConn:      db,
		SyncService: mockSync,
	})

	ctx := context.Background()
	testData := []byte(`{"id": "test-123"}`)

	err := scheduler.EnqueueProcessNotification(ctx, "1", testData)
	require.NoError(t, err)

	// Verify job is persisted in database
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE queue = ?", QueueProcessNotification).
		Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Verify payload
	var payload string
	err = db.QueryRowContext(ctx, "SELECT payload FROM jobs WHERE queue = ?", QueueProcessNotification).
		Scan(&payload)
	require.NoError(t, err)
	require.Equal(t, string(testData), payload)
}

func TestSQLiteScheduler_EnqueueSyncNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	db := setupTestDB(t)

	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:      zap.NewNop(),
		DBConn:      db,
		SyncService: mockSync,
	})

	ctx := context.Background()

	err := scheduler.EnqueueSyncNotifications(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, 1, len(scheduler.syncNotificationsQueue))
}

func TestSQLiteScheduler_EnqueueApplyRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	db := setupTestDB(t)

	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:      zap.NewNop(),
		DBConn:      db,
		SyncService: mockSync,
	})

	ctx := context.Background()

	err := scheduler.EnqueueApplyRule(ctx, "1", "42")
	require.NoError(t, err)

	select {
	case job := <-scheduler.applyRuleQueue:
		require.Equal(t, "42", job.RuleID)
	default:
		t.Fatal("expected rule ID in queue")
	}
}

func TestSQLiteScheduler_WorkerPoolProcessesNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var processedCount atomic.Int32

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().GetSyncContext(gomock.Any(), gomock.Any()).Return(sync.SyncContext{
		IsSyncConfigured: false,
	}, nil).AnyTimes()

	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ types.NotificationThread) error {
			processedCount.Add(1)
			return nil
		}).
		Times(5)

	mockStore := setupMockStore(ctrl)
	db := setupTestDB(t)
	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zap.NewNop(),
		DBConn:       db,
		Store:        mockStore,
		SyncService:  mockSync,
		SyncInterval: 1 * time.Hour,
	})

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Enqueue 5 notifications
	for i := 0; i < 5; i++ {
		thread := types.NotificationThread{
			ID: fmt.Sprintf("notif-%d", i),
			Repository: types.RepositorySnapshot{
				ID:       int64(i),
				FullName: "test/repo",
			},
		}
		data, _ := json.Marshal(thread)
		err := scheduler.EnqueueProcessNotification(ctx, "1", data)
		require.NoError(t, err)
	}

	// Wait for processing
	deadline := time.Now().Add(5 * time.Second)
	for processedCount.Load() < 5 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	require.Equal(t, int32(5), processedCount.Load())

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

func TestSQLiteScheduler_JobRetryOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var attempts atomic.Int32

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().GetSyncContext(gomock.Any(), gomock.Any()).Return(sync.SyncContext{
		IsSyncConfigured: false,
	}, nil).AnyTimes()

	// Fail first 2 times, then succeed
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ types.NotificationThread) error {
			attempt := attempts.Add(1)
			if attempt <= 2 {
				return fmt.Errorf("temporary error")
			}
			return nil
		}).
		Times(3)

	mockStore := setupMockStore(ctrl)
	db := setupTestDB(t)
	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zaptest.NewLogger(t),
		DBConn:       db,
		Store:        mockStore,
		SyncService:  mockSync,
		SyncInterval: 1 * time.Hour,
	})

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Enqueue a notification
	thread := types.NotificationThread{ID: "test-retry"}
	data, _ := json.Marshal(thread)
	err = scheduler.EnqueueProcessNotification(ctx, "1", data)
	require.NoError(t, err)

	// Wait for processing with retries (backoff is exponential: 1s, 2s, ...)
	deadline := time.Now().Add(10 * time.Second)
	for attempts.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}

	require.Equal(t, int32(3), attempts.Load(), "should have retried and succeeded")

	// Job should be deleted (acked) after success
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE queue = ?", QueueProcessNotification).
		Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count, "job should be deleted after successful processing")

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

func TestSQLiteScheduler_JobDeadLetterAfterMaxAttempts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var attempts atomic.Int32

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().GetSyncContext(gomock.Any(), gomock.Any()).Return(sync.SyncContext{
		IsSyncConfigured: false,
	}, nil).AnyTimes()

	// Always fail
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ types.NotificationThread) error {
			attempts.Add(1)
			return fmt.Errorf("permanent error")
		}).
		AnyTimes()

	mockStore := setupMockStore(ctrl)
	db := setupTestDB(t)
	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zaptest.NewLogger(t),
		DBConn:       db,
		Store:        mockStore,
		SyncService:  mockSync,
		SyncInterval: 1 * time.Hour,
	})

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Enqueue a notification with low max attempts for faster test
	_, err = db.ExecContext(ctx, `
		INSERT INTO jobs (queue, payload, max_attempts, scheduled_at)
		VALUES (?, ?, 2, datetime('now'))
	`, QueueProcessNotification, `{"id":"test-dead-letter"}`)
	require.NoError(t, err)

	// Wait for job to be dead-lettered
	deadline := time.Now().Add(10 * time.Second)
	var status string
	for time.Now().Before(deadline) {
		err = db.QueryRowContext(ctx, "SELECT status FROM jobs WHERE queue = ?", QueueProcessNotification).
			Scan(&status)
		if err == nil && status == "failed" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.Equal(t, "failed", status, "job should be marked as failed after max attempts")
	require.GreaterOrEqual(
		t,
		attempts.Load(),
		int32(2),
		"should have attempted at least max_attempts times",
	)

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

func TestSQLiteScheduler_ContextCancellationStopsWorkers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().GetSyncContext(gomock.Any(), gomock.Any()).Return(sync.SyncContext{
		IsSyncConfigured: false,
	}, nil).AnyTimes()

	db := setupTestDB(t)
	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zap.NewNop(),
		DBConn:       db,
		SyncService:  mockSync,
		SyncInterval: 1 * time.Hour,
	})

	ctx, cancel := context.WithCancel(context.Background())
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for doneCh to close
	select {
	case <-scheduler.doneCh:
		// Scheduler stopped as expected
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}

func TestSQLiteScheduler_PeriodicSyncTriggered(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var syncCalls atomic.Int32

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		GetSyncContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string) (sync.SyncContext, error) {
			syncCalls.Add(1)
			return sync.SyncContext{IsSyncConfigured: false}, nil
		}).
		AnyTimes()

	mockStore := setupMockStore(ctrl)
	db := setupTestDB(t)
	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zap.NewNop(),
		DBConn:       db,
		Store:        mockStore,
		SyncService:  mockSync,
		SyncInterval: 100 * time.Millisecond,
	})

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for at least 2 periodic syncs
	time.Sleep(250 * time.Millisecond)

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, syncCalls.Load(), int32(2))
}

func TestSQLiteScheduler_EnqueueSyncOlder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	db := setupTestDB(t)

	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:      zap.NewNop(),
		DBConn:      db,
		SyncService: mockSync,
	})

	ctx := context.Background()
	args := SyncOlderNotificationsArgs{
		Days:       30,
		UnreadOnly: true,
	}

	err := scheduler.EnqueueSyncOlder(ctx, args)
	require.NoError(t, err)

	select {
	case received := <-scheduler.syncOlderQueue:
		require.Equal(t, args.Days, received.Days)
		require.Equal(t, args.UnreadOnly, received.UnreadOnly)
	default:
		t.Fatal("expected args in queue")
	}
}

func TestSQLiteScheduler_ConcurrentNotificationProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().GetSyncContext(gomock.Any(), gomock.Any()).Return(sync.SyncContext{
		IsSyncConfigured: false,
	}, nil).AnyTimes()

	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ types.NotificationThread) error {
			current := currentConcurrent.Add(1)
			for {
				maxC := maxConcurrent.Load()
				if current <= maxC || maxConcurrent.CompareAndSwap(maxC, current) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			currentConcurrent.Add(-1)
			return nil
		}).
		Times(8)

	mockStore := setupMockStore(ctrl)
	db := setupTestDB(t)
	scheduler := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zap.NewNop(),
		DBConn:       db,
		Store:        mockStore,
		SyncService:  mockSync,
		SyncInterval: 1 * time.Hour,
	})

	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Enqueue 8 notifications
	for i := 0; i < 8; i++ {
		thread := types.NotificationThread{ID: fmt.Sprintf("notif-%d", i)}
		data, _ := json.Marshal(thread)
		err := scheduler.EnqueueProcessNotification(ctx, "1", data)
		require.NoError(t, err)
	}

	// Wait for all to be processed
	time.Sleep(1 * time.Second)

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, maxConcurrent.Load(), int32(2), "should have concurrent processing")
	require.LessOrEqual(
		t,
		maxConcurrent.Load(),
		int32(defaultNotificationWorkers),
		"should not exceed worker count",
	)
}

func TestSQLiteScheduler_JobsSurviveRestart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := setupTestDB(t)

	// First scheduler - enqueue jobs but don't process them
	mockSync1 := syncmocks.NewMockSyncOperations(ctrl)
	mockStore1 := setupMockStore(ctrl)

	scheduler1 := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zap.NewNop(),
		DBConn:       db,
		Store:        mockStore1,
		SyncService:  mockSync1,
		SyncInterval: 1 * time.Hour,
	})

	ctx := context.Background()

	// Enqueue jobs without starting the scheduler
	for i := 0; i < 3; i++ {
		thread := types.NotificationThread{ID: fmt.Sprintf("persist-%d", i)}
		data, _ := json.Marshal(thread)
		err := scheduler1.EnqueueProcessNotification(ctx, "1", data)
		require.NoError(t, err)
	}

	// Verify jobs are in database
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE status = 'pending'").
		Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// Second scheduler - should pick up the persisted jobs
	var processedCount atomic.Int32

	mockSync2 := syncmocks.NewMockSyncOperations(ctrl)
	mockSync2.EXPECT().GetSyncContext(gomock.Any(), gomock.Any()).Return(sync.SyncContext{
		IsSyncConfigured: false,
	}, nil).AnyTimes()

	mockSync2.EXPECT().
		ProcessNotification(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ types.NotificationThread) error {
			processedCount.Add(1)
			return nil
		}).
		Times(3)

	mockStore2 := setupMockStore(ctrl)
	scheduler2 := NewSQLiteScheduler(SQLiteSchedulerConfig{
		Logger:       zap.NewNop(),
		DBConn:       db,
		Store:        mockStore2,
		SyncService:  mockSync2,
		SyncInterval: 1 * time.Hour,
	})

	err = scheduler2.Start(ctx)
	require.NoError(t, err)

	// Wait for processing
	deadline := time.Now().Add(5 * time.Second)
	for processedCount.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	require.Equal(t, int32(3), processedCount.Load(), "should have processed all persisted jobs")

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler2.Stop(stopCtx)
	require.NoError(t, err)
}
