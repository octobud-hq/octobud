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

import type {
	BackendNotificationDetailResponse,
	BackendNotificationResponse,
	Notification,
	NotificationDetail,
	NotificationFilters,
	NotificationPage,
	NotificationSubjectSummary,
	NotificationViewFilter,
	NotificationTimelineResponse,
} from "./types";
import { constructGitHubHtmlUrl } from "$lib/utils/githubUrls";
import { fetchWithAuth, buildApiUrl, ApiUnreachableError, isProxyConnectionError } from "./fetch";

const PAGE_SIZE = 30;

export interface FetchNotificationsParams {
	page?: number;
	pageSize?: number;
	filters?: Partial<NotificationFilters>;
}

const normalizeSubjectType = (subjectType: string): string => {
	const normalized = subjectType.toLowerCase();
	if (normalized === "pullrequest" || normalized === "pull_request") {
		return "pull_request";
	}
	if (normalized === "issue") {
		return "issue";
	}
	return normalized;
};

const fromBackendNotification = (notification: BackendNotificationResponse): Notification => {
	const repoFullName =
		notification.repository?.fullName ?? notification.repository?.name ?? "Unknown repository";

	// Parse subject summary to extract html_url and other metadata
	const subject = parseSubjectSummary(notification.subjectRaw);

	// Compute htmlUrl with fallback logic:
	// 1. Use html_url from parsed subject (extracted from subject_raw)
	// 2. Try to construct from repo name and subject type
	// Note: githubUrl and subjectUrl are API URLs, not suitable for display
	const htmlUrl =
		subject?.url ??
		constructGitHubHtmlUrl(
			repoFullName,
			notification.subjectType,
			notification.subjectUrl ?? undefined,
			subject?.number
		) ??
		undefined;

	return {
		id: String(notification.id),
		githubId: notification.githubId,
		repoFullName,
		subjectTitle: notification.subjectTitle,
		subjectType: normalizeSubjectType(notification.subjectType),
		reason: notification.reason ?? "unspecified",
		archived: notification.archived,
		isRead: notification.isRead,
		muted: notification.muted,
		starred: notification.starred,
		filtered: notification.filtered ?? false,
		snoozedUntil: notification.snoozedUntil ?? undefined,
		snoozedAt: notification.snoozedAt ?? undefined,
		updatedAt: notification.githubUpdatedAt ?? notification.importedAt,
		labels: [],
		viewIds: ["inbox"],
		repositoryId: notification.repositoryId,
		state: subject?.state as any, // Use actual state from parsed subject
		authorLogin: notification.authorLogin ?? undefined,
		githubUrl: notification.githubUrl ?? undefined,
		htmlUrl,
		subjectUrl: notification.subjectUrl ?? undefined,
		subjectLatestCommentUrl: notification.subjectLatestCommentUrl ?? undefined,
		importedAt: notification.importedAt,
		repository: notification.repository ?? null,
		subjectRaw: notification.subjectRaw,
		subjectFetchedAt: notification.subjectFetchedAt ?? undefined,
		subjectNumber: notification.subjectNumber ?? undefined,
		subjectState: notification.subjectState ?? undefined,
		subjectMerged: notification.subjectMerged ?? undefined,
		subjectStateReason: notification.subjectStateReason ?? undefined,
		actionHints: notification.actionHints,
		tags: notification.tags ?? [],
		effectiveSortDate: notification.effectiveSortDate,
	};
};

export const parseSubjectSummary = (raw: unknown): NotificationSubjectSummary | null => {
	if (!raw || typeof raw !== "object") {
		return null;
	}

	const data = raw as Record<string, unknown>;

	const lookup = <T extends string>(key: string): T | undefined => {
		const value = data[key];
		return typeof value === "string" ? (value as T) : undefined;
	};

	const lookupNumber = (key: string): number | undefined => {
		const value = data[key];
		return typeof value === "number" ? value : undefined;
	};

	const lookupBoolean = (key: string): boolean | undefined => {
		const value = data[key];
		return typeof value === "boolean" ? value : undefined;
	};

	const nestedLogin = (key: string): string | undefined => {
		const nested = data[key];
		if (nested && typeof nested === "object") {
			const login = (nested as Record<string, unknown>).login;
			if (typeof login === "string") {
				return login;
			}
		}
		return undefined;
	};

	const nestedTitle = (key: string): string | undefined => {
		const nested = data[key];
		if (nested && typeof nested === "object") {
			const title = (nested as Record<string, unknown>).title;
			if (typeof title === "string") {
				return title;
			}
		}
		return undefined;
	};

	const extractLabels = (): string[] | undefined => {
		const labels = data.labels;
		if (Array.isArray(labels)) {
			return labels
				.map((label) => {
					if (typeof label === "string") return label;
					if (label && typeof label === "object") {
						const name = (label as Record<string, unknown>).name;
						return typeof name === "string" ? name : null;
					}
					return null;
				})
				.filter((name): name is string => name !== null);
		}
		return undefined;
	};

	const extractAssignees = (): string[] | undefined => {
		const assignees = data.assignees;
		if (Array.isArray(assignees)) {
			return assignees
				.map((assignee) => {
					if (assignee && typeof assignee === "object") {
						const login = (assignee as Record<string, unknown>).login;
						return typeof login === "string" ? login : null;
					}
					return null;
				})
				.filter((login): login is string => login !== null);
		}
		return undefined;
	};

	return {
		title: lookup<string>("title"),
		url: lookup<string>("html_url") ?? lookup<string>("url"),
		state: lookup<string>("state"),
		author: nestedLogin("user") ?? nestedLogin("owner"),
		body: lookup<string>("body"),
		createdAt: lookup<string>("created_at") ?? lookup<string>("createdAt"),
		updatedAt: lookup<string>("updated_at") ?? lookup<string>("updatedAt"),
		number: lookupNumber("number"),
		labels: extractLabels(),
		draft: lookupBoolean("draft"),
		merged: lookupBoolean("merged"),
		mergedAt: lookup<string>("merged_at"),
		comments: lookupNumber("comments"),
		reviewComments: lookupNumber("review_comments"),
		assignees: extractAssignees(),
		milestone: nestedTitle("milestone"),
		reviewDecision: lookup<string>("review_decision"),
		closedAt: lookup<string>("closed_at"),
	};
};

const buildFilterParams = (filters: NotificationViewFilter[] = []): NotificationViewFilter[] =>
	filters
		.map((filter) => ({
			field: filter.field?.trim() ?? "",
			operator: filter.operator,
			value: filter.value?.trim() ?? "",
		}))
		.filter(
			(filter) => filter.field.length > 0 && filter.operator.length > 0 && filter.value.length > 0
		);

export async function fetchNotifications(
	params: FetchNotificationsParams = {},
	fetchImpl?: typeof fetch
): Promise<NotificationPage> {
	const { page = 1, pageSize = PAGE_SIZE, filters = {} } = params;

	const searchParams = new URLSearchParams();
	searchParams.set("page", String(page));
	searchParams.set("pageSize", String(pageSize));

	// Use combined query string if provided (includes key-value pairs, free text, and status filtering)
	// Always send query parameter (even if empty) to ensure new query engine is used with inbox defaults
	if (filters.query !== undefined) {
		searchParams.set("query", filters.query);
	} else {
		// Fall back to structured filters for backward compatibility
		const structuredFilters = buildFilterParams(filters.filters);
		if (structuredFilters.length > 0) {
			searchParams.set("filters", JSON.stringify(structuredFilters));
		}
	}

	const url = `/api/notifications?${searchParams.toString()}`;
	const response = await fetchWithAuth(url, {}, fetchImpl);
	if (!response.ok) {
		// Try to extract error message from response
		let errorMessage = `Failed to load notifications (${response.status})`;
		let responseText = "";
		let isValidJson = false;
		try {
			// First get raw text to check for proxy errors
			responseText = await response.text();
			// Try to parse as JSON
			const errorData: { error?: string } = JSON.parse(responseText);
			isValidJson = true;
			if (errorData.error) {
				errorMessage = errorData.error;
			}
		} catch {
			// If JSON parsing fails, responseText might contain proxy error
			if (responseText) {
				errorMessage = responseText;
			}
		}

		// Check if this is a proxy connection error (backend unreachable)
		// This happens when Vite proxy can't connect to the backend and returns 500/502/503
		// Also treat empty responses or non-JSON responses on 500 as proxy errors
		const isProxyError =
			(response.status === 500 || response.status === 502 || response.status === 503) &&
			(!responseText || !isValidJson || isProxyConnectionError(responseText || errorMessage));

		if (isProxyError) {
			throw new ApiUnreachableError();
		}

		const error = new Error(errorMessage) as Error & { statusCode?: number };
		error.statusCode = response.status;
		throw error;
	}

	const payload: {
		notifications: BackendNotificationResponse[];
		total?: number;
		page?: number;
		pageSize?: number;
	} = await response.json();
	const notifications = (payload.notifications ?? []).map(fromBackendNotification);

	return {
		items: notifications,
		total: payload.total ?? notifications.length,
		pageSize: payload.pageSize ?? pageSize,
		page: payload.page ?? page,
	};
}

export interface FetchNotificationDetailOptions {
	fetch?: typeof fetch;
	fallback?: Notification;
	query?: string;
}

export async function fetchNotificationDetail(
	githubId: string,
	options: FetchNotificationDetailOptions = {}
): Promise<NotificationDetail> {
	let url = `/api/notifications/${encodeURIComponent(githubId)}`;
	if (options.query) {
		const separator = url.includes("?") ? "&" : "?";
		url = `${url}${separator}query=${encodeURIComponent(options.query)}`;
	}
	const response = await fetchWithAuth(url, {}, options.fetch);
	if (!response.ok) {
		if (response.status === 404 && options.fallback) {
			return {
				notification: options.fallback,
				subject: parseSubjectSummary(options.fallback.subjectRaw),
			};
		}
		throw new Error(`Failed to load notification (${response.status})`);
	}

	const payload: BackendNotificationDetailResponse = await response.json();
	const notification = fromBackendNotification(payload.notification);
	return {
		notification,
		subject: parseSubjectSummary(payload.notification.subjectRaw),
	};
}

export interface UpdateNotificationResponse {
	notification: BackendNotificationResponse;
}

export interface BulkUpdateNotificationResponse {
	count: number;
}

// Mark notification as read
export async function markNotificationRead(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/mark-read`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to mark notification as read (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Mark notification as unread
export async function markNotificationUnread(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/mark-unread`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to mark notification as unread (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Archive notification
export async function archiveNotification(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/archive`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to archive notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Unarchive notification
export async function unarchiveNotification(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/unarchive`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unarchive notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Mute notification
export async function muteNotification(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/mute`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to mute notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Unmute notification
export async function unmuteNotification(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/unmute`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unmute notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Snooze notification
export async function snoozeNotification(
	githubId: string,
	snoozedUntil: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/snooze`,
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ snoozedUntil }),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to snooze notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Unsnooze notification (clear snoozed_until)
export async function unsnoozeNotification(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/unsnooze`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unsnooze notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Star notification
export async function starNotification(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/star`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to star notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Unstar notification
export async function unstarNotification(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/unstar`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unstar notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Unfilter notification (move to inbox)
export async function unfilterNotification(
	githubId: string,
	fetchImpl?: typeof fetch
): Promise<Notification> {
	const response = await fetchWithAuth(
		`/api/notifications/${encodeURIComponent(githubId)}/unfilter`,
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unfilter notification (${response.status})`);
	}

	const payload: UpdateNotificationResponse = await response.json();
	return fromBackendNotification(payload.notification);
}

// Bulk snooze notifications
export async function bulkSnoozeNotifications(
	githubIds: string[],
	snoozedUntil: string,
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query, snoozedUntil } : { githubIds, snoozedUntil };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/snooze",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to bulk snooze notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk unsnooze notifications
export async function bulkUnsnoozeNotifications(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/unsnooze",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to bulk unsnooze notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk mark as unread
export async function bulkMarkNotificationsUnread(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/mark-unread",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to mark notifications as unread (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk mark as read
export async function bulkMarkNotificationsRead(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/mark-read",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to mark notifications as read (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk archive
export async function bulkArchiveNotifications(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/archive",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to archive notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk unarchive
export async function bulkUnarchiveNotifications(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/unarchive",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unarchive notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk mute
export async function bulkMuteNotifications(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/mute",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to mute notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk unmute
export async function bulkUnmuteNotifications(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/unmute",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unmute notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk star
export async function bulkStarNotifications(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/star",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to star notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk unstar
export async function bulkUnstarNotifications(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/unstar",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unstar notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk unfilter notifications (move to inbox)
export async function bulkUnfilterNotifications(
	githubIds: string[],
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query } : { githubIds };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/unfilter",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to unfilter notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk assign tag
export async function bulkAssignTag(
	githubIds: string[],
	tagId: string,
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query, tagId: tagId } : { githubIds, tagId: tagId };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/assign-tag",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to assign tag to notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

// Bulk remove tag
export async function bulkRemoveTag(
	githubIds: string[],
	tagId: string,
	query?: string,
	fetchImpl?: typeof fetch
): Promise<number> {
	// Send either githubIds or query, but not both
	// Note: query !== undefined includes empty string, which is valid for inbox semantics
	const body = query !== undefined ? { query, tagId: tagId } : { githubIds, tagId: tagId };

	const response = await fetchWithAuth(
		"/api/notifications/bulk/remove-tag",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(body),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to remove tag from notifications (${response.status})`);
	}

	const payload: BulkUpdateNotificationResponse = await response.json();
	return payload.count;
}

export async function refreshNotificationSubject(
	githubId: string,
	options: { fetch?: typeof fetch; query?: string } = {}
): Promise<Notification> {
	let url = `/api/notifications/${encodeURIComponent(githubId)}/refresh-subject`;
	if (options.query) {
		const separator = url.includes("?") ? "&" : "?";
		url = `${url}${separator}query=${encodeURIComponent(options.query)}`;
	}
	const response = await fetchWithAuth(
		url,
		{
			method: "POST",
		},
		options.fetch
	);

	if (!response.ok) {
		throw new Error(`Failed to refresh subject (${response.status})`);
	}

	const payload: { notification: BackendNotificationResponse } = await response.json();
	return fromBackendNotification(payload.notification);
}

export async function fetchNotificationTimeline(
	githubId: string,
	perPage: number,
	page: number,
	fetchImpl?: typeof fetch
): Promise<NotificationTimelineResponse> {
	const url = `/api/notifications/${encodeURIComponent(githubId)}/timeline?per_page=${perPage}&page=${page}`;
	const response = await fetchWithAuth(url, {}, fetchImpl);

	if (!response.ok) {
		throw new Error(`Failed to fetch timeline (${response.status})`);
	}

	const payload: NotificationTimelineResponse = await response.json();
	return payload;
}

export { fromBackendNotification };
