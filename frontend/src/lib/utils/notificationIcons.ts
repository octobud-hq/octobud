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

export interface NotificationIconConfig {
	path: string;
	colorClass: string;
	label: string;
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
 * Determines the appropriate icon, color, and label for a notification
 * based on its type and state.
 */
export function getNotificationIcon(
	subjectType: string,
	subjectRaw?: unknown,
	isRead: boolean = false,
	subjectState?: string,
	subjectMerged?: boolean
): NotificationIconConfig {
	const normalizedType = subjectType.toLowerCase().replace(/[-_\s]/g, "");

	// Parse subject metadata to check state and merged status
	// Prefer extracted fields (subjectState, subjectMerged) over parsing from subjectRaw
	const subject = parseSubjectMetadata(subjectRaw);
	const state = subjectState ?? subject?.state;
	const merged = subjectMerged ?? subject?.merged;

	// Helper to get color class based on read status
	// In light mode, keep same color for read/unread (no opacity changes)
	// In dark mode, apply opacity to read items
	const getColorClass = (fullColor: string, dimmedColor: string): string => {
		if (isRead) {
			// Extract light mode color from fullColor (e.g., "text-green-500")
			// and dark mode dimmed color from dimmedColor (e.g., "dark:text-green-400/50")
			const lightColorParts = fullColor.split(" ").filter((c) => !c.startsWith("dark:"));
			const darkDimmedParts = dimmedColor.split(" ").filter((c) => c.startsWith("dark:"));
			return [...lightColorParts, ...darkDimmedParts].join(" ");
		}
		return fullColor;
	};

	// Pull Request
	// Logic: Open -> green PR icon, Closed and not merged -> red closed PR icon, Merged -> purple merge icon
	if (normalizedType === "pullrequest" || normalizedType === "pr") {
		// Merged PR (regardless of state)
		if (merged === true) {
			return {
				path: getIconPath("git-merge"),
				colorClass: getColorClass(
					"text-purple-500 dark:text-purple-400",
					"text-purple-400/60 dark:text-purple-400/50"
				),
				label: "Merged PR",
			};
		}

		// Closed and not merged -> red closed PR icon
		if (state === "closed" && merged === false) {
			return {
				path: getIconPath("git-pull-request-closed"),
				colorClass: getColorClass(
					"text-red-500 dark:text-red-400",
					"text-red-400/60 dark:text-red-400/50"
				),
				label: "Closed PR",
			};
		}

		// Default: Open PR (green)
		return {
			path: getIconPath("git-pull-request"),
			colorClass: getColorClass(
				"text-green-500 dark:text-green-400",
				"text-green-400/60 dark:text-green-400/50"
			),
			label: "Open PR",
		};
	}

	// Issue
	if (normalizedType === "issue") {
		if (state === "closed") {
			return {
				path: getIconPath("issue-closed"),
				colorClass: getColorClass(
					"text-purple-500 dark:text-purple-400",
					"text-purple-400/60 dark:text-purple-400/50"
				),
				label: "Closed Issue",
			};
		}

		// Default: Open Issue (green)
		return {
			path: getIconPath("issue-opened"),
			colorClass: getColorClass(
				"text-green-500 dark:text-green-400",
				"text-green-400/60 dark:text-green-400/50"
			),
			label: "Open Issue",
		};
	}

	// Discussion
	if (normalizedType === "discussion") {
		return {
			path: getIconPath("comment-discussion"),
			colorClass: "text-gray-500 dark:text-gray-400",
			label: "Discussion",
		};
	}

	// Commit
	if (normalizedType === "commit") {
		return {
			path: getIconPath("git-commit"),
			colorClass: "text-gray-500 dark:text-gray-400",
			label: "Commit",
		};
	}

	// Release
	if (normalizedType === "release") {
		return {
			path: getIconPath("tag"),
			colorClass: "text-gray-500 dark:text-gray-400",
			label: "Release",
		};
	}

	// Check Run / Check Suite (CI/Actions)
	if (normalizedType === "checkrun" || normalizedType === "checksuite") {
		return {
			path: getIconPath("play"),
			colorClass: "text-gray-500 dark:text-gray-400",
			label: "CI Activity",
		};
	}

	// Security Alert
	if (normalizedType === "repositoryvulnerabilityalert" || normalizedType === "securityalert") {
		return {
			path: getIconPath("alert"),
			colorClass: "text-gray-500 dark:text-gray-400",
			label: "Security Alert",
		};
	}

	// Default fallback
	return {
		path: getIconPath("issue-opened"),
		colorClass: "text-gray-500 dark:text-gray-400",
		label: "Notification",
	};
}

/**
 * Parse subject metadata from raw data
 */
function parseSubjectMetadata(raw: unknown): { state?: string; merged?: boolean } | null {
	if (!raw || typeof raw !== "object") return null;
	const data = raw as Record<string, unknown>;

	const state = typeof data.state === "string" ? data.state : undefined;
	const merged = typeof data.merged === "boolean" ? data.merged : undefined;

	return { state, merged };
}

/**
 * Get the status bar color class based on read status
 */
export function getStatusBarColor(isRead: boolean): string {
	return isRead ? "bg-transparent" : "bg-blue-500";
}
