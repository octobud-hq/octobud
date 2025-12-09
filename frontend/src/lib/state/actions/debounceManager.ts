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
 * Debounce Manager
 * Centralized debounce state management to prevent bugs from split timeouts
 */

export const SEARCH_DEBOUNCE_MS = 250;

export interface DebounceManager {
	setSearchDebounce: (fn: () => void) => void;
	setQueryDebounce: (fn: () => void) => void;
	clearAll: () => void;
}

/**
 * Create a debounce manager with shared timeout state
 */
export function createDebounceManager(): DebounceManager {
	let searchDebounceTimeout: ReturnType<typeof setTimeout> | null = null;
	let queryDebounceTimeout: ReturnType<typeof setTimeout> | null = null;

	return {
		setSearchDebounce: (fn: () => void) => {
			if (searchDebounceTimeout) {
				clearTimeout(searchDebounceTimeout);
			}
			searchDebounceTimeout = setTimeout(() => {
				searchDebounceTimeout = null;
				fn();
			}, SEARCH_DEBOUNCE_MS);
		},

		setQueryDebounce: (fn: () => void) => {
			if (queryDebounceTimeout !== null) {
				clearTimeout(queryDebounceTimeout);
			}
			queryDebounceTimeout = setTimeout(() => {
				queryDebounceTimeout = null;
				fn();
			}, SEARCH_DEBOUNCE_MS);
		},

		clearAll: () => {
			if (searchDebounceTimeout) {
				clearTimeout(searchDebounceTimeout);
				searchDebounceTimeout = null;
			}
			if (queryDebounceTimeout) {
				clearTimeout(queryDebounceTimeout);
				queryDebounceTimeout = null;
			}
		},
	};
}
