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

	export let isOpen = false;
	export let buttonElement: HTMLElement | null = null;
	export let builtInViews: NotificationView[] = [];
	export let userViews: NotificationView[] = [];
	export let selectedViewId: string = "";
	export let selectedViewSlug: string = "";
	export let inboxView: NotificationView | undefined = undefined;
	export let onSelectView: (slug: string) => void | Promise<void> = () => {};
	export let onNewView: () => void = () => {};
	export let onClose: () => void = () => {};

	let dropdownElement: HTMLDivElement | null = null;
	let detachListeners: (() => void) | null = null;

	// Built-in view icon paths (matching BuiltInViewCard)
	const builtInIconPaths = {
		inbox: [
			"M3 12V15.8C3 16.9201 3 17.4802 3.21799 17.908C3.40973 18.2843 3.71569 18.5903 4.09202 18.782C4.51984 19 5.0799 19 6.2 19H17.8C18.9201 19 19.4802 19 19.908 18.782C20.2843 18.5903 20.5903 18.2843 20.782 17.908C21 17.4802 21 16.9201 21 15.8V12M3 12H6.67452C7.16369 12 7.40829 12 7.63846 12.0553C7.84254 12.1043 8.03763 12.1851 8.21657 12.2947C8.4184 12.4184 8.59136 12.5914 8.93726 12.9373L9.06274 13.0627C9.40865 13.4086 9.5816 13.5816 9.78343 13.7053C9.96237 13.8149 10.1575 13.8957 10.3615 13.9447C10.5917 14 10.8363 14 11.3255 14H12.6745C13.1637 14 13.4083 14 13.6385 13.9447C13.8425 13.8957 14.0376 13.8149 14.2166 13.7053C14.4184 13.5816 14.5914 13.4086 14.9373 13.0627L15.0627 12.9373C15.4086 12.5914 15.5816 12.4184 15.7834 12.2947C15.9624 12.1851 16.1575 12.1043 16.3615 12.0553C16.5917 12 16.8363 12 17.3255 12H21M3 12L5.32639 6.83025C5.78752 5.8055 6.0181 5.29312 6.38026 4.91755C6.70041 4.58556 7.09278 4.33186 7.52691 4.17615C8.01802 4 8.57988 4 9.70361 4H14.2964C15.4201 4 15.982 4 16.4731 4.17615C16.9072 4.33186 17.2996 4.58556 17.6197 4.91755C17.9819 5.29312 18.2125 5.8055 18.6736 6.83025L21 12",
		],
		infinity: [
			"M252,128a59.99981,59.99981,0,0,1-102.42627,42.42627q-.25709-.25708-.49805-.5293L89.21533,102.30615a36,36,0,1,0,0,51.3877L92.30078,150.21a12,12,0,0,1,17.9668,15.91211l-3.34326,3.7749q-.241.27173-.498.5293a60,60,0,1,1,0-84.85254q.25709.25709.498.5293l59.86035,67.59082a36,36,0,1,0,0-51.3877L163.69922,105.79a12,12,0,0,1-17.9668-15.91211l3.34326-3.7749q-.241-.27174.49805-.5293A60,60,0,0,1,252,128Z",
		],
		check: ["M5 14L8.23309 16.4248C8.66178 16.7463 9.26772 16.6728 9.60705 16.2581L18 6"],
		archive: [
			"M2 12c0-4.714 0-7.071 1.464-8.536C4.93 2 7.286 2 12 2c4.714 0 7.071 0 8.535 1.464C22 4.93 22 7.286 22 12",
			"M2 14c0-2.8 0-4.2.545-5.27A5 5 0 0 1 4.73 6.545C5.8 6 7.2 6 10 6h4c2.8 0 4.2 0 5.27.545a5 5 0 0 1 2.185 2.185C22 9.8 22 11.2 22 14c0 2.8 0 4.2-.545 5.27a5 5 0 0 1-2.185 2.185C18.2 22 16.8 22 14 22h-4c-2.8 0-4.2 0-5.27-.545a5 5 0 0 1-2.185-2.185C2 18.2 2 16.8 2 14Z",
			"M12 11v6m0 0 2.5-2.5M12 17l-2.5-2.5",
		],
		snooze: [
			"M12 6V12L16 14M22 12C22 17.5228 17.5228 22 12 22C6.47715 22 2 17.5228 2 12C2 6.47715 6.47715 2 12 2C17.5228 2 22 6.47715 22 12Z",
		],
	} as const;

	// Get icon type for built-in views
	function getBuiltInIconType(
		slug: string
	): "inbox" | "infinity" | "check" | "archive" | "snooze" | null {
		if (slug === "inbox") return "inbox";
		if (slug === "archive") return "archive";
		if (slug === "snoozed") return "snooze";
		if (slug === "done") return "check";
		if (slug === "everything") return "infinity";
		return null;
	}

	// Position dropdown
	$: if (isOpen && buttonElement && dropdownElement) {
		positionDropdown();
	}

	function positionDropdown() {
		if (!buttonElement || !dropdownElement) return;

		const buttonRect = buttonElement.getBoundingClientRect();
		const viewportHeight = window.innerHeight;
		const bottomMargin = 16; // Margin from bottom of viewport
		const topMargin = 16; // Margin from top of viewport

		// Calculate available space
		const spaceBelow = viewportHeight - buttonRect.bottom - bottomMargin;
		const spaceAbove = buttonRect.top - topMargin;

		// Determine position and max-height
		if (spaceBelow >= 200 || spaceBelow >= spaceAbove) {
			// Position below
			dropdownElement.style.top = `${buttonRect.bottom + 8}px`;
			dropdownElement.style.maxHeight = `${spaceBelow - 8}px`;
			dropdownElement.style.bottom = "auto";
		} else {
			// Position above
			dropdownElement.style.bottom = `${viewportHeight - buttonRect.top + 8}px`;
			dropdownElement.style.maxHeight = `${spaceAbove - 8}px`;
			dropdownElement.style.top = "auto";
		}

		dropdownElement.style.left = `${buttonRect.left}px`;
		dropdownElement.style.width = `${buttonRect.width}px`;
	}

	function handleSelectView(slug: string) {
		void onSelectView(slug);
		onClose();
	}

	function handleNewView() {
		onNewView();
		onClose();
	}

	function handleDocumentClick(event: MouseEvent) {
		if (!isOpen) return;

		const target = event.target as Node | null;
		if (buttonElement?.contains(target) || dropdownElement?.contains(target)) {
			return;
		}
		onClose();
	}

	function handleDocumentKeydown(event: KeyboardEvent) {
		if (event.key === "Escape" && isOpen) {
			onClose();
		}
	}

	function attachListeners() {
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

	function detachAllListeners() {
		if (detachListeners) {
			detachListeners();
		}
	}

	$: if (isOpen) {
		attachListeners();
	} else {
		detachAllListeners();
	}

	onDestroy(() => {
		detachAllListeners();
	});

	function isViewSelected(view: NotificationView): boolean {
		return selectedViewSlug === view.slug || selectedViewId === String(view.id);
	}

	function getViewCount(view: NotificationView): number | undefined {
		if (view.slug === "inbox" && inboxView) {
			return inboxView.unreadCount;
		}
		return view.unreadCount;
	}
</script>

{#if isOpen}
	<div
		bind:this={dropdownElement}
		class="fixed z-50 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-800 dark:bg-gray-900 overflow-y-auto"
		role="menu"
		style="min-width: 280px;"
	>
		<!-- Built-in Views Section -->
		<div class="py-2">
			<ul class="space-y-0">
				{#each builtInViews as view (view.id)}
					{@const iconType = getBuiltInIconType(view.slug)}
					{@const selected = isViewSelected(view)}
					{@const count = getViewCount(view)}
					<li>
						<button
							type="button"
							class={`group relative flex w-full items-center gap-2 rounded-md border border-transparent px-3 py-2 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 ${
								selected
									? "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100"
									: "bg-transparent text-gray-900 hover:bg-gray-50 dark:text-gray-100 dark:hover:bg-gray-800/50"
							}`}
							aria-pressed={selected}
							on:click={() => handleSelectView(view.slug)}
						>
							<span
								class={`flex items-center justify-center ${selected ? "text-gray-900 dark:text-gray-100" : "text-gray-700 dark:text-gray-300"}`}
								aria-hidden="true"
							>
								{#if iconType}
									<svg
										class="h-[18px] w-[18px]"
										viewBox={iconType === "infinity" ? "0 0 256 256" : "0 0 24 24"}
										fill="none"
										stroke={iconType === "infinity" ? "none" : "currentColor"}
										stroke-width={iconType === "infinity" ? undefined : "1.5"}
										stroke-linecap="round"
										stroke-linejoin="round"
										xmlns="http://www.w3.org/2000/svg"
										role="img"
										aria-hidden="true"
									>
										{#each builtInIconPaths[iconType] as path (path)}
											<path d={path} fill={iconType === "infinity" ? "currentColor" : undefined} />
										{/each}
									</svg>
								{/if}
							</span>
							<span class="flex flex-1 flex-col">
								<span class="flex items-center justify-between gap-2">
									<span
										class={`text-[14px] font-normal ${
											selected
												? "text-gray-900 dark:text-gray-100"
												: "text-gray-900 dark:text-gray-100"
										}`}
									>
										{view.name}
									</span>
									{#if count !== undefined && count > 0}
										<span
											class={`flex h-6 flex-shrink-0 items-center justify-center rounded-full px-2 py-1 text-xs font-regular ${
												selected
													? "bg-gray-200 text-gray-900 dark:bg-gray-700 dark:text-gray-100"
													: "bg-gray-100/60 text-gray-700 dark:bg-gray-800/60 dark:text-gray-100"
											}`}
											aria-label={`${count} new notification${count === 1 ? "" : "s"}`}
										>
											{count}
										</span>
									{/if}
								</span>
								{#if view.description}
									<span
										class={`text-xs ${
											selected
												? "text-gray-600 dark:text-gray-400"
												: "text-gray-600 dark:text-gray-400"
										}`}
									>
										{view.description}
									</span>
								{/if}
							</span>
						</button>
					</li>
				{/each}
			</ul>
		</div>

		<!-- Divider -->
		<div
			class="my-2 border-t border-gray-200 dark:border-gray-800"
			role="presentation"
			aria-hidden="true"
		></div>

		<!-- Custom Views Section -->
		<div class="py-2">
			<div class="mb-2 flex items-center justify-between px-3">
				<h2 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
					Views
				</h2>
				<button
					type="button"
					class="rounded-full px-2 py-1 text-xs font-semibold text-gray-600 transition hover:text-gray-800 dark:text-gray-500 dark:hover:text-gray-300"
					on:click={handleNewView}
				>
					Add
				</button>
			</div>
			{#if userViews.length === 0}
				<div
					class="mx-3 mb-2 flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-gray-300 bg-gray-50 py-6 text-center dark:border-gray-700 dark:bg-gray-900"
				>
					<p class="text-sm font-semibold text-gray-800 dark:text-gray-200">No views created yet</p>
					<p class="text-xs text-gray-600 dark:text-gray-400">
						Views let you quickly revisit your favorite filters.
					</p>
				</div>
			{:else}
				<ul class="space-y-0">
					{#each userViews as view (view.id)}
						{@const selected = isViewSelected(view)}
						{@const displayIcon = normalizeViewIcon(view.icon)}
						{@const count = view.unreadCount}
						{@const hasDescription =
							typeof view.description === "string" && view.description.trim().length > 0}
						<li>
							<button
								type="button"
								class={`group relative flex w-full items-center gap-2 rounded-md border border-transparent px-3 py-2 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 ${
									selected
										? "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100"
										: "bg-transparent text-gray-900 hover:bg-gray-50 dark:text-gray-100 dark:hover:bg-gray-800/50"
								}`}
								aria-pressed={selected}
								on:click={() => handleSelectView(view.slug)}
							>
								<span
									class={`flex flex-shrink-0 items-center justify-center text-base ${
										selected
											? "text-gray-900 dark:text-gray-100"
											: "text-gray-700 dark:text-gray-300"
									} ${hasDescription ? "mt-0.5" : ""}`}
									aria-hidden="true"
								>
									{displayIcon}
								</span>
								<div class="flex-1">
									<div class="flex items-center gap-1.5">
										<span
											class={`text-[14px] font-normal ${
												selected
													? "text-gray-900 dark:text-gray-100"
													: "text-gray-900 dark:text-gray-100"
											}`}
										>
											{view.name}
										</span>
										{#if count !== undefined && count > 0}
											<span
												class={`ml-auto flex h-6 items-center justify-center rounded-full px-2 text-[12px] font-regular transition ${
													selected
														? "bg-gray-200 text-gray-900 dark:bg-gray-700 dark:text-gray-100"
														: "bg-gray-100/60 text-gray-700 dark:bg-gray-800/60 dark:text-gray-100"
												}`}
												aria-label={`${count} new notification${count === 1 ? "" : "s"}`}
											>
												{count}
											</span>
										{/if}
									</div>
									{#if hasDescription}
										<p
											class={`text-xs ${
												selected
													? "text-gray-600 dark:text-gray-400"
													: "text-gray-600 dark:text-gray-400"
											}`}
										>
											{view.description}
										</p>
									{/if}
								</div>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
{/if}
