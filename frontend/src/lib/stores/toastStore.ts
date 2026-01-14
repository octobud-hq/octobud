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

import { writable } from "svelte/store";

export type ToastType = "success" | "info" | "warning" | "error";

export interface Toast {
	id: string;
	message: string;
	type: ToastType;
	duration: number;
	/** Whether this toast has an undo action */
	undoable?: boolean;
	/** Callback to execute when undo is clicked */
	onUndo?: () => void;
	/** Whether this toast has been morphed to "undone" state */
	morphedToUndone?: boolean;
}

interface ToastStore {
	toasts: Toast[];
}

function createToastStore() {
	const { subscribe, update } = writable<ToastStore>({ toasts: [] });

	let idCounter = 0;

	function show(message: string, type: ToastType = "info", duration = 3000) {
		const id = `toast-${Date.now()}-${idCounter++}`;
		const toast: Toast = { id, message, type, duration };

		update((state) => ({
			toasts: [...state.toasts, toast],
		}));

		if (duration > 0) {
			setTimeout(() => {
				dismiss(id);
			}, duration);
		}

		return id;
	}

	/**
	 * Show a toast with an undo button
	 */
	function showWithUndo(
		message: string,
		type: ToastType = "success",
		duration = 7000,
		onUndo: () => void
	) {
		const id = `toast-${Date.now()}-${idCounter++}`;
		const toast: Toast = {
			id,
			message,
			type,
			duration,
			undoable: true,
			onUndo,
		};

		update((state) => ({
			toasts: [...state.toasts, toast],
		}));

		if (duration > 0) {
			setTimeout(() => {
				dismiss(id);
			}, duration);
		}

		return id;
	}

	function dismiss(id: string) {
		update((state) => ({
			toasts: state.toasts.filter((t) => t.id !== id),
		}));
	}

	function success(message: string, duration = 3000) {
		return show(message, "success", duration);
	}

	function info(message: string, duration = 3000) {
		return show(message, "info", duration);
	}

	function warning(message: string, duration = 3000) {
		return show(message, "warning", duration);
	}

	function error(message: string, duration = 3000) {
		return show(message, "error", duration);
	}

	/**
	 * Morph an existing toast to show "Undone" state, then dismiss after a short delay
	 */
	function morphToUndone(id: string, duration = 1500) {
		update((state) => ({
			toasts: state.toasts.map((t) =>
				t.id === id
					? {
							...t,
							message: "Undone",
							morphedToUndone: true,
							undoable: false,
							onUndo: undefined,
						}
					: t
			),
		}));

		// Dismiss after the duration
		setTimeout(() => {
			dismiss(id);
		}, duration);
	}

	return {
		subscribe,
		show,
		showWithUndo,
		dismiss,
		morphToUndone,
		success,
		info,
		warning,
		error,
	};
}

export const toastStore = createToastStore();
