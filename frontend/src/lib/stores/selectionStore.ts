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

import { writable, derived, get } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import type { SelectAllMode } from "$lib/state/types";
import type { NotificationStore } from "./notificationStore";

/**
 * Selection Store
 * Manages selection and multiselect state
 */
export function createSelectionStore(notificationStore: NotificationStore) {
	const selectedIds = writable<Set<string>>(new Set());
	const multiselectMode = writable<boolean>(false);
	const selectAllMode = writable<SelectAllMode>("none");

	// Derived stores
	let selectionMapVersion = 0;
	const selectionMap = derived(
		[selectedIds, selectAllMode, notificationStore.pageData],
		([$selectedIds, $selectAllMode, $pageData]) => {
			const map = new Map<string, boolean>();

			if ($selectAllMode === "all") {
				for (const item of $pageData.items) {
					const key = item.githubId ?? item.id;
					if (key) {
						map.set(key, true);
					}
				}
			} else {
				for (const id of $selectedIds) {
					map.set(id, true);
				}
			}

			// Force new reference for reactivity
			(map as any)._version = ++selectionMapVersion;
			return map;
		}
	);

	const selectedCount = derived(
		[selectAllMode, selectedIds, notificationStore.pageData],
		([$selectAllMode, $selectedIds, $pageData]) => {
			if ($selectAllMode === "all") {
				return $pageData.total;
			} else {
				return $selectedIds.size;
			}
		}
	);

	const individualSelectionDisabled = derived(
		selectAllMode,
		($selectAllMode) => $selectAllMode === "all"
	);

	const selectionEnabled = derived(multiselectMode, ($multiselectMode) => $multiselectMode);

	return {
		// Expose stores
		selectedIds: selectedIds as Writable<Set<string>>,
		multiselectMode: multiselectMode as Writable<boolean>,
		selectAllMode: selectAllMode as Writable<SelectAllMode>,

		// Derived stores
		selectionMap: selectionMap as Readable<Map<string, boolean>>,
		selectedCount: selectedCount as Readable<number>,
		individualSelectionDisabled: individualSelectionDisabled as Readable<boolean>,
		selectionEnabled: selectionEnabled as Readable<boolean>,

		// Actions
		toggleSelection: (id: string) => {
			selectedIds.update((ids) => {
				const newIds = new Set(ids);
				if (newIds.has(id)) {
					newIds.delete(id);
				} else {
					newIds.add(id);
				}
				return newIds;
			});
		},
		clearSelection: () => {
			selectedIds.set(new Set());
			selectAllMode.set("none");
		},
		selectAllPage: () => {
			const pageData = get(notificationStore.pageData);
			const newIds = new Set<string>();

			for (const item of pageData.items) {
				const key = item.githubId ?? item.id;
				if (key) {
					newIds.add(key);
				}
			}

			selectedIds.set(newIds);
			selectAllMode.set("page");
		},
		selectAllAcrossPages: () => {
			selectedIds.set(new Set());
			selectAllMode.set("all");
		},
		clearSelectAllMode: () => {
			selectAllMode.set("none");
			selectedIds.set(new Set());
		},
		toggleMultiselectMode: () => {
			multiselectMode.update((mode) => !mode);
			// Reset select all mode when toggling multiselect off
			if (!get(multiselectMode)) {
				selectAllMode.set("none");
			}
		},
		exitMultiselectMode: () => {
			multiselectMode.set(false);
			selectedIds.set(new Set());
			selectAllMode.set("none");
		},
	};
}

export type SelectionStore = ReturnType<typeof createSelectionStore>;
