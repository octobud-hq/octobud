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

import { writable, get } from "svelte/store";
import type { Writable } from "svelte/store";

/**
 * Generic composable for drag-and-drop reordering functionality.
 *
 * @template T - The type of items being reordered (must have an `id` property)
 * @param initialItems - Initial array of items to reorder
 * @param onSave - Callback function to save the new order. Receives array of item IDs in new order.
 * @param options - Optional configuration
 * @returns Reactive state and event handlers for drag-and-drop reordering
 */
export function useDragAndDropReorder<T extends { id: string }>(
	initialItems: T[],
	onSave: (itemIds: string[]) => Promise<void>,
	options: {
		onSaveSuccess?: () => void;
		onSaveError?: (error: unknown) => void;
		onCancel?: () => void;
	} = {}
) {
	const reorderMode = writable(false);
	const draggedId = writable<string | null>(null);
	const draggedOverId = writable<string | null>(null);
	const localOrder = writable<T[]>([...initialItems]);
	const originalOrder = writable<T[]>([...initialItems]);
	const reordering = writable(false);

	/**
	 * Toggle reorder mode on/off
	 */
	function toggleReorderMode() {
		const currentMode = get(reorderMode);
		if (currentMode) {
			// Exit reorder mode - save changes
			void handleSaveReorder();
		} else {
			// Enter reorder mode
			const currentItems = get(localOrder);
			reorderMode.set(true);
			originalOrder.set([...currentItems]);
			localOrder.set([...currentItems]);
		}
	}

	/**
	 * Save the current reorder state
	 */
	async function handleSaveReorder() {
		reordering.set(true);
		try {
			const currentOrder = get(localOrder);
			const itemIds = currentOrder.map((item) => item.id);
			await onSave(itemIds);
			options.onSaveSuccess?.();
			reorderMode.set(false);
		} catch (error) {
			// Revert to original order on error
			const original = get(originalOrder);
			localOrder.set([...original]);
			options.onSaveError?.(error);
		} finally {
			reordering.set(false);
			draggedId.set(null);
			draggedOverId.set(null);
		}
	}

	/**
	 * Cancel reorder mode and revert to original order
	 */
	function cancelReorder() {
		const original = get(originalOrder);
		localOrder.set([...original]);
		reorderMode.set(false);
		draggedId.set(null);
		draggedOverId.set(null);
		options.onCancel?.();
	}

	/**
	 * Handle drag start event
	 */
	function handleDragStart(itemId: string, event?: DragEvent) {
		if (event?.dataTransfer) {
			event.dataTransfer.effectAllowed = "move";
		}
		draggedId.set(itemId);
	}

	/**
	 * Handle drag over event
	 */
	function handleDragOver(targetId: string, event?: DragEvent) {
		const currentDraggedId = get(draggedId);
		if (!currentDraggedId || currentDraggedId === targetId) {
			draggedOverId.set(null);
			return;
		}
		if (event) {
			event.preventDefault();
		}
		draggedOverId.set(targetId);
	}

	/**
	 * Handle drop event
	 */
	function handleDrop(targetId: string, event?: DragEvent) {
		const currentDraggedId = get(draggedId);
		if (!currentDraggedId || currentDraggedId === targetId) {
			draggedOverId.set(null);
			return;
		}
		if (event) {
			event.preventDefault();
		}

		// Reorder the local list
		const currentOrder = get(localOrder);
		const draggedIndex = currentOrder.findIndex((item) => item.id === currentDraggedId);
		const targetIndex = currentOrder.findIndex((item) => item.id === targetId);

		if (draggedIndex !== -1 && targetIndex !== -1) {
			const newOrder = [...currentOrder];
			const [draggedItem] = newOrder.splice(draggedIndex, 1);
			newOrder.splice(targetIndex, 0, draggedItem);
			localOrder.set(newOrder);
		}

		draggedOverId.set(null);
	}

	/**
	 * Handle drag end event
	 */
	function handleDragEnd() {
		draggedId.set(null);
		draggedOverId.set(null);
	}

	/**
	 * Update the items (e.g., when the source list changes)
	 */
	function updateItems(newItems: T[]) {
		const currentMode = get(reorderMode);
		if (!currentMode) {
			localOrder.set([...newItems]);
			originalOrder.set([...newItems]);
		}
		// If in reorder mode, don't update (user is manually ordering)
	}

	return {
		// Stores
		reorderMode: reorderMode as Writable<boolean>,
		draggedId: draggedId as Writable<string | null>,
		draggedOverId: draggedOverId as Writable<string | null>,
		localOrder: localOrder as Writable<T[]>,
		reordering: reordering as Writable<boolean>,

		// Actions
		toggleReorderMode,
		handleSaveReorder,
		cancelReorder,
		handleDragStart,
		handleDragOver,
		handleDrop,
		handleDragEnd,
		updateItems,
	};
}
