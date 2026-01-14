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

import type { UndoableActionType } from "./types";
import { archiveIconPaths, unarchiveIconPath } from "$lib/utils/archiveIcons";
import { bellOffIconPath, bellOnIconPath } from "$lib/utils/muteIcons";
import { mailOpenIcon, mailClosedIcon } from "$lib/utils/notificationHelpers";

export interface ActionIconConfig {
	paths: string[];
	viewBox: string;
	useStroke: boolean;
	strokeWidth?: string;
	fillRule?: "evenodd" | "nonzero";
	clipRule?: "evenodd" | "nonzero";
}

// Star icon path
const starIconPath =
	"M11.245 4.174C11.4765 3.50808 11.5922 3.17513 11.7634 3.08285C11.9115 3.00298 12.0898 3.00298 12.238 3.08285C12.4091 3.17513 12.5248 3.50808 12.7563 4.174L14.2866 8.57639C14.3525 8.76592 14.3854 8.86068 14.4448 8.93125C14.4972 8.99359 14.5641 9.04218 14.6396 9.07278C14.725 9.10743 14.8253 9.10947 15.0259 9.11356L19.6857 9.20852C20.3906 9.22288 20.743 9.23007 20.8837 9.36432C21.0054 9.48051 21.0605 9.65014 21.0303 9.81569C20.9955 10.007 20.7146 10.2199 20.1528 10.6459L16.4387 13.4616C16.2788 13.5829 16.1989 13.6435 16.1501 13.7217C16.107 13.7909 16.0815 13.8695 16.0757 13.9507C16.0692 14.0427 16.0982 14.1387 16.1563 14.3308L17.506 18.7919C17.7101 19.4667 17.8122 19.8041 17.728 19.9793C17.6551 20.131 17.5108 20.2358 17.344 20.2583C17.1513 20.2842 16.862 20.0829 16.2833 19.6802L12.4576 17.0181C12.2929 16.9035 12.2106 16.8462 12.1211 16.8239C12.042 16.8043 11.9593 16.8043 11.8803 16.8239C11.7908 16.8462 11.7084 16.9035 11.5437 17.0181L7.71805 19.6802C7.13937 20.0829 6.85003 20.2842 6.65733 20.2583C6.49056 20.2358 6.34626 20.131 6.27337 19.9793C6.18915 19.8041 6.29123 19.4667 6.49538 18.7919L7.84503 14.3308C7.90313 14.1387 7.93218 14.0427 7.92564 13.9507C7.91986 13.8695 7.89432 13.7909 7.85123 13.7217C7.80246 13.6435 7.72251 13.5829 7.56262 13.4616L3.84858 10.6459C3.28678 10.2199 3.00588 10.007 2.97101 9.81569C2.94082 9.65014 2.99594 9.48051 3.11767 9.36432C3.25831 9.23007 3.61074 9.22289 4.31559 9.20852L8.9754 9.11356C9.176 9.10947 9.27631 9.10743 9.36177 9.07278C9.43726 9.04218 9.50414 8.99359 9.55657 8.93125C9.61593 8.86068 9.64887 8.76592 9.71475 8.57639L11.245 4.174Z";

// Snooze/clock icon path
const snoozeIconPath =
	"M12 6V12L16 14M22 12C22 17.5228 17.5228 22 12 22C6.47715 22 2 17.5228 2 12C2 6.47715 6.47715 2 12 2C17.5228 2 22 6.47715 22 12Z";

// Tag icon paths
const tagFilledIconPath =
	"M17.63 5.84C17.27 5.33 16.67 5 16 5L5 5.01C3.9 5.01 3 5.9 3 7v10c0 1.1.9 1.99 2 1.99L16 19c.67 0 1.27-.33 1.63-.84L22 12l-4.37-6.16z";

/**
 * Get the icon configuration for an action type
 */
export function getActionIconConfig(actionType: UndoableActionType): ActionIconConfig {
	switch (actionType) {
		case "archive":
			return {
				paths: archiveIconPaths,
				viewBox: "0 0 24 24",
				useStroke: true,
				strokeWidth: "1.5",
			};
		case "unarchive":
			return {
				paths: [unarchiveIconPath],
				viewBox: "0 0 24 24",
				useStroke: false,
			};
		case "mute":
			return {
				paths: [bellOffIconPath],
				viewBox: "0 0 24 24",
				useStroke: true,
				strokeWidth: "1.5",
			};
		case "unmute":
			return {
				paths: [bellOnIconPath],
				viewBox: "0 0 24 24",
				useStroke: true,
				strokeWidth: "1.5",
			};
		case "star":
		case "unstar":
			return {
				paths: [starIconPath],
				viewBox: "0 0 24 24",
				useStroke: true,
				strokeWidth: "2",
			};
		case "markRead":
			return {
				paths: [mailOpenIcon.path],
				viewBox: "0 0 24 24",
				useStroke: false,
				fillRule: mailOpenIcon.fillRule,
				clipRule: mailOpenIcon.clipRule,
			};
		case "markUnread":
			return {
				paths: [mailClosedIcon.path],
				viewBox: "0 0 24 24",
				useStroke: false,
				fillRule: mailClosedIcon.fillRule,
				clipRule: mailClosedIcon.clipRule,
			};
		case "snooze":
		case "unsnooze":
			return {
				paths: [snoozeIconPath],
				viewBox: "0 0 24 24",
				useStroke: true,
				strokeWidth: "1.8",
			};
		case "assignTag":
		case "removeTag":
			return {
				paths: [tagFilledIconPath],
				viewBox: "0 0 24 24",
				useStroke: false,
			};
		default:
			return {
				paths: [snoozeIconPath],
				viewBox: "0 0 24 24",
				useStroke: true,
				strokeWidth: "1.8",
			};
	}
}

/**
 * Get the display label for an action type
 */
export function getActionLabel(actionType: UndoableActionType): string {
	switch (actionType) {
		case "archive":
			return "Archived";
		case "unarchive":
			return "Unarchived";
		case "mute":
			return "Muted";
		case "unmute":
			return "Unmuted";
		case "star":
			return "Starred";
		case "unstar":
			return "Unstarred";
		case "markRead":
			return "Marked read";
		case "markUnread":
			return "Marked unread";
		case "snooze":
			return "Snoozed";
		case "unsnooze":
			return "Unsnoozed";
		case "assignTag":
			return "Added tag";
		case "removeTag":
			return "Removed tag";
		default:
			return "Action";
	}
}
