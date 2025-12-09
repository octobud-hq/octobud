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
 * Shared utilities for notification display and formatting
 */

import { formatRelativeShort } from "./time";

/**
 * Format a timestamp using relative time for current day, otherwise Gmail-style:
 * - Current day -> "2h ago", "5m ago", etc. (relative time)
 * - Current year -> "Jan 18" (month and day)
 * - Past year -> "Jan 18, 2024" (month, day, and year)
 */
export function formatRelative(isoDate: string): string {
	const updated = new Date(isoDate);
	const now = new Date();

	// Check if same day
	const isSameDay =
		updated.getFullYear() === now.getFullYear() &&
		updated.getMonth() === now.getMonth() &&
		updated.getDate() === now.getDate();

	if (isSameDay) {
		// Current day -> show relative time (e.g., "2h ago", "5m ago")
		return formatRelativeShort(updated);
	}

	// Check if same year
	const isSameYear = updated.getFullYear() === now.getFullYear();

	if (isSameYear) {
		// Current year -> show month and day (e.g., "Jan 18")
		return updated.toLocaleDateString(undefined, {
			month: "short",
			day: "numeric",
		});
	}

	// Past year -> show month, day, and year (e.g., "Jan 18, 2024")
	return updated.toLocaleDateString(undefined, {
		month: "short",
		day: "numeric",
		year: "numeric",
	});
}

/**
 * Format a timestamp as absolute time using locale string
 */
export function formatAbsolute(isoDate: string): string {
	return new Date(isoDate).toLocaleString();
}

/**
 * Normalize reason labels to human-friendly, title case versions for pills
 * @param reason - The raw reason string from the API
 * @param options - Configuration options
 * @param options.short - If true, returns abbreviated versions suitable for width-constrained contexts
 */
export function normalizeReason(
	reason: string | null | undefined,
	options: { short?: boolean } = {}
): string {
	if (!reason) return "Other";

	const { short = false } = options;

	// Convert snake_case to spaces and normalize
	const normalized = reason.toLowerCase().replace(/_/g, " ");

	// Return short versions for width-constrained contexts
	if (short) {
		if (normalized.includes("review")) return "Review";
		if (normalized === "mention") return "Mention";
		if (normalized === "assign" || normalized === "assigned") return "Assigned";
		if (normalized === "author") return "Author";
		if (normalized === "comment") return "Comment";
		if (normalized === "manual") return "Manual";
		if (normalized === "team mention" || normalized.includes("team")) return "Team";
		if (normalized === "subscribed" || normalized === "subscription") return "Subscribed";
		if (normalized === "ci activity" || normalized.includes("ci")) return "CI";
		if (normalized === "security alert" || normalized.includes("security")) return "Security";

		// Default: take first word and capitalize it
		const firstWord = normalized.split(" ")[0];
		return firstWord.charAt(0).toUpperCase() + firstWord.slice(1);
	}

	// Return full, human-friendly version with title case
	// Handle special cases where words should be uppercase (like "CI")
	return normalized
		.split(" ")
		.map((word) => {
			// Special case: "ci" should be uppercase
			if (word === "ci") return "CI";
			return word.charAt(0).toUpperCase() + word.slice(1);
		})
		.join(" ");
}

/**
 * Parse subject metadata from raw data
 * Note: Prefer using extracted fields (subjectMerged, subjectState, etc.) from the notification
 * This is kept as a fallback for cases where extracted fields are not available
 */
export function parseSubjectMetadata(raw: unknown): {
	number?: number;
	author?: string;
	merged?: boolean;
	state?: string;
	stateReason?: string;
} | null {
	if (!raw || typeof raw !== "object") return null;
	const data = raw as Record<string, unknown>;

	const number = typeof data.number === "number" ? data.number : undefined;
	const merged = typeof data.merged === "boolean" ? data.merged : undefined;
	const state = typeof data.state === "string" ? data.state : undefined;
	const stateReason = typeof data.state_reason === "string" ? data.state_reason : undefined;

	let author: string | undefined;
	const user = data.user;
	if (user && typeof user === "object") {
		const login = (user as Record<string, unknown>).login;
		if (typeof login === "string") author = login;
	}

	return { number, author, merged, state, stateReason };
}

/**
 * Mail icon definitions for read/unread states
 */
export const mailOpenIcon = {
	path: "M21.8178 7.68475C22.5632 8.25194 23 9.13431 23 10.071V20C23 21.6569 21.6569 23 20 23H4C2.34315 23 1 21.6569 1 20V10.071C1 9.13431 1.43675 8.25194 2.18224 7.68475C4.36739 6.02221 8.93135 2.55149 10.2 1.6C11.2667 0.8 12.7333 0.8 13.8 1.6C15.0686 2.55148 19.6326 6.02221 21.8178 7.68475ZM3 10.1688L10.1411 15.8065C11.231 16.667 12.769 16.667 13.8589 15.8065L21 10.1688V20C21 20.5523 20.5523 21 20 21H4C3.44772 21 3 20.5523 3 20V10.1688ZM12.6 3.2L19.7792 8.58443L12.6196 14.2367C12.2563 14.5236 11.7437 14.5236 11.3804 14.2367L4.22077 8.58443L11.4 3.2C11.7556 2.93333 12.2444 2.93333 12.6 3.2Z",
	fillRule: "evenodd" as const,
	clipRule: "evenodd" as const,
} as const;

export const mailClosedIcon = {
	path: "M20 4C21.6569 4 23 5.34315 23 7V17C23 18.6569 21.6569 20 20 20H4C2.34315 20 1 18.6569 1 17V7C1 5.34315 2.34315 4 4 4H20ZM19.2529 6H4.74718L11.3804 11.2367C11.7437 11.5236 12.2563 11.5236 12.6197 11.2367L19.2529 6ZM3 7.1688V17C3 17.5523 3.44772 18 4 18H20C20.5523 18 21 17.5523 21 17V7.16882L13.8589 12.8065C12.769 13.667 11.231 13.667 10.1411 12.8065L3 7.1688Z",
	fillRule: "evenodd" as const,
	clipRule: "evenodd" as const,
} as const;

/**
 * Check icon path for done state
 */
export const checkIconPath = "m4.5 12.75 4.5 4.5 10.5-10.5";
