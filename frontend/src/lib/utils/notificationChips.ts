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

import octicons from "@primer/octicons";

export interface NotificationChipConfig {
	path: string;
	stateLabel: string;
	bgColorClass: string;
	textColorClass: string;
	iconColorClass: string;
}

// Helper to get the SVG paths from an octicon (returns the full path HTML)
function getIconPath(iconName: string): string {
	const icon = octicons[iconName];
	if (!icon || !icon.heights || !icon.heights["16"]) {
		return ""; // fallback to empty path
	}
	// Return the full path element(s) as they may have multiple paths
	return icon.heights["16"].path;
}

/**
 * Parse subject metadata from raw data
 */
function parseSubjectMetadata(
	raw: unknown
): { state?: string; merged?: boolean; draft?: boolean } | null {
	if (!raw || typeof raw !== "object") return null;
	const data = raw as Record<string, unknown>;

	const state = typeof data.state === "string" ? data.state : undefined;
	const merged = typeof data.merged === "boolean" ? data.merged : undefined;
	const draft = typeof data.draft === "boolean" ? data.draft : undefined;

	return { state, merged, draft };
}

/**
 * Determines the appropriate GitHub-style chip configuration for a notification
 * based on its type and state (for PRs and Issues)
 * Returns null for non-PR/Issue notifications
 */
export function getNotificationChip(
	subjectType: string,
	subjectRaw?: unknown,
	subjectState?: string,
	subjectMerged?: boolean
): NotificationChipConfig | null {
	const normalizedType = subjectType.toLowerCase().replace(/[-_\s]/g, "");

	// Parse subject metadata to check state and merged/draft status
	// Prefer extracted fields (subjectState, subjectMerged) over parsing from subjectRaw
	const subject = parseSubjectMetadata(subjectRaw);
	const state = subjectState ?? subject?.state;
	const merged = subjectMerged ?? subject?.merged;

	// Pull Request
	// Logic: Open -> green PR chip, Closed and not merged -> red closed PR chip, Merged -> purple merge chip
	if (normalizedType === "pullrequest" || normalizedType === "pr") {
		// Merged PR (regardless of state)
		if (merged === true) {
			return {
				path: getIconPath("git-merge"),
				stateLabel: "Merged",
				bgColorClass: "bg-purple-600 dark:bg-purple-600/90",
				textColorClass: "text-purple-50 dark:text-purple-100",
				iconColorClass: "text-purple-50 dark:text-purple-100",
			};
		}

		// Closed and not merged -> red closed PR chip
		if (state === "closed" && merged === false) {
			return {
				path: getIconPath("git-pull-request-closed"),
				stateLabel: "Closed",
				bgColorClass: "bg-red-600 dark:bg-red-600/90",
				textColorClass: "text-red-50 dark:text-red-100",
				iconColorClass: "text-red-50 dark:text-red-100",
			};
		}

		// Draft PR
		if (subject?.draft) {
			return {
				path: getIconPath("git-pull-request-draft"),
				stateLabel: "Draft",
				bgColorClass: "bg-gray-600 dark:bg-gray-600/90",
				textColorClass: "text-gray-50 dark:text-gray-100",
				iconColorClass: "text-gray-50 dark:text-gray-100",
			};
		}

		// Default: Open PR
		return {
			path: getIconPath("git-pull-request"),
			stateLabel: "Open",
			bgColorClass: "bg-green-600 dark:bg-green-600/90",
			textColorClass: "text-green-50 dark:text-green-100",
			iconColorClass: "text-green-50 dark:text-green-100",
		};
	}

	// Issue
	if (normalizedType === "issue") {
		// Closed Issue
		if (state === "closed") {
			return {
				path: getIconPath("issue-closed"),
				stateLabel: "Closed",
				bgColorClass: "bg-purple-600 dark:bg-purple-600/90",
				textColorClass: "text-purple-50 dark:text-purple-100",
				iconColorClass: "text-purple-50 dark:text-purple-100",
			};
		}

		// Default: Open Issue
		return {
			path: getIconPath("issue-opened"),
			stateLabel: "Open",
			bgColorClass: "bg-green-600 dark:bg-green-600/90",
			textColorClass: "text-green-50 dark:text-green-100",
			iconColorClass: "text-green-50 dark:text-green-100",
		};
	}

	// Return null for other notification types (they'll use the regular icon display)
	return null;
}
