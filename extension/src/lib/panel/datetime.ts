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
 * Date helpers for the custom snooze dialog.
 *
 * They live outside the component because `svelte/prefer-svelte-reactivity`
 * flags plain `Date` instances inside `.svelte` files — a fair rule for reactive
 * state, but these are pure conversions with no state to track.
 */

/**
 * `<input type="datetime-local">` reads and writes local wall-clock time in
 * `YYYY-MM-DDTHH:mm`, not an ISO instant, so the offset has to be folded in
 * before slicing.
 */
export function toLocalInputValue(date: Date): string {
	const offsetMs = date.getTimezoneOffset() * 60_000;
	return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
}

/** Matches the dropdown's "Tomorrow (8am)" preset. */
export function defaultSnoozeTarget(now: Date = new Date()): string {
	const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1, 8, 0, 0, 0);
	return toLocalInputValue(tomorrow);
}

export function nowAsInputValue(now: Date = new Date()): string {
	return toLocalInputValue(now);
}

/** Local-time input strings are only valid if they land in the future. */
export function isFutureInputValue(value: string, now: number = Date.now()): boolean {
	if (value === "") return false;
	const parsed = Date.parse(value);
	return Number.isFinite(parsed) && parsed > now;
}

export function toIsoInstant(value: string): string {
	return new Date(value).toISOString();
}
