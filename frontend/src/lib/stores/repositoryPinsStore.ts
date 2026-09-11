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

import { writable, get, type Readable } from "svelte/store";
import { browser } from "$app/environment";

const PINNED_REPOSITORIES_KEY = "octobud:repositories:pinned";

/**
 * Read pinned repository IDs from storage. Accepts the current `number[]` shape and the
 * earlier `{id, ...}[]` shape; anything else yields no pins.
 */
export function readPinnedRepositoryIds(storage: Pick<Storage, "getItem">): number[] {
	try {
		const raw = storage.getItem(PINNED_REPOSITORIES_KEY);
		if (!raw) return [];
		const parsed: unknown = JSON.parse(raw);
		if (!Array.isArray(parsed)) return [];
		const ids: number[] = [];
		for (const entry of parsed) {
			const id =
				typeof entry === "number"
					? entry
					: entry && typeof entry === "object"
						? (entry as { id?: unknown }).id
						: undefined;
			if (typeof id === "number" && Number.isInteger(id) && id > 0 && !ids.includes(id)) {
				ids.push(id);
			}
		}
		return ids;
	} catch {
		return [];
	}
}

/**
 * Pinned repositories are a global, per-browser preference: they surface at the top of
 * every repository selector regardless of the current view. Only IDs are stored; the
 * server supplies names and avatars (the counts endpoint always returns included IDs).
 */
export function createRepositoryPinsStore(storage: Storage | null = browser ? localStorage : null) {
	const pinnedRepositoryIds = writable<number[]>(storage ? readPinnedRepositoryIds(storage) : []);

	if (storage) {
		pinnedRepositoryIds.subscribe((ids) => {
			try {
				storage.setItem(PINNED_REPOSITORIES_KEY, JSON.stringify(ids));
			} catch {
				// Storage may be unavailable (private mode, quota); pins then live for the session only
			}
		});
	}

	function isPinned(id: number): boolean {
		return get(pinnedRepositoryIds).includes(id);
	}

	/** Pin or unpin; returns the new pinned state. */
	function togglePin(id: number): boolean {
		const pinned = !isPinned(id);
		pinnedRepositoryIds.update((ids) => (pinned ? [...ids, id] : ids.filter((x) => x !== id)));
		return pinned;
	}

	return {
		pinnedRepositoryIds: pinnedRepositoryIds as Readable<number[]>,
		isPinned,
		togglePin,
	};
}

export type RepositoryPinsStore = ReturnType<typeof createRepositoryPinsStore>;

let instance: RepositoryPinsStore | null = null;

/** Shared singleton so every repository selector sees the same pins. */
export function getRepositoryPinsStore(): RepositoryPinsStore {
	if (!instance) {
		instance = createRepositoryPinsStore();
	}
	return instance;
}
