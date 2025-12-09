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

package db

import (
	"context"
	"database/sql"
)

//go:generate mockgen -source=store.go -destination=mocks/mock_store.go -package=mocks

// Store defines the interface for database queries used by the business logic layer.
// This interface allows us to mock database operations in unit tests.
// All methods that access user data require a userID parameter for multi-user support.
type Store interface {
	// Notification methods
	GetNotificationByGithubID(ctx context.Context, userID, githubID string) (Notification, error)
	GetNotificationByID(ctx context.Context, userID string, id int64) (Notification, error)
	ListNotificationsFromQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (ListNotificationsFromQueryResult, error)
	MarkNotificationRead(ctx context.Context, userID, githubID string) (Notification, error)
	MarkNotificationUnread(ctx context.Context, userID, githubID string) (Notification, error)
	ArchiveNotification(ctx context.Context, userID, githubID string) (Notification, error)
	UnarchiveNotification(ctx context.Context, userID, githubID string) (Notification, error)
	MuteNotification(ctx context.Context, userID, githubID string) (Notification, error)
	UnmuteNotification(ctx context.Context, userID, githubID string) (Notification, error)
	SnoozeNotification(
		ctx context.Context,
		userID string,
		arg SnoozeNotificationParams,
	) (Notification, error)
	UnsnoozeNotification(ctx context.Context, userID, githubID string) (Notification, error)
	StarNotification(ctx context.Context, userID, githubID string) (Notification, error)
	UnstarNotification(ctx context.Context, userID, githubID string) (Notification, error)
	MarkNotificationFiltered(ctx context.Context, userID, githubID string) (Notification, error)
	MarkNotificationUnfiltered(ctx context.Context, userID, githubID string) (Notification, error)
	BulkSnoozeNotifications(
		ctx context.Context,
		userID string,
		arg BulkSnoozeNotificationsParams,
	) (int64, error)

	BulkMarkNotificationsRead(ctx context.Context, userID string, githubIDs []string) (int64, error)

	BulkMarkNotificationsUnread(
		ctx context.Context,
		userID string,
		githubIDs []string,
	) (int64, error)

	BulkArchiveNotifications(ctx context.Context, userID string, githubIDs []string) (int64, error)

	BulkUnarchiveNotifications(
		ctx context.Context,
		userID string,
		githubIDs []string,
	) (int64, error)
	BulkMarkNotificationsReadByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)
	BulkMarkNotificationsUnreadByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)
	BulkArchiveNotificationsByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)
	BulkUnarchiveNotificationsByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)
	BulkSnoozeNotificationsByQuery(
		ctx context.Context,
		userID string,
		arg BulkSnoozeNotificationsByQueryParams,
	) (int64, error)

	BulkUnsnoozeNotifications(ctx context.Context, userID string, githubIDs []string) (int64, error)
	BulkUnsnoozeNotificationsByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)

	BulkMuteNotifications(ctx context.Context, userID string, githubIDs []string) (int64, error)

	BulkUnmuteNotifications(ctx context.Context, userID string, githubIDs []string) (int64, error)
	BulkMuteNotificationsByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)
	BulkUnmuteNotificationsByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)

	BulkStarNotifications(ctx context.Context, userID string, githubIDs []string) (int64, error)

	BulkUnstarNotifications(ctx context.Context, userID string, githubIDs []string) (int64, error)
	BulkStarNotificationsByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)
	BulkUnstarNotificationsByQuery(
		ctx context.Context,
		userID string,
		query NotificationQuery,
	) (int64, error)

	BulkMarkNotificationsUnfiltered(
		ctx context.Context,
		userID string,
		githubIDs []string,
	) (int64, error)

	// Tag methods (IDs are now UUIDs/strings)
	GetTag(ctx context.Context, userID, id string) (Tag, error)
	GetTagByName(ctx context.Context, userID, name string) (Tag, error)
	ListAllTags(ctx context.Context, userID string) ([]Tag, error)
	UpsertTag(ctx context.Context, userID string, arg UpsertTagParams) (Tag, error)
	UpdateTag(ctx context.Context, userID string, arg UpdateTagParams) (Tag, error)
	DeleteTag(ctx context.Context, userID, id string) error
	UpdateTagDisplayOrder(ctx context.Context, userID string, arg UpdateTagDisplayOrderParams) error
	ListTagsForEntity(
		ctx context.Context,
		userID string,
		arg ListTagsForEntityParams,
	) ([]Tag, error)
	AssignTagToEntity(
		ctx context.Context,
		userID string,
		arg AssignTagToEntityParams,
	) (TagAssignment, error)
	RemoveTagAssignment(ctx context.Context, userID string, arg RemoveTagAssignmentParams) error

	// View methods (IDs are now UUIDs/strings)
	GetView(ctx context.Context, userID, id string) (View, error)
	ListViews(ctx context.Context, userID string) ([]View, error)
	CreateView(ctx context.Context, userID string, arg CreateViewParams) (View, error)
	UpdateView(ctx context.Context, userID string, arg UpdateViewParams) (View, error)
	DeleteView(ctx context.Context, userID, id string) (int64, error)
	UpdateViewOrder(ctx context.Context, userID string, arg UpdateViewOrderParams) error
	GetRulesByViewID(ctx context.Context, userID string, viewID sql.NullString) ([]Rule, error)

	// Rule methods (IDs are now UUIDs/strings)
	GetRule(ctx context.Context, userID, id string) (Rule, error)
	ListRules(ctx context.Context, userID string) ([]Rule, error)
	ListEnabledRulesOrdered(ctx context.Context, userID string) ([]Rule, error)
	CreateRule(ctx context.Context, userID string, arg CreateRuleParams) (Rule, error)
	UpdateRule(ctx context.Context, userID string, arg UpdateRuleParams) (Rule, error)
	DeleteRule(ctx context.Context, userID, id string) error
	UpdateRuleOrder(ctx context.Context, userID string, arg UpdateRuleOrderParams) error

	// Repository methods
	GetRepositoryByID(ctx context.Context, userID string, id int64) (Repository, error)
	ListRepositories(ctx context.Context, userID string) ([]Repository, error)
	UpsertRepository(
		ctx context.Context,
		userID string,
		arg UpsertRepositoryParams,
	) (Repository, error)

	// Pull Request methods
	UpsertPullRequest(
		ctx context.Context,
		userID string,
		arg UpsertPullRequestParams,
	) (PullRequest, error)

	// Sync methods
	GetSyncState(ctx context.Context, userID string) (GetSyncStateRow, error)
	UpsertSyncState(
		ctx context.Context,
		userID string,
		arg UpsertSyncStateParams,
	) (UpsertSyncStateRow, error)

	// Notification upsert/update methods
	UpsertNotification(
		ctx context.Context,
		userID string,
		arg UpsertNotificationParams,
	) (Notification, error)
	UpdateNotificationSubject(
		ctx context.Context,
		userID string,
		arg UpdateNotificationSubjectParams,
	) error

	// User methods (no userID param - these operate on the single user record)
	GetUser(ctx context.Context) (User, error)
	CreateUser(ctx context.Context) (User, error)
	UpdateUserGitHubIdentity(ctx context.Context, arg UpdateUserGitHubIdentityParams) (User, error)
	UpdateUserSyncSettings(ctx context.Context, syncSettings NullRawMessage) (User, error)
	UpdateUserGitHubToken(ctx context.Context, githubTokenEncrypted sql.NullString) (User, error)
	ClearUserGitHubToken(ctx context.Context) (User, error)
	UpdateUserRetentionSettings(ctx context.Context, retentionSettings sql.NullString) (User, error)
	UpdateUserUpdateSettings(ctx context.Context, updateSettings NullRawMessage) (User, error)
	UpdateUserMutedUntil(ctx context.Context, mutedUntil sql.NullTime) (User, error)

	// Storage management methods
	GetStorageStats(ctx context.Context, userID string) (StorageStats, error)
	CountEligibleForCleanup(ctx context.Context, userID string, params CleanupParams) (int64, error)
	DeleteOldArchivedNotifications(
		ctx context.Context,
		userID string,
		params CleanupParams,
	) (int64, error)
	DeleteOrphanedPullRequests(ctx context.Context, userID string) (int64, error)
	DeleteAllGitHubData(ctx context.Context, userID string) error
}
