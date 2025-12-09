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

package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/github"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToGetSyncState           = errors.New("failed to get sync state")
	ErrFailedToUpdateSyncState        = errors.New("failed to update sync state")
	ErrFailedToFetchNotifications     = errors.New("failed to fetch notifications")
	ErrFailedToUpsertRepository       = errors.New("failed to upsert repository")
	ErrNotificationMissingSubjectURL  = errors.New("notification has no subject URL")
	ErrFailedToFetchSubject           = errors.New("failed to fetch subject")
	ErrFailedToGetRepository          = errors.New("failed to get repository")
	ErrFailedToExtractPullRequestData = errors.New("failed to extract pull request data")
)

// GetSyncContext gathers ALL necessary state for a sync operation.
// This should be called once at the start of a sync job.
func (s *Service) GetSyncContext(ctx context.Context, userID string) (SyncContext, error) {
	// Get sync settings
	syncSettings, err := s.getUserSyncSettings(ctx)
	if err != nil {
		s.logger.Error("failed to get sync settings", zap.Error(err))
		return SyncContext{}, errors.Join(ErrFailedToGetSyncState, err)
	}

	// Check if sync is configured
	isSyncConfigured := syncSettings != nil && syncSettings.SetupCompleted
	if !isSyncConfigured {
		s.logger.Debug("sync not configured")
		return SyncContext{UserID: userID, IsSyncConfigured: false}, nil
	}

	// Get sync state
	state, err := s.syncStateService.GetSyncState(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get sync state", zap.Error(err))
		return SyncContext{}, errors.Join(ErrFailedToGetSyncState, err)
	}

	// Determine if this is initial sync
	isInitialSync := !state.InitialSyncCompletedAt.Valid

	// Pre-compute the 'since' timestamp for GitHub API
	var sinceTimestamp *time.Time
	if isInitialSync {
		// Initial sync: use the user's configured time period
		if syncSettings.InitialSyncDays != nil && *syncSettings.InitialSyncDays > 0 {
			cutoff := calculateSyncSinceDate(*syncSettings.InitialSyncDays)
			sinceTimestamp = &cutoff
		}
		// If InitialSyncDays is nil (all time), sinceTimestamp stays nil
	} else {
		// Regular sync: use the latest notification timestamp
		if state.LatestNotificationAt.Valid {
			t := state.LatestNotificationAt.Time
			sinceTimestamp = &t
		}
		// If no LatestNotificationAt, sinceTimestamp stays nil
	}

	// Build the context with all needed values
	syncCtx := SyncContext{
		UserID:           userID,
		IsSyncConfigured: true,
		IsInitialSync:    isInitialSync,
		SinceTimestamp:   sinceTimestamp,
		MaxCount:         syncSettings.InitialSyncMaxCount,
		UnreadOnly:       syncSettings.InitialSyncUnreadOnly,
	}

	if state.OldestNotificationSyncedAt.Valid {
		syncCtx.OldestNotificationSyncedAt = state.OldestNotificationSyncedAt.Time
	}

	s.logger.Debug("sync context prepared",
		zap.String("userID", userID),
		zap.Bool("isInitialSync", isInitialSync),
		zap.Any("sinceTimestamp", sinceTimestamp),
		zap.Any("maxCount", syncSettings.InitialSyncMaxCount),
		zap.Bool("unreadOnly", syncSettings.InitialSyncUnreadOnly))

	return syncCtx, nil
}

// FetchNotificationsToSync fetches notifications from GitHub using the provided context.
// This method ONLY fetches and filters - it does NOT fetch or update any state.
// The syncCtx parameter MUST come from GetSyncContext().
func (s *Service) FetchNotificationsToSync(
	ctx context.Context,
	syncCtx SyncContext,
) ([]types.NotificationThread, error) {
	// Defensive check - caller should have checked IsSyncConfigured
	if !syncCtx.IsSyncConfigured {
		s.logger.Warn("FetchNotificationsToSync called but sync not configured")
		return []types.NotificationThread{}, nil
	}

	s.logger.Info("fetching notifications from GitHub",
		zap.Any("since", syncCtx.SinceTimestamp),
		zap.Bool("isInitialSync", syncCtx.IsInitialSync))

	// Fetch from GitHub
	// Pass UnreadOnly to control whether to fetch all or only unread notifications
	// Regular sync has no upper bound (before=nil)
	threads, err := s.client.FetchNotifications(
		ctx,
		syncCtx.SinceTimestamp,
		nil,
		syncCtx.UnreadOnly,
	)
	if err != nil {
		s.logger.Error("failed to fetch notifications from GitHub", zap.Error(err))
		return nil, errors.Join(ErrFailedToFetchNotifications, err)
	}

	s.logger.Info("fetched notifications from GitHub",
		zap.Int("count", len(threads)),
		zap.Bool("isInitialSync", syncCtx.IsInitialSync))

	// Apply initial sync limits if this is the initial sync
	if syncCtx.IsInitialSync && (syncCtx.MaxCount != nil || syncCtx.UnreadOnly) {
		threads = applyInitialSyncLimitsFromContext(threads, syncCtx)
		s.logger.Info("applied initial sync limits",
			zap.Int("countAfterLimits", len(threads)))
	}

	return threads, nil
}

// FetchOlderNotificationsToSync fetches notifications older than the specified until time.
// This is used for backfilling older notifications that weren't included in initial sync.
func (s *Service) FetchOlderNotificationsToSync(
	ctx context.Context,
	since time.Time,
	until time.Time,
	maxCount *int,
	unreadOnly bool,
) ([]types.NotificationThread, error) {
	s.logger.Info("fetching older notifications from GitHub",
		zap.Time("since", since),
		zap.Time("until", until),
		zap.Any("maxCount", maxCount),
		zap.Bool("unreadOnly", unreadOnly))

	// Fetch from GitHub with since and before parameters
	// The GitHub API handles the time range filtering for us
	threads, err := s.client.FetchNotifications(ctx, &since, &until, unreadOnly)
	if err != nil {
		s.logger.Error("failed to fetch older notifications from GitHub", zap.Error(err))
		return nil, errors.Join(ErrFailedToFetchNotifications, err)
	}

	s.logger.Info("fetched notifications from GitHub",
		zap.Int("count", len(threads)))

	// Apply max count limit if specified
	if maxCount != nil && len(threads) > *maxCount {
		threads = threads[:*maxCount]
		s.logger.Info("applied max count limit",
			zap.Int("afterMaxCount", len(threads)))
	}

	return threads, nil
}

// UpdateSyncStateAfterProcessing updates the sync state after notifications have been processed.
// This should be called after all notification jobs have been queued or processed.
func (s *Service) UpdateSyncStateAfterProcessing(
	ctx context.Context,
	userID string,
	latestUpdate time.Time,
) error {
	return s.UpdateSyncStateAfterProcessingWithInitialSync(ctx, userID, latestUpdate, nil, nil)
}

// UpdateSyncStateAfterProcessingWithInitialSync updates the sync state with initial sync tracking
func (s *Service) UpdateSyncStateAfterProcessingWithInitialSync(
	ctx context.Context,
	userID string,
	latestUpdate time.Time,
	initialSyncCompletedAt *time.Time,
	oldestNotificationSyncedAt *time.Time,
) error {
	now := s.clock().UTC()
	var latestNotification *time.Time
	if !latestUpdate.IsZero() {
		latest := latestUpdate.UTC()
		latestNotification = &latest
	}

	if _, err := s.syncStateService.UpsertSyncStateWithInitialSync(
		ctx,
		userID,
		&now,
		latestNotification,
		initialSyncCompletedAt,
		oldestNotificationSyncedAt,
	); err != nil {
		s.logger.Error("failed to update sync state", zap.Error(err))
		return errors.Join(ErrFailedToUpdateSyncState, err)
	}

	return nil
}

// IsInitialSyncComplete checks if the initial sync has been completed
func (s *Service) IsInitialSyncComplete(ctx context.Context, userID string) (bool, error) {
	state, err := s.syncStateService.GetSyncState(ctx, userID)
	if err != nil {
		return false, err
	}
	return state.InitialSyncCompletedAt.Valid, nil
}

// ProcessNotification handles the complete processing of a single notification.
// This includes upserting the repository, fetching subject details, and upserting the notification.
func (s *Service) ProcessNotification(
	ctx context.Context,
	userID string,
	thread types.NotificationThread,
) error {
	// Upsert repository
	rawRepo := thread.Repository.Raw()
	repoParams := db.UpsertRepositoryParams{
		GithubID:       models.SQLNullInt64(thread.Repository.ID),
		NodeID:         models.SQLNullString(thread.Repository.NodeID),
		Name:           thread.Repository.Name,
		FullName:       thread.Repository.FullName,
		OwnerLogin:     models.SQLNullString(thread.Repository.Owner.Login),
		OwnerID:        models.SQLNullInt64(thread.Repository.Owner.ID),
		OwnerAvatarURL: models.SQLNullString(thread.Repository.Owner.AvatarURL),
		OwnerHTMLURL:   models.SQLNullString(thread.Repository.Owner.HTMLURL),
		Private:        models.SQLNullBoolPtr(&thread.Repository.Private),
		Description:    models.SQLNullStringPtr(thread.Repository.Description),
		HTMLURL:        models.SQLNullString(thread.Repository.HTMLURL),
		Fork:           models.SQLNullBoolPtr(&thread.Repository.Fork),
		Visibility:     models.SQLNullStringPtr(thread.Repository.Visibility),
		DefaultBranch:  models.SQLNullStringPtr(thread.Repository.DefaultBranch),
		Archived:       thread.Repository.Archived,
		Disabled:       models.SQLNullBoolPtr(&thread.Repository.Disabled),
		PushedAt:       models.SQLNullTime(thread.Repository.PushedAt),
		CreatedAt:      models.SQLNullTime(thread.Repository.CreatedAt),
		UpdatedAt:      models.SQLNullTime(thread.Repository.UpdatedAt),
		Raw: db.NullRawMessage{
			RawMessage: rawRepo,
			Valid:      len(rawRepo) > 0,
		},
	}

	repo, err := s.repositoryService.UpsertRepository(ctx, userID, repoParams)
	if err != nil {
		s.logger.Error(
			"failed to upsert repository",
			zap.String("fullName", thread.Repository.FullName),
			zap.Error(err),
		)
		return errors.Join(ErrFailedToUpsertRepository, err)
	}

	// Fetch subject details
	var (
		subjectPayload   db.NullRawMessage
		subjectFetchedAt sql.NullTime
	)

	if rawSubject, err := s.client.FetchSubjectRaw(ctx, thread.Subject.URL); err == nil &&
		len(rawSubject) > 0 {
		subjectPayload = db.NullRawMessage{
			RawMessage: rawSubject,
			Valid:      true,
		}
		fetched := s.clock().UTC()
		subjectFetchedAt = models.SQLNullTime(&fetched)
	} else if err != nil {
		// Log but don't fail - subject fetch is optional
		//nolint:lll // Long warning message with multiple zap fields
		s.logger.Warn("failed to fetch subject data (continuing without it)", zap.String("githubID", thread.ID), zap.String("subjectURL", thread.Subject.URL), zap.Error(err))
	}

	// Process pull request metadata if subject is a PullRequest
	var pullRequestID sql.NullInt64
	if strings.EqualFold(thread.Subject.Type, "PullRequest") && subjectPayload.Valid {
		if pr, err := s.upsertPullRequestFromSubject(ctx, userID, repo.ID, subjectPayload.RawMessage); err == nil &&
			pr != nil {
			pullRequestID = sql.NullInt64{Int64: pr.ID, Valid: true}
		} else if err != nil {
			// Log but don't fail - PR metadata is optional
			//nolint:lll // Long warning message with multiple zap fields
			s.logger.Warn("failed to upsert pull request metadata (continuing without it)", zap.String("githubID", thread.ID), zap.Int64("repoID", repo.ID), zap.Error(err))
		}
	}

	// Extract author and subject metadata from subject data
	var authorLogin sql.NullString
	var authorID sql.NullInt64
	var subjectNumber sql.NullInt32
	var subjectState sql.NullString
	var subjectMerged sql.NullBool
	var subjectStateReason sql.NullString
	if subjectPayload.Valid {
		authorLogin, authorID = github.ExtractAuthorFromSubject(subjectPayload.RawMessage)
		subjectNumber = github.ExtractSubjectNumber(subjectPayload.RawMessage)
		subjectState = github.ExtractSubjectState(subjectPayload.RawMessage)
		subjectMerged = github.ExtractSubjectMerged(subjectPayload.RawMessage)
		subjectStateReason = github.ExtractSubjectStateReason(subjectPayload.RawMessage)
	}

	// Upsert notification
	notificationParams := db.UpsertNotificationParams{
		GithubID:                thread.ID,
		RepositoryID:            repo.ID,
		PullRequestID:           pullRequestID,
		SubjectType:             thread.Subject.Type,
		SubjectTitle:            thread.Subject.Title,
		SubjectURL:              models.SQLNullString(thread.Subject.URL),
		SubjectLatestCommentURL: models.SQLNullString(thread.Subject.LatestCommentURL),
		Reason:                  models.SQLNullString(thread.Reason),
		GithubUnread:            models.SQLNullBoolPtr(&thread.Unread),
		GithubUpdatedAt:         models.SQLNullTime(&thread.UpdatedAt),
		GithubLastReadAt:        models.SQLNullTime(thread.LastReadAt),
		GithubURL:               models.SQLNullString(thread.URL),
		GithubSubscriptionURL:   models.SQLNullString(thread.SubscriptionURL),
		Payload: db.NullRawMessage{
			RawMessage: thread.Raw,
			Valid:      len(thread.Raw) > 0,
		},
		SubjectRaw:         subjectPayload,
		SubjectFetchedAt:   subjectFetchedAt,
		AuthorLogin:        authorLogin,
		AuthorID:           authorID,
		SubjectNumber:      subjectNumber,
		SubjectState:       subjectState,
		SubjectMerged:      subjectMerged,
		SubjectStateReason: subjectStateReason,
	}

	if _, err := s.notificationService.UpsertNotification(ctx, userID, notificationParams); err != nil {
		s.logger.Error(
			"failed to upsert notification",
			zap.String("githubID", thread.ID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// upsertPullRequestFromSubject extracts PR data from subject JSON and upserts it to the database.
func (s *Service) upsertPullRequestFromSubject(
	ctx context.Context,
	userID string,
	repoID int64,
	subjectJSON json.RawMessage,
) (*db.PullRequest, error) {
	prData, err := github.ExtractPullRequestData(subjectJSON)
	if err != nil {
		s.logger.Error("failed to extract pull request data", zap.Error(err))
		return nil, errors.Join(ErrFailedToExtractPullRequestData, err)
	}

	// Convert extracted data to database params
	params := db.UpsertPullRequestParams{
		RepositoryID: repoID,
		GithubID:     models.SQLNullInt64Ptr(prData.GithubID),
		NodeID:       models.SQLNullStringPtr(prData.NodeID),

		Number:      int32(prData.Number),
		Title:       models.SQLNullStringPtr(prData.Title),
		State:       models.SQLNullStringPtr(prData.State),
		Draft:       models.SQLNullBoolPtr(prData.Draft),
		Merged:      models.SQLNullBoolPtr(prData.Merged),
		AuthorLogin: models.SQLNullStringPtr(prData.AuthorLogin),
		AuthorID:    models.SQLNullInt64Ptr(prData.AuthorID),
		CreatedAt:   models.SQLNullTime(prData.CreatedAt),
		UpdatedAt:   models.SQLNullTime(prData.UpdatedAt),
		ClosedAt:    models.SQLNullTime(prData.ClosedAt),
		MergedAt:    models.SQLNullTime(prData.MergedAt),
		Raw: db.NullRawMessage{
			RawMessage: subjectJSON,
			Valid:      true,
		},
	}

	pr, err := s.pullRequestService.UpsertPullRequest(ctx, userID, params)
	if err != nil {
		s.logger.Error(
			"failed to upsert pull request",
			zap.Int64("repoID", repoID),
			zap.Int("number", prData.Number),
			zap.Error(err),
		)
		return nil, err
	}

	return &pr, nil
}

// RefreshSubjectData fetches fresh subject data from GitHub and updates the notification
func (s *Service) RefreshSubjectData(ctx context.Context, userID string, githubID string) error {
	// Get the notification
	notification, err := s.notificationService.GetByGithubID(ctx, userID, githubID)
	if err != nil {
		s.logger.Error(
			"failed to get notification",
			zap.String("githubID", githubID),
			zap.Error(err),
		)
		return err
	}

	// Skip refresh for CI activity and Discussions (no API endpoint available)
	normalizedType := strings.ToLower(strings.ReplaceAll(notification.SubjectType, "_", ""))
	if normalizedType == "checkrun" || normalizedType == "checksuite" ||
		normalizedType == "discussion" {
		s.logger.Debug(
			"skipping subject refresh for unsupported type",
			zap.String("githubID", githubID),
			zap.String("subjectType", notification.SubjectType),
		)
		return ErrNotificationMissingSubjectURL // Return same error as missing URL for consistency
	}

	if !notification.SubjectURL.Valid || notification.SubjectURL.String == "" {
		s.logger.Warn("notification has no subject URL", zap.String("githubID", githubID))
		return ErrNotificationMissingSubjectURL
	}

	// Fetch fresh subject data
	subjectRaw, err := s.client.FetchSubjectRaw(ctx, notification.SubjectURL.String)
	if err != nil {
		s.logger.Error(
			"failed to fetch subject from GitHub",
			zap.String("githubID", githubID),
			zap.String("subjectURL", notification.SubjectURL.String),
			zap.Error(err),
		)
		return errors.Join(ErrFailedToFetchSubject, err)
	}

	// Update the notification with fresh subject data
	subjectPayload := db.NullRawMessage{
		RawMessage: subjectRaw,
		Valid:      len(subjectRaw) > 0,
	}
	fetched := s.clock().UTC()
	subjectFetchedAt := models.SQLNullTime(&fetched)

	// If it's a PR, update the pull_request table too
	var pullRequestID sql.NullInt64
	if strings.EqualFold(notification.SubjectType, "PullRequest") && subjectPayload.Valid {
		repo, repoErr := s.repositoryService.GetRepositoryByID(
			ctx,
			userID,
			notification.RepositoryID,
		)
		if repoErr != nil {
			s.logger.Error(
				"failed to get repository",
				zap.Int64("repositoryID", notification.RepositoryID),
				zap.Error(repoErr),
			)
			return errors.Join(ErrFailedToGetRepository, repoErr)
		}

		if pr, prErr := s.upsertPullRequestFromSubject(ctx, userID, repo.ID, subjectPayload.RawMessage); prErr == nil &&
			pr != nil {
			pullRequestID = sql.NullInt64{Int64: pr.ID, Valid: true}
		} else if prErr != nil {
			// Log but don't fail - PR metadata update is optional
			s.logger.Warn(
				"failed to upsert pull request metadata during refresh (continuing without it)",
				zap.String("githubID", githubID),
				zap.Int64("repoID", repo.ID),
				zap.Error(prErr),
			)
		}
	}

	// Extract subject metadata from fresh subject data
	var subjectNumber sql.NullInt32
	var subjectState sql.NullString
	var subjectMerged sql.NullBool
	var subjectStateReason sql.NullString
	if subjectPayload.Valid {
		subjectNumber = github.ExtractSubjectNumber(subjectPayload.RawMessage)
		subjectState = github.ExtractSubjectState(subjectPayload.RawMessage)
		subjectMerged = github.ExtractSubjectMerged(subjectPayload.RawMessage)
		subjectStateReason = github.ExtractSubjectStateReason(subjectPayload.RawMessage)
	}

	// Update the notification with the fresh subject data
	err = s.notificationService.UpdateNotificationSubject(
		ctx,
		userID,
		db.UpdateNotificationSubjectParams{
			GithubID:           githubID,
			SubjectRaw:         subjectPayload,
			SubjectFetchedAt:   subjectFetchedAt,
			PullRequestID:      pullRequestID,
			SubjectNumber:      subjectNumber,
			SubjectState:       subjectState,
			SubjectMerged:      subjectMerged,
			SubjectStateReason: subjectStateReason,
		},
	)
	if err != nil {
		s.logger.Error(
			"failed to update notification subject",
			zap.String("githubID", githubID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// ProcessNotificationData processes a notification from JSON data.
// This is a convenience method that unmarshals the JSON and calls ProcessNotification.
func (s *Service) ProcessNotificationData(ctx context.Context, userID string, data []byte) error {
	var thread types.NotificationThread
	if err := json.Unmarshal(data, &thread); err != nil {
		return err
	}
	return s.ProcessNotification(ctx, userID, thread)
}
