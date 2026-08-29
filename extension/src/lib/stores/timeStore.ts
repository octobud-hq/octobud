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

import { readable } from "svelte/store";

/**
 * Ticks so relative timestamps ("2h ago") stay fresh without a reload.
 *
 * frontend/src/lib/stores/timeStore.ts cannot be vendored because it gates the
 * interval on SvelteKit's `$app/environment` `browser` flag. A `readable` store
 * gives the panel the same behaviour and, unlike the desktop version, stops the
 * interval automatically when the last subscriber goes away — which matters here
 * because the side panel is torn down and recreated as the user opens and closes
 * it.
 */
const TICK_INTERVAL_MS = 30_000;

export const currentTime = readable(new Date(), (set) => {
	const interval = setInterval(() => set(new Date()), TICK_INTERVAL_MS);
	return () => clearInterval(interval);
});
