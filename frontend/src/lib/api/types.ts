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

export type NotificationState = "open" | "closed" | "merged" | "draft";

export type NotificationTargetType = "issue" | "pull_request" | string;

export interface Tag {
	id: string;
	name: string;
	color?: string;
	description?: string;
	slug: string;
	unreadCount?: number;
}

export interface BackendRepositoryResponse {
	id: number;
	githubId?: number | null;
	nodeId?: string | null;
	name: string;
	fullName: string;
	ownerLogin?: string | null;
	ownerId?: number | null;
	private?: boolean | null;
	description?: string | null;
	htmlUrl?: string | null;
	fork?: boolean | null;
	visibility?: string | null;
	defaultBranch?: string | null;
	archived?: boolean | null;
	disabled?: boolean | null;
	pushedAt?: string | null;
	createdAt?: string | null;
	updatedAt?: string | null;
	raw?: unknown;
	ownerAvatarUrl?: string | null;
	ownerHtmlUrl?: string | null;
}

export interface BackendNotificationResponse {
	id: number;
	githubId: string;
	repositoryId: number;
	pullRequestId?: number | null;
	subjectType: string;
	subjectTitle: string;
	subjectUrl?: string | null;
	subjectLatestCommentUrl?: string | null;
	reason?: string | null;
	archived: boolean;
	isRead: boolean;
	muted: boolean;
	starred: boolean;
	filtered: boolean;
	snoozedUntil?: string | null;
	snoozedAt?: string | null;
	effectiveSortDate: string;
	githubUnread?: boolean | null;
	githubUpdatedAt?: string | null;
	githubLastReadAt?: string | null;
	githubUrl?: string | null;
	githubSubscriptionUrl?: string | null;
	importedAt: string;
	payload?: unknown;
	repository?: BackendRepositoryResponse | null;
	subjectRaw?: unknown;
	subjectFetchedAt?: string | null;
	subjectNumber?: number | null;
	subjectState?: string | null;
	subjectMerged?: boolean | null;
	subjectStateReason?: string | null;
	actionHints?: ActionHints;
	tags?: Tag[];
	authorLogin?: string | null;
}

export interface ActionHints {
	dismissedOn: string[];
}

export interface Notification {
	id: string;
	githubId?: string;
	repoFullName: string;
	subjectTitle: string;
	subjectType: NotificationTargetType;
	reason: string;
	archived: boolean;
	isRead: boolean;
	muted: boolean;
	starred: boolean;
	filtered?: boolean;
	snoozedUntil?: string;
	snoozedAt?: string;
	updatedAt: string;
	labels: string[];
	viewIds: string[];
	repositoryId?: number;
	state?: NotificationState;
	authorLogin?: string;
	githubUrl?: string;
	htmlUrl?: string;
	subjectUrl?: string;
	subjectLatestCommentUrl?: string;
	importedAt?: string;
	repository?: BackendRepositoryResponse | null;
	subjectRaw?: unknown;
	subjectFetchedAt?: string;
	subjectNumber?: number;
	subjectState?: string;
	subjectMerged?: boolean;
	subjectStateReason?: string;
	actionHints?: ActionHints;
	tags?: Tag[];
	effectiveSortDate?: string;
}

export type ViewFilterOperator = "equals" | "contains" | "does not equal" | "does not contain";

export interface NotificationViewFilter {
	field: string;
	operator: ViewFilterOperator;
	value: string;
}

export interface NotificationView {
	id: string;
	name: string;
	slug: string;
	description?: string;
	icon?: string;
	isDefault?: boolean;
	systemView?: boolean;
	query: string; // New: query string instead of filters array
	unreadCount: number;
	displayOrder?: number;
}

export interface NotificationViewInput {
	name: string;
	description?: string;
	icon?: string;
	query: string; // New: query string instead of filters array
}

export interface NotificationViewDraft {
	id?: string;
	name: string;
	description: string;
	icon?: string;
	query: string; // New: query string instead of filters array
}

export interface NotificationFilters {
	query?: string; // Combined query string with key-value pairs and free text (including status filtering)
	filters?: NotificationViewFilter[]; // Deprecated: kept for backward compatibility
	repoName?: string;
	reason?: string;
}

export interface NotificationPage {
	items: Notification[];
	total: number;
	pageSize: number;
	page: number;
}

export interface NotificationTarget {
	type: NotificationTargetType;
	title: string;
	state: NotificationState;
	url: string;
	lastActivity: string;
}

export interface NotificationSubjectSummary {
	title?: string;
	url?: string; // html_url from GitHub API
	state?: string;
	author?: string;
	body?: string;
	createdAt?: string;
	updatedAt?: string;
	number?: number;
	labels?: string[];
	draft?: boolean;
	merged?: boolean;
	mergedAt?: string;
	comments?: number;
	reviewComments?: number; // For PRs: number of review comments
	assignees?: string[];
	milestone?: string;
	reviewDecision?: string;
	closedAt?: string;
}

export interface NotificationDetail {
	notification: Notification;
	subject: NotificationSubjectSummary | null;
}

export interface BackendNotificationDetailResponse {
	notification: BackendNotificationResponse;
}

export interface ViewCountsResponse {
	viewId: string;
	slug?: string;
	systemView?: boolean;
	counts: Record<string, number>;
}

export interface AllViewCountsResponse {
	views: ViewCountsResponse[];
}

export interface NotificationComment {
	type: "comment";
	id: number;
	body: string;
	author: {
		login: string;
		avatarUrl: string;
	};
	createdAt: string;
	updatedAt: string;
	htmlUrl: string;
}

export interface NotificationReview {
	type: "review";
	id: number;
	state: "APPROVED" | "CHANGES_REQUESTED" | "COMMENTED" | "DISMISSED" | "PENDING";
	body: string;
	author: {
		login: string;
		avatarUrl: string;
	};
	submittedAt: string;
	htmlUrl: string;
}

export type TimelineEventType =
	| "comment" // Issue comment (also called "commented" in timeline)
	| "reviewed" // PR review
	| "committed" // Commit
	| "merged" // PR merged
	| "closed" // Issue/PR closed
	| "reopened" // Issue/PR reopened
	| "assigned" // Assignee added
	| "unassigned" // Assignee removed
	| "review_requested" // Review requested
	| "review_request_removed" // Review request removed
	| "ready_for_review" // Draft PR marked ready
	| "converted_to_draft" // PR converted to draft
	| "deployed" // Deployment
	| "referenced" // Referenced from another issue/PR
	| "cross-referenced" // Cross-referenced
	| "labeled" // Label added
	| "unlabeled" // Label removed
	| "milestoned" // Milestone added
	| "demilestoned" // Milestone removed
	| "renamed" // Title changed
	| "locked" // Conversation locked
	| "unlocked" // Conversation unlocked
	| "head_ref_deleted" // PR head branch deleted
	| "head_ref_restored" // PR head branch restored
	| "connected" // Connected to another issue
	| "disconnected" // Disconnected from another issue
	| "added_to_project_v2" // Added to project
	| "removed_from_project_v2" // Removed from project
	| "project_v2_item_status_changed" // Project status changed
	| string; // Fallback for unknown events

export interface NotificationTimelineItem {
	type: TimelineEventType;
	id: number | string;
	timestamp?: string;
	actor?: {
		login: string;
		avatarUrl: string;
	};
	// For comments and reviews
	body?: string;
	author?: {
		login: string;
		avatarUrl: string;
	};
	// For reviews
	state?: "APPROVED" | "CHANGES_REQUESTED" | "COMMENTED" | "DISMISSED" | "PENDING";
	// For commits
	message?: string;
	sha?: string;
	// Common fields
	htmlUrl?: string;
	createdAt?: string;
	updatedAt?: string;
	submittedAt?: string;
}

export type NotificationThreadItem =
	| NotificationComment
	| NotificationReview
	| NotificationTimelineItem;

export interface NotificationCommentsResponse {
	items: NotificationThreadItem[];
	total: number;
	page: number;
	perPage: number;
	hasMore: boolean;
}

export interface NotificationTimelineResponse {
	items: NotificationTimelineItem[];
	total: number;
	page: number;
	perPage: number;
	hasMore: boolean;
}
