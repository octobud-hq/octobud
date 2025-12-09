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

	import { createEventDispatcher, onMount } from "svelte";
	import type { Rule, RuleActions } from "$lib/api/rules";
	import type { Tag } from "$lib/api/tags";
	import type { NotificationView } from "$lib/api/types";
	import { createRule, updateRule } from "$lib/api/rules";
	import { fetchTags } from "$lib/api/tags";
	import { fetchViews } from "$lib/api/views";
	import { toastStore } from "$lib/stores/toastStore";
	import RuleConfigFields from "$lib/components/shared/RuleConfigFields.svelte";
	import QueryInput from "$lib/components/shared/QueryInput.svelte";
	import QuerySyntaxHelp from "$lib/components/shared/QuerySyntaxHelp.svelte";
	import Modal from "$lib/components/shared/Modal.svelte";
	import { invalidateAll } from "$app/navigation";
	import { parseQuery } from "$lib/utils/queryParser";

	export let open = false;
	export let rule: Rule | null = null; // null for create, rule object for edit
	export let onDelete: (() => void) | null = null;

	const dispatch = createEventDispatcher();

	let name = "";
	let description = "";
	let ruleMode: "query" | "view" = "query";
	let query = "";
	let selectedViewId: string = "";
	let skipInbox = true;
	let markRead = false;
	let star = false;
	let archive = false;
	let mute = false;
	let selectedTags: string[] = [];
	let enabled = true;
	let applyToExisting = false;
	let availableTags: Tag[] = [];
	let availableViews: NotificationView[] = [];
	let saving = false;
	let viewDropdownOpen = false;
	let showQuerySyntaxModal = false;

	$: isEditMode = rule !== null;
	$: queryValid = (() => {
		if (ruleMode === "view") return true;
		const trimmed = query.trim();
		if (!trimmed) return false;
		const parsed = parseQuery(trimmed);
		return parsed.isValid && parsed.errors.length === 0;
	})();
	$: isValid = name.trim() && (ruleMode === "query" ? query.trim() && queryValid : selectedViewId);
	$: selectedView = availableViews.find((v) => v.id === selectedViewId);

	async function loadTags() {
		try {
			availableTags = await fetchTags();
		} catch (error) {}
	}

	onMount(loadTags);

	async function loadViews() {
		try {
			const allViews = await fetchViews();
			// Filter to only custom views (exclude system views)
			availableViews = allViews.filter((v) => !v.systemView);
		} catch (error) {}
	}

	$: if (open) {
		// Load views and tags when dialog opens to pick up any newly created items
		void loadViews();
		void loadTags();

		if (rule) {
			// Edit mode - populate form
			name = rule.name;
			description = rule.description || "";
			if (rule.viewId) {
				ruleMode = "view";
				selectedViewId = rule.viewId;
				query = "";
			} else {
				ruleMode = "query";
				query = rule.query;
				selectedViewId = "";
			}
			skipInbox = rule.actions.skipInbox;
			markRead = rule.actions.markRead || false;
			star = rule.actions.star || false;
			archive = rule.actions.archive || false;
			mute = rule.actions.mute || false;
			// selectedTags is already tag IDs from the API
			selectedTags = rule.actions.assignTags || [];
			enabled = rule.enabled;
			applyToExisting = false; // Only for create
		} else {
			// Create mode - reset form
			name = "";
			description = "";
			ruleMode = "query";
			query = "";
			selectedViewId = "";
			skipInbox = true;
			markRead = false;
			star = false;
			archive = false;
			mute = false;
			selectedTags = [];
			enabled = true;
			applyToExisting = false;
		}
		viewDropdownOpen = false; // Close dropdown when dialog opens/closes
	}

	function handleClose() {
		open = false;
		viewDropdownOpen = false;
		dispatch("close");
	}

	async function handleSave() {
		if (!isValid || saving) return;

		saving = true;
		try {
			const actions: RuleActions = {
				skipInbox,
				markRead: markRead || undefined,
				star: star || undefined,
				archive: archive || undefined,
				mute: mute || undefined,
				assignTags: selectedTags.length > 0 ? selectedTags : undefined,
			};

			const payload: any = {
				name: name.trim(),
				description: description.trim() || undefined,
				actions,
				enabled,
			};

			if (ruleMode === "view") {
				payload.viewId = selectedViewId;
				// Don't include query in payload - let the view provide it
			} else {
				payload.query = query.trim();
			}

			if (isEditMode && rule) {
				await updateRule(rule.id, payload);
				toastStore.show("Rule updated successfully", "success");
			} else {
				payload.applyToExisting = applyToExisting;
				await createRule(payload);
				toastStore.show("Rule created successfully", "success");
			}

			await invalidateAll();
			handleClose();
		} catch (error) {
			toastStore.show(`Failed to save rule: ${error}`, "error");
		} finally {
			saving = false;
		}
	}

	function handleQueryChange(newQuery: string) {
		query = newQuery;
	}

	function selectView(viewId: string) {
		selectedViewId = viewId;
		viewDropdownOpen = false;
	}

	function toggleViewDropdown() {
		viewDropdownOpen = !viewDropdownOpen;
	}

	function closeViewDropdown() {
		viewDropdownOpen = false;
	}

	function handleSubmit() {
		void handleSave();
	}
</script>

<Modal
	{open}
	title={isEditMode ? "Edit Rule" : "New Rule"}
	size="md"
	maxHeight="calc(100vh - 12rem)"
	onClose={handleClose}
>
	<form class="space-y-5" on:submit|preventDefault={handleSubmit}>
		<div class="rounded-xl bg-gray-50 dark:bg-gray-900/80 p-5 space-y-5">
			<!-- Rule Mode Selector -->
			<div class="space-y-2">
				<div class="text-sm font-medium text-gray-900 dark:text-gray-200">
					Rule Type <span class="text-red-400">*</span>
				</div>
				<div class="flex gap-4">
					<label class="flex items-center cursor-pointer">
						<input
							type="radio"
							bind:group={ruleMode}
							value="query"
							class="w-4 h-4 text-blue-600 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 focus:ring-blue-600 focus:ring-offset-white dark:focus:ring-offset-gray-900"
						/>
						<span class="ml-2 text-sm text-gray-700 dark:text-gray-300">Query-based</span>
					</label>
					<label class="flex items-center cursor-pointer">
						<input
							type="radio"
							bind:group={ruleMode}
							value="view"
							class="w-4 h-4 text-blue-600 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 focus:ring-blue-600 focus:ring-offset-white dark:focus:ring-offset-gray-900"
						/>
						<span class="ml-2 text-sm text-gray-700 dark:text-gray-300">View-linked</span>
					</label>
				</div>
				<p class="text-xs text-gray-600 dark:text-gray-500">
					{#if ruleMode === "query"}
						Use a custom query to match notifications
					{:else}
						Link to a view's query (automatically updates when the view changes)
					{/if}
				</p>
			</div>

			<!-- Query or View Selection -->
			{#if ruleMode === "query"}
				<div class="space-y-2">
					<div class="flex items-center justify-between gap-3">
						<div>
							<label for="rule-query" class="text-sm font-medium text-gray-900 dark:text-gray-200">
								Query <span class="text-red-400">*</span>
							</label>
						</div>
						<button
							type="button"
							class="text-xs text-blue-500 transition hover:text-blue-400 flex-shrink-0 cursor-pointer"
							on:click={() => (showQuerySyntaxModal = true)}
						>
							Query syntax
						</button>
					</div>
					<p class="text-xs text-gray-500 mb-2">
						Use the same query syntax as custom views to filter notifications
					</p>
					<QueryInput
						bind:value={query}
						onChange={handleQueryChange}
						placeholder="e.g., repo:owner/name type:issue urgent fix -author:dependabot"
						inputId="rule-query"
						showValidation={true}
						showDropdownOnFocus={true}
					/>
				</div>
			{:else}
				<div class="relative space-y-2">
					<label for="rule-view" class="text-sm font-medium text-gray-900 dark:text-gray-200">
						Linked View <span class="text-red-400">*</span>
					</label>
					<p class="text-xs text-gray-600 dark:text-gray-500">
						{#if selectedView}
							The rule will automatically use this view's current query
						{:else}
							Select which view's query this rule should follow
						{/if}
					</p>

					<!-- Custom dropdown button -->
					<button
						type="button"
						on:click={toggleViewDropdown}
						class="w-full rounded-lg border border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-left outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30 flex items-center justify-between cursor-pointer"
					>
						{#if selectedView}
							<span class="flex items-center gap-2 text-gray-900 dark:text-gray-200">
								<span class="text-lg">{selectedView.icon || "📋"}</span>
								<span>{selectedView.name}</span>
							</span>
						{:else}
							<span class="text-gray-600 dark:text-gray-500">Select a view...</span>
						{/if}
						<svg
							class="w-4 h-4 text-gray-600 dark:text-gray-400 transition-transform {viewDropdownOpen
								? 'rotate-180'
								: ''}"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M19 9l-7 7-7-7"
							/>
						</svg>
					</button>

					<!-- Dropdown menu -->
					{#if viewDropdownOpen}
						<div
							class="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg max-h-60 overflow-y-auto"
							role="listbox"
						>
							{#if availableViews.length === 0}
								<div class="px-3 py-2 text-sm text-gray-600 dark:text-gray-500 text-center">
									No custom views available
								</div>
							{:else}
								{#each availableViews as view (view.id)}
									<button
										type="button"
										on:click={() => selectView(view.id)}
										class="w-full px-3 py-2 text-left hover:bg-gray-700 transition-colors flex items-center gap-2 cursor-pointer {selectedViewId ===
										view.id
											? 'bg-gray-700'
											: ''}"
									>
										<span class="text-lg">{view.icon || "📋"}</span>
										<div class="flex-1 min-w-0">
											<div class="text-sm text-gray-900 dark:text-gray-200 truncate">
												{view.name}
											</div>
											{#if view.description}
												<div class="text-xs text-gray-600 dark:text-gray-500 truncate">
													{view.description}
												</div>
											{/if}
										</div>
										{#if selectedViewId === view.id}
											<svg
												class="w-4 h-4 text-blue-400 flex-shrink-0"
												fill="currentColor"
												viewBox="0 0 20 20"
											>
												<path
													fill-rule="evenodd"
													d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
													clip-rule="evenodd"
												/>
											</svg>
										{/if}
									</button>
								{/each}
							{/if}
						</div>
					{/if}

					{#if selectedView}
						<p class="text-xs text-gray-600 dark:text-gray-500">
							Current query: <span class="font-mono">{selectedView.query || "(empty)"}</span>
						</p>
					{/if}
				</div>
			{/if}

			<!-- Rule Configuration Fields (shared component) -->
			<RuleConfigFields
				bind:name
				bind:description
				bind:skipInbox
				bind:markRead
				bind:star
				bind:archive
				bind:mute
				bind:selectedTags
				bind:enabled
				bind:applyToExisting
				{availableTags}
				showApplyToExisting={!isEditMode}
				inline={false}
			/>
		</div>

		<div class="flex items-center justify-between gap-3 pt-2">
			{#if isEditMode && onDelete}
				<button
					type="button"
					class="rounded-full border border-gray-300 dark:border-gray-800 px-4 py-2 text-sm font-medium text-rose-600 dark:text-rose-300 transition hover:border-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/40 hover:text-rose-700 dark:hover:text-rose-200 cursor-pointer"
					on:click={onDelete}
					disabled={saving}
				>
					Delete rule
				</button>
			{:else}
				<span></span>
			{/if}
			<div class="flex items-center gap-3">
				<button
					type="button"
					on:click={handleClose}
					disabled={saving}
					class="rounded-full border border-transparent px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 transition hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
				>
					Cancel
				</button>
				<button
					type="submit"
					disabled={!isValid || saving}
					class="rounded-full bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-indigo-700 disabled:opacity-50 disabled:hover:bg-indigo-600 cursor-pointer"
				>
					{#if saving}
						<span class="flex items-center gap-2">
							<svg
								class="animate-spin w-4 h-4"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								></path>
							</svg>
							Saving…
						</span>
					{:else}
						{isEditMode ? "Save changes" : "Create rule"}
					{/if}
				</button>
			</div>
		</div>
	</form>
</Modal>

<Modal
	open={showQuerySyntaxModal}
	title="Query Syntax Guide"
	size="lg"
	onClose={() => (showQuerySyntaxModal = false)}
	maxHeight="calc(100vh - 8rem)"
>
	<QuerySyntaxHelp />
</Modal>
