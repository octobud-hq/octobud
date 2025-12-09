<script lang="ts">
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

	import { onMount, tick } from "svelte";

	export let open = false;
	export let action = "perform this action on";
	export let count = 0;
	export let onConfirm: () => void = () => {};
	export let onCancel: () => void = () => {};
	export let confirming = false;

	let confirmButton: HTMLButtonElement | null = null;

	// Convert action names to user-friendly text
	function getActionLabel(action: string): string {
		const actionMap: Record<string, string> = {
			markRead: "mark as read",
			markUnread: "mark as unread",
			archive: "archive",
			unarchive: "unarchive",
			mute: "mute",
			unmute: "unmute",
			snooze: "snooze",
			unsnooze: "unsnooze",
			star: "star",
			unstar: "unstar",
		};
		return actionMap[action] || action;
	}

	// Generate body text with proper word order
	function getBodyText(action: string, count: number): string {
		const notificationText = `${count} notification${count === 1 ? "" : "s"}`;

		// Special case for mark read/unread - put count in the middle
		if (action === "markRead") {
			return `Are you sure you want to mark ${notificationText} as read?`;
		}
		if (action === "markUnread") {
			return `Are you sure you want to mark ${notificationText} as unread?`;
		}

		// For all other actions, use verb first
		const actionLabel = getActionLabel(action);
		return `Are you sure you want to ${actionLabel} ${notificationText}?`;
	}

	$: actionLabel = getActionLabel(action);
	$: title = `Confirm bulk ${actionLabel}`;
	$: body = getBodyText(action, count);

	// Focus the confirm button when dialog opens
	$: if (open && confirmButton) {
		tick().then(() => {
			confirmButton?.focus();
		});
	}

	const handleKey = (event: KeyboardEvent) => {
		if (!open) return;

		if (event.key === "Escape") {
			event.preventDefault();
			event.stopPropagation();
			event.stopImmediatePropagation();
			onCancel();
		} else if (event.key === "Enter" && !confirming) {
			event.preventDefault();
			event.stopPropagation();
			event.stopImmediatePropagation();
			onConfirm();
		}
	};

	onMount(() => {
		if (typeof window === "undefined") {
			return;
		}

		// Use capture phase to handle before other keyboard shortcuts
		const listenerOptions = { capture: true };
		window.addEventListener("keydown", handleKey, listenerOptions);
		return () => {
			window.removeEventListener("keydown", handleKey, listenerOptions);
		};
	});
</script>

{#if open}
	<div class="fixed inset-0 z-[1200] flex items-center justify-center px-4 py-8">
		<div
			class="absolute inset-0 bg-black/60 dark:bg-gray-950/60 backdrop-blur-sm"
			role="presentation"
			on:click={onCancel}
		></div>
		<div
			class="relative w-full max-w-md rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 text-gray-900 dark:text-gray-200 shadow-2xl"
			role="dialog"
			aria-modal="true"
			aria-label={title}
		>
			<header class="space-y-2">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-gray-300">{title}</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">{body}</p>
			</header>
			<footer class="mt-6 flex justify-end gap-3">
				<button
					type="button"
					class="rounded-full border border-transparent px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 transition hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
					on:click={onCancel}
					disabled={confirming}
				>
					Cancel
				</button>
				<button
					bind:this={confirmButton}
					type="button"
					class="rounded-full bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-indigo-500 disabled:opacity-50 cursor-pointer"
					on:click={onConfirm}
					disabled={confirming}
				>
					{confirming ? "Working…" : "Confirm"}
				</button>
			</footer>
		</div>
	</div>
{/if}
