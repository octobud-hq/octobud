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
 * Format a snooze date into a human-readable message
 */
export function formatSnoozeMessage(snoozedUntil: string, snoozedAt?: string): string {
	const now = new Date();
	const snoozeDate = new Date(snoozedUntil);

	// If snooze date is in the past, show "Snoozed X ago" from the snoozedAt time
	if (snoozeDate <= now) {
		// Use snoozedAt if available, otherwise fall back to snoozedUntil
		const referenceDate = snoozedAt ? new Date(snoozedAt) : snoozeDate;
		const diffMs = now.getTime() - referenceDate.getTime();
		const diffMinutes = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMinutes / 60);
		const diffDays = Math.floor(diffHours / 24);

		if (diffDays > 0) {
			return `Snoozed ${diffDays}d ago`;
		} else if (diffHours > 0) {
			return `Snoozed ${diffHours}h ago`;
		} else if (diffMinutes > 0) {
			return `Snoozed ${diffMinutes}m ago`;
		} else {
			return `Snoozed just now`;
		}
	}

	// If snooze date is in the future, show "Snoozed until X"
	const isToday = snoozeDate.toDateString() === now.toDateString();
	const tomorrow = new Date(now);
	tomorrow.setDate(tomorrow.getDate() + 1);
	const isTomorrow = snoozeDate.toDateString() === tomorrow.toDateString();

	if (isToday) {
		// Format as "Snoozed until 6pm"
		const hours = snoozeDate.getHours();
		const minutes = snoozeDate.getMinutes();
		const period = hours >= 12 ? "pm" : "am";
		const displayHours = hours % 12 || 12;
		const timeStr =
			minutes > 0
				? `${displayHours}:${minutes.toString().padStart(2, "0")}${period}`
				: `${displayHours}${period}`;
		return `Snoozed until ${timeStr}`;
	} else if (isTomorrow) {
		return "Snoozed until tomorrow";
	} else {
		// Format as "Snoozed until Wednesday" or with date if far in future
		const diffDays = Math.floor((snoozeDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

		if (diffDays < 7) {
			// Show day of week
			const dayNames = [
				"Sunday",
				"Monday",
				"Tuesday",
				"Wednesday",
				"Thursday",
				"Friday",
				"Saturday",
			];
			return `Snoozed until ${dayNames[snoozeDate.getDay()]}`;
		} else {
			// Show date
			const month = snoozeDate.toLocaleDateString("en-US", { month: "short" });
			const day = snoozeDate.getDate();
			return `Snoozed until ${month} ${day}`;
		}
	}
}
