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

export function formatRelativeTime(date: Date): string {
	const diffMs = date.getTime() - Date.now();
	const diffInSeconds = Math.round(diffMs / 1000);
	const absSeconds = Math.abs(diffInSeconds);
	const formatter = new Intl.RelativeTimeFormat(undefined, {
		numeric: "auto",
	});

	if (absSeconds < 60) {
		return formatter.format(diffInSeconds, "second");
	}

	const diffInMinutes = Math.round(diffInSeconds / 60);
	if (Math.abs(diffInMinutes) < 60) {
		return formatter.format(diffInMinutes, "minute");
	}

	const diffInHours = Math.round(diffInMinutes / 60);
	if (Math.abs(diffInHours) < 24) {
		return formatter.format(diffInHours, "hour");
	}

	const diffInDays = Math.round(diffInHours / 24);
	if (Math.abs(diffInDays) < 7) {
		return formatter.format(diffInDays, "day");
	}

	const diffInWeeks = Math.round(diffInDays / 7);
	if (Math.abs(diffInWeeks) < 5) {
		return formatter.format(diffInWeeks, "week");
	}

	const diffInMonths = Math.round(diffInDays / 30);
	if (Math.abs(diffInMonths) < 12) {
		return formatter.format(diffInMonths, "month");
	}

	const diffInYears = Math.round(diffInDays / 365);
	return formatter.format(diffInYears, "year");
}

export function parseTimestamp(value: string | null | undefined): Date | null {
	if (!value) {
		return null;
	}
	const parsed = new Date(value);
	if (Number.isNaN(parsed.getTime())) {
		return null;
	}
	return parsed;
}

export function formatRelativeShort(date: Date): string {
	const now = Date.now();
	const diffMs = date.getTime() - now;
	const isFuture = diffMs > 0;
	const absMs = Math.abs(diffMs);
	const absSeconds = Math.round(absMs / 1000);
	const suffix = isFuture ? "from now" : "ago";

	if (absSeconds < 60) {
		const seconds = Math.max(0, absSeconds);
		return `${seconds}s ${suffix}`;
	}

	const absMinutes = Math.floor(absSeconds / 60);
	if (absMinutes < 60) {
		return `${absMinutes}m ${suffix}`;
	}

	const absHours = Math.floor(absMinutes / 60);
	if (absHours < 24) {
		return `${absHours}h ${suffix}`;
	}

	const absDays = Math.floor(absHours / 24);
	if (absDays < 7) {
		return `${absDays}d ${suffix}`;
	}

	const absWeeks = Math.floor(absDays / 7);
	if (absWeeks < 5) {
		return `${absWeeks}w ${suffix}`;
	}

	const absMonths = Math.floor(absDays / 30);
	if (absMonths < 12) {
		return `${absMonths}mo ${suffix}`;
	}

	const absYears = Math.floor(absDays / 365);
	return `${absYears}y ${suffix}`;
}

/**
 * Convert an RFC3339/ISO date string to the format expected by datetime-local inputs.
 * Returns the date in local timezone as YYYY-MM-DDTHH:mm format.
 */
export function toLocalDatetime(rfc3339: string): string {
	if (!rfc3339) return "";
	const timestamp = Date.parse(rfc3339);
	const date = new Date(timestamp);
	// Format as YYYY-MM-DDTHH:mm for datetime-local input (in local timezone)
	const year = date.getFullYear();
	const month = String(date.getMonth() + 1).padStart(2, "0");
	const day = String(date.getDate()).padStart(2, "0");
	const hours = String(date.getHours()).padStart(2, "0");
	const minutes = String(date.getMinutes()).padStart(2, "0");
	return `${year}-${month}-${day}T${hours}:${minutes}`;
}
