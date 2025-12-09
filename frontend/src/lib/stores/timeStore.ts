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

import { browser } from "$app/environment";
import { writable } from "svelte/store";

/**
 * A reactive store that updates every minute to trigger timestamp refreshes
 * Components can subscribe to this to automatically update relative timestamps
 */
function createTimeStore() {
	const { subscribe, set } = writable(new Date());
	let intervalId: ReturnType<typeof setInterval> | null = null;

	// Start the interval only on the client side
	// This ensures the store updates every minute, triggering reactivity in components
	if (browser) {
		// Update every minute to keep relative timestamps fresh
		// For better UX, we could update more frequently (every 30s) for very recent notifications
		intervalId = setInterval(() => {
			set(new Date());
		}, 30000); // Update every 60 seconds
	}

	function destroy() {
		if (intervalId) {
			clearInterval(intervalId);
			intervalId = null;
		}
	}

	return {
		subscribe,
		destroy,
	};
}

export const currentTime = createTimeStore();
