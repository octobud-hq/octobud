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

import type { Notification } from "$lib/api/types";
import type { SelectionStore } from "../../stores/selectionStore";

/**
 * Selection Action Controller
 * Only keeps toggleSelection which has logic (extracts key from notification).
 * Other methods are exposed directly from the store.
 */
export function createSelectionActionController(
	selectionStore: SelectionStore
): (notification: Notification) => void {
	function toggleSelection(notification: Notification): void {
		const key = notification.githubId ?? notification.id;
		if (key) {
			selectionStore.toggleSelection(key);
		}
	}

	return toggleSelection;
}
