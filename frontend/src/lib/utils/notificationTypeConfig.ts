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

/**
 * Configuration for different GitHub notification types
 * Defines UI behavior and content labels for each notification type
 */

export interface NotificationTypeConfig {
	showCommentThread: boolean;
	contentLabel: string;
	emptyContentMessage: string;
	timestampVerb: string; // e.g., "commented", "committed", "released"
}

/**
 * Get the configuration for a specific notification type
 */
export function getNotificationTypeConfig(subjectType: string): NotificationTypeConfig {
	const normalizedType = subjectType.toLowerCase().replace(/[-_\s]/g, "");

	switch (normalizedType) {
		case "pullrequest":
		case "pr":
			return {
				showCommentThread: true,
				contentLabel: "Description",
				emptyContentMessage: "No description provided.",
				timestampVerb: "opened",
			};

		case "issue":
			return {
				showCommentThread: true,
				contentLabel: "Description",
				emptyContentMessage: "No description provided.",
				timestampVerb: "opened",
			};

		case "discussion":
			return {
				showCommentThread: true,
				contentLabel: "Discussion",
				emptyContentMessage: "No description provided.",
				timestampVerb: "started",
			};

		case "commit":
			return {
				showCommentThread: false,
				contentLabel: "Commit Message",
				emptyContentMessage: "No commit message provided.",
				timestampVerb: "committed",
			};

		case "release":
			return {
				showCommentThread: false,
				contentLabel: "Release Notes",
				emptyContentMessage: "No release notes provided.",
				timestampVerb: "released",
			};

		case "checkrun":
		case "checksuite":
			return {
				showCommentThread: false,
				contentLabel: "Check Details",
				emptyContentMessage: "No details available.",
				timestampVerb: "triggered",
			};

		case "repositoryvulnerabilityalert":
		case "securityalert":
			return {
				showCommentThread: false,
				contentLabel: "Security Alert",
				emptyContentMessage: "No details available.",
				timestampVerb: "reported",
			};

		default:
			// Fallback for unknown types
			return {
				showCommentThread: false,
				contentLabel: "Details",
				emptyContentMessage: "No details available.",
				timestampVerb: "updated",
			};
	}
}

/**
 * Check if a notification type supports comments
 */
export function supportsComments(subjectType: string): boolean {
	const config = getNotificationTypeConfig(subjectType);
	return config.showCommentThread;
}
