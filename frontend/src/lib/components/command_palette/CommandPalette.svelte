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

	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/stores";
	import { onDestroy, onMount, tick } from "svelte";
	import { get, derived } from "svelte/store";
	import type { NotificationView, Tag } from "$lib/api/types";
	import {
		createCommandPaletteController,
		type CommandPaletteCommandDefinition,
		type CommandPaletteCommandId,
		type BulkAction,
		getMatchingBulkActions,
	} from "$lib/state/commandPaletteController";
	import {
		getMatchingViews,
		addIndicesToMatches,
		separateViewMatches,
		viewKey,
		type ViewMatch,
	} from "$lib/utils/viewMatching";
	import CommandPaletteDropdown from "./CommandPaletteDropdown.svelte";
	import RootCommandList from "./RootCommandList.svelte";
	import ViewSelectionList from "./ViewSelectionList.svelte";
	import BulkActionSelectionList from "./BulkActionSelectionList.svelte";
	import UnknownCommandError from "./UnknownCommandError.svelte";

	export let inputEl: HTMLInputElement | null = null;
	export let placeholder = "Run a command";
	export let detailOpen = false;
	export let builtInViews: NotificationView[] = [];
	export let views: NotificationView[] = [];
	export let tags: Tag[] = [];
	export let selectedViewSlug = "";
	export let defaultViewSlug = "";
	export let onSelectView: (slug: string) => void | Promise<void> = () => {};
	export let onRefresh: () => void | Promise<void> = () => {};
	export let onShowMoreShortcuts: () => void = () => {};
	export let onShowQueryGuide: () => void = () => {};
	export let onMarkAllRead: () => void | Promise<void> = () => {};
	export let onMarkAllUnread: () => void | Promise<void> = () => {};
	export let onMarkAllArchive: () => void | Promise<void> = () => {};

	const MAX_VIEW_RESULTS = 12;

	const controller = createCommandPaletteController();
	const {
		stores: { isOpen, inputValue },
		derived: { parsed, activeCommandId, commandToken, commandQuery, rootCommandMatches },
		actions: { open, openCommand, close, resetInput, setInputValue },
	} = controller;

	// Extract isPartialCommand from parsed
	$: isPartialCommand = $parsed.isPartialCommand;

	let container: HTMLElement | null = null;
	let lastSearchQuery = "";
	let previousCommandId: CommandPaletteCommandId | null = null;

	// View matching
	$: normalizedViewQuery = $commandQuery.trim().toLowerCase();

	// Create tag views from tags
	$: tagViews = (tags ?? []).map(
		(tag): NotificationView => ({
			id: `tag-${tag.id}`,
			name: tag.name,
			slug: `tag-${tag.slug}`,
			description: tag.description || `View notifications tagged with ${tag.name}`,
			query: `tags:${tag.slug}`,
			unreadCount: tag.unreadCount ?? 0,
		})
	);

	// Create settings view
	const settingsView = {
		id: "settings",
		name: "Settings",
		slug: "settings",
		description: "Manage your preferences and configuration",
		query: "",
		unreadCount: 0,
	} as NotificationView;

	// Filter tag views and settings view based on query
	$: filteredTagViews = normalizedViewQuery
		? tagViews.filter((view) => {
				const haystack = `${view.name} ${view.slug} ${view.description ?? ""}`.toLowerCase();
				return haystack.includes(normalizedViewQuery);
			})
		: tagViews;

	$: settingsViewMatchesQuery =
		!normalizedViewQuery ||
		"settings".includes(normalizedViewQuery) ||
		settingsView.name.toLowerCase().includes(normalizedViewQuery) ||
		(settingsView.description?.toLowerCase().includes(normalizedViewQuery) ?? false);

	// Get base view matches
	$: baseViewMatches = getMatchingViews(builtInViews, views, normalizedViewQuery, MAX_VIEW_RESULTS);

	// Add tag views and settings view
	$: allViewMatches = [
		...baseViewMatches,
		...filteredTagViews.map((view) => ({ view, source: "tag" as const })),
		...(settingsViewMatchesQuery ? [{ view: settingsView, source: "settings" as const }] : []),
	] as ViewMatch[];

	$: viewMatchEntries = addIndicesToMatches(allViewMatches);
	$: separatedMatches = separateViewMatches(viewMatchEntries);
	$: builtInViewMatches = separatedMatches.builtIn;
	$: savedViewMatches = separatedMatches.saved;
	$: tagViewMatches = separatedMatches.tag;
	$: settingsViewMatches = separatedMatches.settings;

	// Determine effective selected slug - handle settings and tag views specially
	$: isSettingsRoute = $page.url.pathname.startsWith("/settings");
	$: isTagViewRoute = $page.url.pathname.startsWith("/views/tag-");
	$: tagSlugFromRoute = isTagViewRoute ? $page.url.pathname.replace("/views/", "") : null;

	$: effectiveSelectedSlug = isSettingsRoute
		? "settings"
		: (tagSlugFromRoute ?? (selectedViewSlug || defaultViewSlug || ""));
	$: unknownCommandToken =
		$commandToken && !$activeCommandId && !isPartialCommand ? $commandToken : null;

	// Bulk action matching
	$: normalizedBulkQuery = $commandQuery.trim().toLowerCase();
	$: bulkActionMatches = getMatchingBulkActions(normalizedBulkQuery);

	// Navigation indices (simplified - actual management can stay in component)
	let rootCommandIndex = 0;
	let viewCommandIndex = 0;
	let bulkActionIndex = 0;
	let lastRootQuery = "";
	let lastViewQuery = "";
	let lastBulkQuery = "";

	$: rootCommandMatchesSnapshot = $rootCommandMatches;

	$: if (!$isOpen) {
		rootCommandIndex = 0;
		lastRootQuery = "";
		viewCommandIndex = 0;
		lastViewQuery = "";
		bulkActionIndex = 0;
		lastBulkQuery = "";
	}

	$: if (!$activeCommandId && !unknownCommandToken) {
		const currentRootQuery = $parsed.rootQuery;
		if (currentRootQuery !== lastRootQuery) {
			rootCommandIndex = 0;
			lastRootQuery = currentRootQuery;
		}
	}

	$: if ($activeCommandId !== "view") {
		viewCommandIndex = 0;
	}

	$: if ($activeCommandId !== "bulk") {
		bulkActionIndex = 0;
	}

	$: if ($activeCommandId === "view") {
		const currentViewQuery = normalizedViewQuery;
		if (currentViewQuery !== lastViewQuery) {
			viewCommandIndex = 0;
			lastViewQuery = currentViewQuery;
		}
		const count = viewMatchEntries.length;
		if (count === 0) {
			viewCommandIndex = -1;
		} else if (viewCommandIndex < 0 || viewCommandIndex >= count) {
			viewCommandIndex = Math.max(0, Math.min(viewCommandIndex, count - 1));
		}
	}

	$: if ($activeCommandId === "bulk") {
		const currentBulkQuery = normalizedBulkQuery;
		if (currentBulkQuery !== lastBulkQuery) {
			bulkActionIndex = 0;
			lastBulkQuery = currentBulkQuery;
		}
		const count = bulkActionMatches.length;
		if (count === 0) {
			bulkActionIndex = -1;
		} else if (bulkActionIndex < 0 || bulkActionIndex >= count) {
			bulkActionIndex = Math.max(0, Math.min(bulkActionIndex, count - 1));
		}
	}

	$: {
		if (!$activeCommandId && !unknownCommandToken) {
			const count = rootCommandMatchesSnapshot.length;
			if (count === 0) {
				rootCommandIndex = -1;
			} else if (rootCommandIndex < 0 || rootCommandIndex >= count) {
				rootCommandIndex = Math.max(0, Math.min(rootCommandIndex, count - 1));
			}
		} else if (rootCommandIndex !== 0) {
			rootCommandIndex = 0;
		}
	}

	// Auto-scroll for root commands
	$: if ($isOpen && rootCommandIndex >= 0 && !$activeCommandId && !unknownCommandToken) {
		void tick().then(() => {
			const element = container?.querySelector(`[data-root-command-index="${rootCommandIndex}"]`);
			if (element) {
				element.scrollIntoView({
					block: "nearest",
					behavior: "smooth",
				});
			}
		});
	}

	// Auto-scroll for view selections
	$: if ($isOpen && $activeCommandId === "view" && viewCommandIndex >= 0) {
		void tick().then(() => {
			const element = container?.querySelector(`[data-view-index="${viewCommandIndex}"]`);
			if (element) {
				element.scrollIntoView({
					block: "nearest",
					behavior: "smooth",
				});
			}
		});
	}

	// Auto-scroll for bulk action selections
	$: if ($isOpen && $activeCommandId === "bulk" && bulkActionIndex >= 0) {
		void tick().then(() => {
			const element = container?.querySelector(`[data-bulk-action-index="${bulkActionIndex}"]`);
			if (element) {
				element.scrollIntoView({
					block: "nearest",
					behavior: "smooth",
				});
			}
		});
	}

	function focusInputAtEnd() {
		if (!inputEl) {
			return;
		}
		const length = inputEl.value.length;
		inputEl.focus();
		inputEl.setSelectionRange(length, length);
	}

	async function openPaletteForCommand(commandId: CommandPaletteCommandId) {
		openCommand(commandId);
		if (commandId === "view") {
			viewCommandIndex = 0;
			lastViewQuery = normalizedViewQuery;
		}
		await tick();
		focusInputAtEnd();
	}

	async function openPaletteRootPrompt() {
		open();
		setInputValue(">");
		rootCommandIndex = 0;
		lastRootQuery = "";
		await tick();
		focusInputAtEnd();
	}

	function closePaletteInternal() {
		close();
		inputEl?.blur();
	}

	function moveRootCommandSelection(delta: number) {
		const matches = rootCommandMatchesSnapshot;
		if (matches.length === 0) {
			return;
		}
		const baseIndex = rootCommandIndex < 0 ? 0 : rootCommandIndex;
		const next = (baseIndex + delta + matches.length) % matches.length;
		rootCommandIndex = next;
	}

	function handleRootCommandPointer(index: number) {
		if (index < 0 || index >= rootCommandMatchesSnapshot.length) {
			return;
		}
		rootCommandIndex = index;
	}

	function moveViewCommandSelection(delta: number) {
		const matches = viewMatchEntries;
		if (matches.length === 0) {
			return;
		}
		let nextIndex: number;
		if (viewCommandIndex === -1) {
			nextIndex = delta > 0 ? 0 : matches.length - 1;
		} else {
			nextIndex = viewCommandIndex + delta;
			if (nextIndex < 0) {
				nextIndex = matches.length - 1;
			} else if (nextIndex >= matches.length) {
				nextIndex = 0;
			}
		}
		viewCommandIndex = nextIndex;
	}

	function handleViewPointer(index: number) {
		if (index < 0 || index >= viewMatchEntries.length) {
			return;
		}
		viewCommandIndex = index;
	}

	function moveBulkActionSelection(delta: number) {
		const matches = bulkActionMatches;
		if (matches.length === 0) {
			return;
		}
		let nextIndex: number;
		if (bulkActionIndex === -1) {
			nextIndex = delta > 0 ? 0 : matches.length - 1;
		} else {
			nextIndex = bulkActionIndex + delta;
			if (nextIndex < 0) {
				nextIndex = matches.length - 1;
			} else if (nextIndex >= matches.length) {
				nextIndex = 0;
			}
		}
		bulkActionIndex = nextIndex;
	}

	function handleBulkActionPointer(index: number) {
		if (index < 0 || index >= bulkActionMatches.length) {
			return;
		}
		bulkActionIndex = index;
	}

	function handleFocus() {
		if (!$isOpen) {
			open();
		}
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === "Escape") {
			event.preventDefault();
			event.stopPropagation();
			const activeElement = typeof document !== "undefined" ? document.activeElement : null;
			if (inputEl && activeElement === inputEl) {
				inputEl.blur();
			}
			closePaletteInternal();
			return;
		}

		if (
			!$activeCommandId &&
			!unknownCommandToken &&
			(event.key === "ArrowDown" || event.key === "ArrowUp")
		) {
			event.preventDefault();
			moveRootCommandSelection(event.key === "ArrowDown" ? 1 : -1);
			return;
		}

		if ($activeCommandId === "view" && (event.key === "ArrowDown" || event.key === "ArrowUp")) {
			event.preventDefault();
			moveViewCommandSelection(event.key === "ArrowDown" ? 1 : -1);
			return;
		}

		if ($activeCommandId === "bulk" && (event.key === "ArrowDown" || event.key === "ArrowUp")) {
			event.preventDefault();
			moveBulkActionSelection(event.key === "ArrowDown" ? 1 : -1);
			return;
		}

		if (event.key === "Enter") {
			event.preventDefault();
			handleSubmit();
		}
	}

	function handleSubmit() {
		if (!$isOpen) {
			return;
		}

		if (!$activeCommandId && !unknownCommandToken) {
			const matches = rootCommandMatchesSnapshot;
			if (matches.length === 0) {
				return;
			}
			const index = rootCommandIndex >= 0 ? rootCommandIndex : 0;
			const selected = matches[index] ?? matches[0];
			if (selected) {
				handleCommandClick(selected, index);
			}
			return;
		}

		if ($activeCommandId === "view") {
			const matches = viewMatchEntries;
			if (matches.length === 0) {
				return;
			}
			const index = viewCommandIndex >= 0 ? viewCommandIndex : 0;
			const selected = matches[index] ?? matches[0];
			if (selected) {
				void handleViewSelect(selected.view, selected.index);
			}
			return;
		}

		if ($activeCommandId === "bulk") {
			const matches = bulkActionMatches;
			if (matches.length === 0) {
				return;
			}
			const index = bulkActionIndex >= 0 ? bulkActionIndex : 0;
			const selected = matches[index] ?? matches[0];
			if (selected) {
				void handleBulkActionSelect(selected, index);
			}
			return;
		}

		if ($activeCommandId === "shortcuts") {
			closePaletteInternal();
			onShowMoreShortcuts();
			return;
		}

		if ($activeCommandId === "query") {
			closePaletteInternal();
			onShowQueryGuide();
			return;
		}

		const firstCommand = rootCommandMatchesSnapshot[0];
		if (firstCommand) {
			handleCommandClick(firstCommand, 0);
		}
	}

	function handleClearClick() {
		resetInput();
		lastSearchQuery = "";
		void tick().then(() => {
			inputEl?.focus();
		});
	}

	function handleCommandClick(command: CommandPaletteCommandDefinition, index?: number) {
		if (typeof index === "number") {
			rootCommandIndex = index;
		}
		switch (command.id) {
			case "view":
				void openPaletteForCommand("view");
				break;
			case "shortcuts":
				closePaletteInternal();
				onShowMoreShortcuts();
				break;
			case "query":
				closePaletteInternal();
				onShowQueryGuide();
				break;
			case "bulk":
				void openPaletteForCommand("bulk");
				break;
		}
	}

	async function handleRefresh() {
		try {
			const result = onRefresh?.();
			if (result && typeof (result as Promise<unknown>).then === "function") {
				await (result as Promise<unknown>);
			}
		} finally {
			closePaletteInternal();
		}
	}

	async function handleBulkActionSelect(action: BulkAction, index?: number) {
		if (typeof index === "number") {
			bulkActionIndex = index;
		}

		try {
			let result: void | Promise<void>;
			switch (action.id) {
				case "mark_all_read":
					result = onMarkAllRead?.();
					break;
				case "mark_all_unread":
					result = onMarkAllUnread?.();
					break;
				case "mark_all_archive":
					result = onMarkAllArchive?.();
					break;
			}

			if (result && typeof (result as Promise<unknown>).then === "function") {
				await (result as Promise<unknown>);
			}
		} finally {
			closePaletteInternal();
		}
	}

	async function handleViewSelect(view: NotificationView, index?: number) {
		if (typeof index === "number") {
			viewCommandIndex = index;
		}

		// Handle settings view specially - navigate to /settings
		if (view.slug === "settings") {
			closePaletteInternal();
			await goto(resolve("/settings"));
			return;
		}

		// Handle tag views - they use the slug directly
		const outcome = onSelectView?.(view.slug);
		if (outcome && typeof (outcome as Promise<unknown>).then === "function") {
			await outcome;
		}
		closePaletteInternal();
	}

	function handleDocumentPointerDown(event: PointerEvent) {
		if (!container || !$isOpen) {
			return;
		}
		const target = event.target as Node | null;
		if (target && container.contains(target)) {
			return;
		}
		if (detailOpen) {
			return;
		}
		closePaletteInternal();
	}

	onMount(() => {
		if (typeof document !== "undefined") {
			document.addEventListener("pointerdown", handleDocumentPointerDown);
		}
	});

	onDestroy(() => {
		if (typeof document !== "undefined") {
			document.removeEventListener("pointerdown", handleDocumentPointerDown);
		}
	});

	export function openWithCommand(commandId: CommandPaletteCommandId = "view") {
		void openPaletteForCommand(commandId);
	}

	export function focusInput() {
		// Focus the input to open palette in empty state
		inputEl?.focus();
	}

	export function closePalette() {
		closePaletteInternal();
	}

	export function openWithPrompt() {
		void openPaletteRootPrompt();
	}

	export function isPaletteOpen(): boolean {
		return get(isOpen);
	}
</script>

<div class="relative" bind:this={container}>
	<div
		class="flex items-center gap-2 rounded-full border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 px-3 py-1 text-sm transition-all duration-200 hover:bg-gray-100 dark:hover:bg-gray-900/50 focus-within:border-blue-600 focus-within:ring-2 focus-within:ring-blue-600/30 focus-within:bg-white dark:focus-within:bg-gray-900"
	>
		<span class="hidden text-gray-600 dark:text-gray-400 sm:inline">⌘⇧K</span>
		<input
			class="w-full appearance-none text-[16px] border-none bg-transparent outline-none focus:ring-0"
			type="text"
			{placeholder}
			bind:this={inputEl}
			value={$inputValue}
			on:input={(e) => setInputValue(e.currentTarget.value)}
			on:focus={handleFocus}
			on:keydown={handleKeyDown}
		/>
		{#if $inputValue.length > 0}
			<button
				type="button"
				class="rounded-full bg-gray-100 dark:bg-gray-800 px-2 py-1 text-xs text-gray-600 dark:text-gray-400 transition hover:bg-gray-200 dark:hover:bg-gray-700 hover:text-gray-800 dark:hover:text-gray-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 cursor-pointer"
				on:click={handleClearClick}
			>
				Clear
			</button>
		{/if}
	</div>

	<CommandPaletteDropdown isOpen={$isOpen}>
		{#if $activeCommandId === "view"}
			<ViewSelectionList
				{viewMatchEntries}
				{builtInViewMatches}
				{savedViewMatches}
				{tagViewMatches}
				{settingsViewMatches}
				selectedIndex={viewCommandIndex}
				{effectiveSelectedSlug}
				commandQuery={$commandQuery}
				onSelect={(entry) => void handleViewSelect(entry.view, entry.index)}
				onPointerMove={handleViewPointer}
			/>
		{:else if $activeCommandId === "bulk"}
			<BulkActionSelectionList
				bulkActions={bulkActionMatches}
				selectedIndex={bulkActionIndex}
				commandQuery={$commandQuery}
				onSelect={handleBulkActionSelect}
				onPointerMove={handleBulkActionPointer}
			/>
		{:else if unknownCommandToken}
			<UnknownCommandError
				commandToken={unknownCommandToken}
				onSelectCommand={(commandId) =>
					void openPaletteForCommand(commandId as CommandPaletteCommandId)}
			/>
		{:else}
			<RootCommandList
				commands={rootCommandMatchesSnapshot}
				selectedIndex={rootCommandIndex}
				rootQuery={$parsed.rootQuery}
				onSelect={handleCommandClick}
				onPointerMove={handleRootCommandPointer}
			/>
		{/if}
	</CommandPaletteDropdown>
</div>

<style>
	:global(body) {
		--global-search-backdrop: rgba(15, 23, 42, 0.92);
	}
</style>
