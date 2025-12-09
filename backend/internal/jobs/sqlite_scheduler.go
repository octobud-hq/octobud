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
	gosync "sync"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/core/auth"
	"github.com/octobud-hq/octobud/backend/internal/core/update"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/jobs/handlers"
	coresync "github.com/octobud-hq/octobud/backend/internal/sync"
)

// SQLiteScheduler is a job scheduler for single-user/SQLite mode.
// It uses a persistent job queue to ensure at-least-once delivery.
type SQLiteScheduler struct {
	logger       *zap.Logger
	store        db.Store
	syncService  coresync.SyncOperations
	syncInterval time.Duration

	// Persistent job queue for reliable processing
	jobQueue JobQueue

	// Handlers for job processing
	applyRuleHandler            *handlers.ApplyRuleHandler
	processNotificationHandler  *handlers.ProcessNotificationHandler
	syncNotificationsHandler    *handlers.SyncNotificationsHandler
	syncOlderHandler            *handlers.SyncOlderHandler
	cleanupNotificationsHandler *handlers.CleanupNotificationsHandler
	checkUpdatesHandler         *handlers.CheckUpdatesHandler

	// Channels for non-persistent jobs (sync triggers)
	syncNotificationsQueue chan struct{}
	applyRuleQueue         chan applyRuleJob
	syncOlderQueue         chan SyncOlderNotificationsArgs

	// Control channels
	stopCh  chan struct{}
	doneCh  chan struct{}
	mu      gosync.Mutex
	running bool

	// Worker pool for notification processing
	notificationWorkers int
	workerWg            gosync.WaitGroup
}

// applyRuleJob contains the data needed to apply a rule
type applyRuleJob struct {
	UserID string
	RuleID string
}

// SQLiteSchedulerConfig contains configuration for the SQLite scheduler.
type SQLiteSchedulerConfig struct {
	Logger        *zap.Logger
	DBConn        *sql.DB // Required for persistent job queue
	Store         db.Store
	SyncService   coresync.SyncOperations
	SyncInterval  time.Duration
	AuthService   auth.AuthService
	UpdateService *update.Service
}

// Default number of workers for processing notifications concurrently.
const defaultNotificationWorkers = 4

// Interval for checking stale jobs and cleaning up
const staleJobCheckInterval = 1 * time.Minute

// Interval for cleanup job (daily)
const cleanupInterval = 24 * time.Hour

// NewSQLiteScheduler creates a new SQLite scheduler with persistent job queue.
func NewSQLiteScheduler(cfg SQLiteSchedulerConfig) *SQLiteScheduler {
	if cfg.SyncInterval == 0 {
		cfg.SyncInterval = 30 * time.Second
	}

	s := &SQLiteScheduler{
		logger:                 cfg.Logger,
		store:                  cfg.Store,
		syncService:            cfg.SyncService,
		syncInterval:           cfg.SyncInterval,
		jobQueue:               NewSQLiteJobQueue(cfg.DBConn),
		syncNotificationsQueue: make(chan struct{}, 10),
		applyRuleQueue:         make(chan applyRuleJob, 10),
		syncOlderQueue:         make(chan SyncOlderNotificationsArgs, 10),
		stopCh:                 make(chan struct{}),
		doneCh:                 make(chan struct{}),
		notificationWorkers:    defaultNotificationWorkers,
	}

	// Initialize handlers
	s.applyRuleHandler = handlers.NewApplyRuleHandler(cfg.Store, cfg.Logger)
	s.processNotificationHandler = handlers.NewProcessNotificationHandler(
		cfg.Store,
		cfg.SyncService,
		cfg.Logger,
	)
	s.syncNotificationsHandler = handlers.NewSyncNotificationsHandler(
		cfg.SyncService,
		s,
		cfg.Logger,
	)
	s.syncOlderHandler = handlers.NewSyncOlderHandler(cfg.SyncService, s, cfg.Logger)
	s.cleanupNotificationsHandler = handlers.NewCleanupNotificationsHandler(cfg.Store, cfg.Logger)

	// Initialize update check handler if services are provided
	if cfg.AuthService != nil && cfg.UpdateService != nil {
		s.checkUpdatesHandler = handlers.NewCheckUpdatesHandler(
			cfg.AuthService,
			cfg.UpdateService,
			cfg.Logger,
		)
	}

	return s
}

// Start begins the scheduler's background processing.
func (s *SQLiteScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	// Reset any stale jobs from previous runs (crashed workers)
	if count, err := s.jobQueue.ResetStale(ctx, DefaultVisibilityTimeout); err == nil && count > 0 {
		s.logger.Info("reset stale jobs from previous run", zap.Int64("count", count))
	}

	// Log pending job stats
	if stats, err := s.jobQueue.AllStats(ctx); err == nil {
		s.logger.Info("job queue status on startup",
			zap.Int64("pending", stats.Pending),
			zap.Int64("processing", stats.Processing),
			zap.Int64("failed", stats.Failed))
	}

	// Start notification processing workers
	s.logger.Info("starting notification workers", zap.Int("count", s.notificationWorkers))
	for i := 0; i < s.notificationWorkers; i++ {
		s.workerWg.Add(1)
		go s.notificationWorker(ctx, i)
	}

	// Start stale job cleanup goroutine
	s.workerWg.Add(1)
	go s.staleJobCleanupLoop(ctx)

	// Start daily cleanup loop
	s.workerWg.Add(1)
	go s.cleanupLoop(ctx)

	// Start update check loop (if handler is configured)
	if s.checkUpdatesHandler != nil {
		s.workerWg.Add(1)
		go s.updateCheckLoop(ctx)
	}

	go s.run(ctx)
	return nil
}

// Stop gracefully shuts down the scheduler.
func (s *SQLiteScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	close(s.stopCh)

	// Wait for main loop to finish
	select {
	case <-s.doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for notification workers to finish processing
	s.logger.Info("waiting for workers to finish")
	workersDone := make(chan struct{})
	go func() {
		s.workerWg.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		s.logger.Info("all workers finished")
		return nil
	case <-ctx.Done():
		s.logger.Warn("context cancelled while waiting for workers")
		return ctx.Err()
	}
}

// getCurrentUserID gets the current user's GitHub user ID for data scoping.
// In single-user mode, this returns the GitHub user ID of the authenticated user.
func (s *SQLiteScheduler) getCurrentUserID(ctx context.Context) (string, error) {
	if s.store == nil {
		return "", errors.New("store not configured")
	}
	user, err := s.store.GetUser(ctx)
	if err != nil {
		return "", err
	}
	if !user.GithubUserID.Valid || user.GithubUserID.String == "" {
		return "", errors.New("user has no GitHub identity configured")
	}
	return user.GithubUserID.String, nil
}

// EnqueueSyncNotifications enqueues a sync job.
func (s *SQLiteScheduler) EnqueueSyncNotifications(ctx context.Context, _ string) error {
	select {
	case s.syncNotificationsQueue <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue is full, skip - sync will happen on next interval anyway
		return nil
	}
}

// EnqueueProcessNotification enqueues a notification processing job.
// Jobs are persisted to the database for reliable at-least-once delivery.
func (s *SQLiteScheduler) EnqueueProcessNotification(
	ctx context.Context,
	_ string,
	notificationData []byte,
) error {
	jobID, err := s.jobQueue.Enqueue(ctx, EnqueueParams{
		Queue:       QueueProcessNotification,
		Payload:     notificationData,
		MaxAttempts: DefaultMaxAttempts,
	})
	if err != nil {
		s.logger.Warn("failed to enqueue notification job", zap.Error(err))
		return err
	}

	s.logger.Debug("notification job enqueued", zap.Int64("jobID", jobID))
	return nil
}

// EnqueueApplyRule enqueues a rule application job with UUID rule ID.
func (s *SQLiteScheduler) EnqueueApplyRule(
	ctx context.Context,
	userID string,
	ruleID string,
) error {
	select {
	case s.applyRuleQueue <- applyRuleJob{UserID: userID, RuleID: ruleID}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// EnqueueSyncOlder enqueues a job to sync older notifications.
func (s *SQLiteScheduler) EnqueueSyncOlder(
	ctx context.Context,
	args SyncOlderNotificationsArgs,
) error {
	select {
	case s.syncOlderQueue <- args:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue is full, run in background goroutine
		go s.doSyncOlder(context.Background(), args)
		return nil
	}
}

func (s *SQLiteScheduler) run(ctx context.Context) {
	defer close(s.doneCh)

	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	// Run initial sync
	s.doSync(ctx)

	for {
		select {
		case <-s.stopCh:
			s.logger.Info("scheduler stopping")
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			return

		case <-ctx.Done():
			s.logger.Info("scheduler context cancelled")
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			return

		case <-ticker.C:
			s.doSync(ctx)

		case <-s.syncNotificationsQueue:
			s.doSync(ctx)

		case job := <-s.applyRuleQueue:
			s.doApplyRule(ctx, job)

		case args := <-s.syncOlderQueue:
			s.doSyncOlder(ctx, args)
		}
	}
}

// notificationWorker processes notifications from the persistent job queue.
// Multiple workers run concurrently for better throughput during large syncs.
func (s *SQLiteScheduler) notificationWorker(ctx context.Context, workerID int) {
	defer s.workerWg.Done()
	s.logger.Debug("notification worker started", zap.Int("workerID", workerID))

	pollInterval := 100 * time.Millisecond

	for {
		select {
		case <-s.stopCh:
			s.logger.Debug("notification worker stopping", zap.Int("workerID", workerID))
			return
		case <-ctx.Done():
			s.logger.Debug("notification worker context cancelled", zap.Int("workerID", workerID))
			return
		default:
		}

		// Try to dequeue a job
		job, err := s.jobQueue.Dequeue(ctx, QueueProcessNotification)
		if err != nil {
			if errors.Is(err, ErrNoJobAvailable) {
				// No jobs available, wait before polling again
				select {
				case <-s.stopCh:
					return
				case <-ctx.Done():
					return
				case <-time.After(pollInterval):
					continue
				}
			}
			// Actual error
			s.logger.Warn("failed to dequeue job", zap.Int("workerID", workerID), zap.Error(err))
			time.Sleep(time.Second) // Back off on errors
			continue
		}

		// Get current user ID for processing
		userID, err := s.getCurrentUserID(ctx)
		if err != nil {
			s.logger.Warn("cannot process notification - no user ID",
				zap.Int64("jobID", job.ID),
				zap.Error(err))
			// Nack the job to retry later when user is configured
			if nackErr := s.jobQueue.Nack(ctx, job.ID, err); nackErr != nil {
				s.logger.Error("failed to nack job", zap.Int64("jobID", job.ID), zap.Error(nackErr))
			}
			continue
		}

		// Process the job
		s.logger.Debug("processing notification job",
			zap.Int("workerID", workerID),
			zap.Int64("jobID", job.ID),
			zap.Int("attempt", job.Attempts),
			zap.Int("maxAttempts", job.MaxAttempts))

		err = s.processNotificationHandler.Handle(ctx, userID, job.Payload)
		if err != nil {
			s.logger.Warn("notification job failed",
				zap.Int64("jobID", job.ID),
				zap.Int("attempt", job.Attempts),
				zap.Error(err))

			// Nack will either retry or dead-letter the job
			if nackErr := s.jobQueue.Nack(ctx, job.ID, err); nackErr != nil {
				s.logger.Error("failed to nack job", zap.Int64("jobID", job.ID), zap.Error(nackErr))
			}
		} else {
			s.logger.Debug("notification job completed",
				zap.Int64("jobID", job.ID),
				zap.Int("attempt", job.Attempts))

			// Ack removes the job from the queue
			if ackErr := s.jobQueue.Ack(ctx, job.ID); ackErr != nil {
				s.logger.Error("failed to ack job", zap.Int64("jobID", job.ID), zap.Error(ackErr))
			}
		}
	}
}

// staleJobCleanupLoop periodically resets jobs stuck in processing state
func (s *SQLiteScheduler) staleJobCleanupLoop(ctx context.Context) {
	defer s.workerWg.Done()

	ticker := time.NewTicker(staleJobCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := s.jobQueue.ResetStale(ctx, DefaultVisibilityTimeout)
			if err != nil {
				s.logger.Warn("failed to reset stale jobs", zap.Error(err))
			} else if count > 0 {
				s.logger.Info("reset stale jobs", zap.Int64("count", count))
			}
		}
	}
}

func (s *SQLiteScheduler) doSync(ctx context.Context) {
	s.logger.Debug("starting notification sync")

	// Get current user ID
	userID, err := s.getCurrentUserID(ctx)
	if err != nil {
		s.logger.Debug("skipping sync - no user ID configured", zap.Error(err))
		return
	}

	result, err := s.syncNotificationsHandler.Handle(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to sync notifications", zap.Error(err))
		return
	}

	// Update sync state after successful processing
	if result != nil {
		if err := s.syncNotificationsHandler.UpdateSyncState(ctx, result); err != nil {
			s.logger.Warn("failed to update sync state", zap.Error(err))
		}
	}
}

func (s *SQLiteScheduler) doApplyRule(ctx context.Context, job applyRuleJob) {
	err := s.applyRuleHandler.Handle(ctx, job.UserID, job.RuleID)
	if err != nil {
		s.logger.Warn("failed to apply rule", zap.String("ruleID", job.RuleID), zap.Error(err))
	}
}

func (s *SQLiteScheduler) doSyncOlder(ctx context.Context, args SyncOlderNotificationsArgs) {
	// Get current user ID
	userID, err := s.getCurrentUserID(ctx)
	if err != nil {
		s.logger.Debug("skipping sync older - no user ID configured", zap.Error(err))
		return
	}

	err = s.syncOlderHandler.Handle(ctx, handlers.SyncOlderArgs{
		UserID:     userID,
		Days:       args.Days,
		UntilTime:  args.UntilTime,
		MaxCount:   args.MaxCount,
		UnreadOnly: args.UnreadOnly,
	})
	if err != nil {
		s.logger.Warn("failed to sync older notifications", zap.Error(err))
	}
}

// cleanupLoop runs the notification cleanup job daily
func (s *SQLiteScheduler) cleanupLoop(ctx context.Context) {
	defer s.workerWg.Done()

	// Run cleanup on startup (after a short delay to let other things initialize)
	select {
	case <-s.stopCh:
		return
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
		s.doCleanup(ctx)
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doCleanup(ctx)
		}
	}
}

func (s *SQLiteScheduler) doCleanup(ctx context.Context) {
	s.logger.Debug("starting daily cleanup")

	// Get current user ID
	userID, err := s.getCurrentUserID(ctx)
	if err != nil {
		s.logger.Debug("skipping cleanup - no user ID configured", zap.Error(err))
		return
	}

	result, err := s.cleanupNotificationsHandler.Handle(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to run cleanup", zap.Error(err))
		return
	}

	if result.Skipped {
		s.logger.Debug("cleanup skipped", zap.String("reason", result.SkipReason))
	} else {
		s.logger.Info("daily cleanup completed",
			zap.Int64("notificationsDeleted", result.NotificationsDeleted),
			zap.Int64("pullRequestsDeleted", result.PullRequestsDeleted))
	}
}

// GetCleanupHandler returns the cleanup handler for API access
func (s *SQLiteScheduler) GetCleanupHandler() *handlers.CleanupNotificationsHandler {
	return s.cleanupNotificationsHandler
}

// updateCheckInterval is how often to check for updates (if enabled and frequency allows)
const updateCheckInterval = 1 * time.Hour

func (s *SQLiteScheduler) updateCheckLoop(ctx context.Context) {
	defer s.workerWg.Done()

	// Run update check on startup (after a short delay)
	select {
	case <-s.stopCh:
		return
	case <-ctx.Done():
		return
	case <-time.After(60 * time.Second): // Wait 1 minute after startup
		s.doUpdateCheck(ctx)
	}

	ticker := time.NewTicker(updateCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doUpdateCheck(ctx)
		}
	}
}

func (s *SQLiteScheduler) doUpdateCheck(ctx context.Context) {
	if s.checkUpdatesHandler == nil {
		return
	}

	s.logger.Debug("checking for updates")

	available, err := s.checkUpdatesHandler.Handle(ctx)
	if err != nil {
		s.logger.Warn("failed to check for updates", zap.Error(err))
		return
	}

	if available {
		s.logger.Info("update check found new version available")
		// Note: The frontend will poll the API endpoint to get update details
		// We just log here that a check was performed
	}
}
