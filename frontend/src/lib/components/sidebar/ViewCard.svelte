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

	import { onDestroy } from "svelte";
	import type { NotificationView } from "$lib/api/types";
	import { normalizeViewIcon } from "$lib/utils/viewIcons";
	export let view: NotificationView;
	export let selected = false;
	export let showActions = true;
	export let count: number | undefined = undefined;
	export let reorderMode = false;
	export let onSelect: (slug: string) => void | Promise<void> = () => {};
	export let onEdit: (view: NotificationView) => void = () => {};
	export let onDragStart: (viewId: string) => void = () => {};
	export let onDragOver: (e: DragEvent, viewId: string) => void = () => {};
	export let onDrop: (e: DragEvent, viewId: string) => void = () => {};
	export let onDragEnd: () => void = () => {};
	export let isDraggedOver = false;

	let hovered = false;
	let actionsOpen = false;
	let actionsButton: HTMLButtonElement | null = null;
	let actionsMenu: HTMLDivElement | null = null;
	let detachListeners: (() => void) | null = null;
	let displayIcon = normalizeViewIcon(view.icon);
	$: displayIcon = normalizeViewIcon(view.icon);
	$: hasDescription = typeof view.description === "string" && view.description.trim().length > 0;

	function handleClick() {
		if (!reorderMode) {
			void onSelect(view.slug);
		}
	}

	function toggleActions() {
		actionsOpen = !actionsOpen;
		if (actionsOpen) {
			attachOutsideListeners();
		} else {
			detachOutsideListeners();
		}
	}

	function closeActions() {
		actionsOpen = false;
		hovered = false;
		detachOutsideListeners();
	}

	function handleActionEdit() {
		onEdit(view);
		closeActions();
	}

	function handleBlur(event: FocusEvent) {
		const related = event.relatedTarget as Node | null;
		if (!actionsButton?.contains(related) && !actionsMenu?.contains(related as Node)) {
			closeActions();
		}
	}

	function handleDocumentClick(event: MouseEvent) {
		const target = event.target as Node | null;
		if (actionsButton?.contains(target) || actionsMenu?.contains(target)) {
			return;
		}
		closeActions();
	}

	function handleDocumentKeydown(event: KeyboardEvent) {
		if (event.key === "Escape") {
			closeActions();
		}
	}

	function attachOutsideListeners() {
		if (typeof window === "undefined") return;
		const clickListener = (event: MouseEvent) => handleDocumentClick(event);
		const keyListener = (event: KeyboardEvent) => handleDocumentKeydown(event);
		window.addEventListener("mousedown", clickListener);
		window.addEventListener("keydown", keyListener);
		detachListeners = () => {
			window.removeEventListener("mousedown", clickListener);
			window.removeEventListener("keydown", keyListener);
			detachListeners = null;
		};
	}

	function detachOutsideListeners() {
		if (detachListeners) {
			detachListeners();
		}
	}

	onDestroy(() => {
		detachOutsideListeners();
	});

	const menuIcon =
		"M6 12a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0Zm7.5 0a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0Zm7.5 0a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0Z";

	// Drag handle icon (grip-vertical)
	const dragHandleIcon =
		"M9 5a1 1 0 1 1-2 0 1 1 0 0 1 2 0Zm0 4a1 1 0 1 1-2 0 1 1 0 0 1 2 0Zm0 4a1 1 0 1 1-2 0 1 1 0 0 1 2 0Zm4-8a1 1 0 1 1-2 0 1 1 0 0 1 2 0Zm0 4a1 1 0 1 1-2 0 1 1 0 0 1 2 0Zm0 4a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z";

	function handleDragStart(e: DragEvent) {
		if (!reorderMode) return;
		e.dataTransfer!.effectAllowed = "move";
		e.dataTransfer!.setData("text/plain", view.id);
		onDragStart(view.id);
	}

	function handleDragOverEvent(e: DragEvent) {
		if (!reorderMode) return;
		e.preventDefault();
		e.dataTransfer!.dropEffect = "move";
		onDragOver(e, view.id);
	}

	function handleDropEvent(e: DragEvent) {
		if (!reorderMode) return;
		e.preventDefault();
		onDrop(e, view.id);
	}

	function handleDragEndEvent() {
		if (!reorderMode) return;
		onDragEnd();
	}
</script>

<li
	draggable={reorderMode}
	on:dragstart={handleDragStart}
	on:dragover={handleDragOverEvent}
	on:drop={handleDropEvent}
	on:dragend={handleDragEndEvent}
>
	<div
		class={`group relative flex w-full items-center rounded-md border transition ${
			selected
				? "bg-indigo-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100"
				: "bg-transparent text-gray-900 hover:bg-gray-200 dark:text-gray-100 dark:hover:bg-gray-800/50"
		} ${isDraggedOver ? "border-blue-500 bg-blue-50 dark:border-blue-400 dark:bg-blue-900/20" : "border-transparent"} ${reorderMode ? "cursor-grab active:cursor-grabbing" : ""}`}
		role="group"
		on:mouseenter={() => {
			hovered = true;
		}}
		on:mouseleave={() => {
			hovered = false;
		}}
	>
		{#if reorderMode}
			<div
				class="flex items-center justify-center px-2 py-1.5 text-gray-500 dark:text-gray-400"
				aria-label="Drag to reorder"
			>
				<svg class="h-4 w-4" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
					<path d={dragHandleIcon} />
				</svg>
			</div>
		{/if}
		<div
			role="button"
			tabindex={reorderMode ? -1 : 0}
			class={`flex-1 px-2 py-1.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 ${reorderMode ? "" : "cursor-pointer"}`}
			aria-pressed={selected}
			on:click={handleClick}
			on:keydown={(event) => {
				if (!reorderMode && (event.key === "Enter" || event.key === " ")) {
					event.preventDefault();
					handleClick();
				}
			}}
		>
			<div class={`flex ${hasDescription ? "items-start" : "items-center"} gap-2 pr-10`}>
				<span
					class={`flex flex-shrink-0 items-center justify-center text-base ${
						selected ? "text-gray-900 dark:text-gray-100" : "text-gray-700 dark:text-gray-300"
					} ${hasDescription ? "mt-0.5" : ""}`}
					aria-hidden="true"
				>
					{displayIcon}
				</span>
				<div class="flex-1 min-w-0">
					<div class="flex items-center gap-1.5">
						<span class="text-[14px] font-medium truncate text-gray-700 dark:text-gray-100">
							{view.name}
						</span>
					</div>
					{#if hasDescription}
						<p
							class={` pr-8 text-xs ${
								selected ? "text-gray-600 dark:text-gray-400" : "text-gray-600 dark:text-gray-400"
							}`}
						>
							{view.description}
						</p>
					{/if}
				</div>
			</div>
		</div>
		{#if count !== undefined && count > 0 && !(showActions && hovered && !view.isDefault) && !reorderMode}
			<span
				class={`absolute right-2 top-1/2 flex h-6 -translate-y-1/2 items-center justify-center rounded-full px-2 text-[11px] font-semibold transition-opacity delay-200 ${
					selected
						? "bg-indigo-200 text-gray-700 dark:bg-gray-700 dark:text-gray-100"
						: "bg-gray-200 text-gray-700 dark:bg-gray-800/60 dark:text-gray-100"
				}`}
				aria-label={`${count} new notification${count === 1 ? "" : "s"}`}
			>
				{count}
			</span>
		{/if}
		{#if showActions && (hovered || actionsOpen) && !view.isDefault && !reorderMode}
			<button
				type="button"
				class="absolute right-2 top-1/2 flex h-6 w-6 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border border-transparent bg-gray-200/80 text-gray-600 transition hover:bg-gray-300/80 hover:text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 dark:bg-gray-900/80 dark:text-gray-400 dark:hover:bg-gray-800/80 dark:hover:text-gray-300"
				aria-label={`Edit view ${view.name}`}
				on:click={toggleActions}
				on:blur={handleBlur}
				bind:this={actionsButton}
			>
				<svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
					<path d={menuIcon} />
				</svg>
			</button>
			{#if actionsOpen}
				<div
					class="absolute right-0 top-[calc(100%+0.25rem)] z-50 w-40 rounded-lg border border-gray-200 bg-white text-sm text-gray-700 shadow-soft dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300"
					role="menu"
					tabindex="-1"
					on:focusout={handleBlur}
					bind:this={actionsMenu}
					on:mouseleave={(event) => {
						const related = event.relatedTarget as Node | null;
						if (actionsButton?.contains(related) || actionsMenu?.contains(related)) {
							return;
						}
						closeActions();
					}}
					on:mouseenter={() => {
						hovered = true;
					}}
				>
					<button
						type="button"
						class="flex w-full cursor-pointer items-center justify-start gap-2 px-3 py-2 text-left text-gray-700 transition hover:bg-gray-50 focus-visible:outline-none focus-visible:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-900/50 dark:focus-visible:bg-gray-900/50"
						on:click={handleActionEdit}
					>
						<span>Edit view</span>
					</button>
				</div>
			{/if}
		{/if}
	</div>
</li>
