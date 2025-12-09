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

	import Modal from "../shared/Modal.svelte";
	import QuerySyntaxHelp from "../shared/QuerySyntaxHelp.svelte";
	import QueryInput from "../shared/QueryInput.svelte";
	import IconSelector from "./IconSelector.svelte";
	import RuleConfigFields from "../shared/RuleConfigFields.svelte";
	import { DEFAULT_VIEW_ICON, normalizeViewIcon } from "$lib/utils/viewIcons";
	import { parseQuery } from "$lib/utils/queryParser";
	import { fetchTags } from "$lib/api/tags";
	import type { Tag } from "$lib/api/tags";
	import { onMount } from "svelte";

	export let open = false;
	export let saving = false;
	export let error: string | null = null;
	export let onSave: (payload: {
		name: string;
		description: string;
		icon: string;
		query: string;
		createAutoRule?: boolean;
		ruleConfig?: {
			name: string;
			description?: string;
			skipInbox: boolean;
			markRead: boolean;
			star: boolean;
			archive?: boolean;
			mute?: boolean;
			assignTags?: string[];
			enabled: boolean;
			applyToExisting?: boolean;
		};
	}) => void = () => {};
	export let onClose: () => void = () => {};
	export let onClearError: (() => void) | null = null;
	export let onDelete: (() => void) | null = null;
	export let initialValue: {
		id?: string;
		name?: string;
		description?: string;
		icon?: string;
		query?: string;
	} | null = null;

	let name = "";
	let description = "";
	let query = "";
	let icon = DEFAULT_VIEW_ICON;
	let showQuerySyntaxModal = false;

	// Auto-rule creation fields
	let createAutoRule = false;
	let ruleName = "";
	let ruleDescription = "";
	let ruleSkipInbox = true;
	let ruleMarkRead = false;
	let ruleStar = false;
	let ruleArchive = false;
	let ruleMute = false;
	let ruleSelectedTags: string[] = [];
	let ruleEnabled = true;
	let ruleApplyToExisting = false;
	let availableTags: Tag[] = [];

	// Auto-update rule name based on view name
	$: if (createAutoRule && name) {
		ruleName = `${name.trim()} Auto-Filter`;
	}

	async function loadTags() {
		try {
			availableTags = await fetchTags();
		} catch (error) {}
	}

	onMount(loadTags);

	function handleQueryChange(newQuery: string) {
		query = newQuery;
	}

	function isValidIcon(iconValue: string): boolean {
		const trimmed = iconValue.trim();
		if (!trimmed) {
			return true; // Empty is valid, will use default
		}

		// For emojis: Use Intl.Segmenter if available for proper grapheme counting
		// This handles emojis with variation selectors, skin tones, etc.
		if (typeof Intl !== "undefined" && "Segmenter" in Intl) {
			const segmenter = new Intl.Segmenter("en", {
				granularity: "grapheme",
			});
			const segments = Array.from(segmenter.segment(trimmed));
			if (segments.length === 1) {
				return true;
			}
		} else {
			// Fallback: Allow short strings (likely emojis)
			// Most emojis with modifiers are under 10 code units
			if (trimmed.length <= 10) {
				return true;
			}
		}

		return false;
	}

	$: iconValid = isValidIcon(icon);

	$: queryValid = (() => {
		const trimmed = query.trim();
		if (!trimmed) return false;
		const parsed = parseQuery(trimmed);
		return parsed.isValid && parsed.errors.length === 0;
	})();

	$: isValid = name.trim().length > 0 && query.trim().length > 0 && iconValid && queryValid;

	function submit() {
		if (!name.trim() || !query.trim()) {
			return;
		}
		const preparedIcon = icon.trim() || DEFAULT_VIEW_ICON;
		const payload: any = {
			name: name.trim(),
			description: description.trim(),
			icon: preparedIcon,
			query: query.trim(),
		};

		if (createAutoRule && !initialValue?.id) {
			payload.createAutoRule = true;
			payload.ruleConfig = {
				name: ruleName.trim(),
				description: ruleDescription.trim() || undefined,
				skipInbox: ruleSkipInbox,
				markRead: ruleMarkRead,
				star: ruleStar,
				archive: ruleArchive || undefined,
				mute: ruleMute || undefined,
				assignTags: ruleSelectedTags.length > 0 ? ruleSelectedTags : undefined,
				enabled: ruleEnabled,
				applyToExisting: ruleApplyToExisting,
			};
		}

		onSave(payload);
	}

	function close() {
		onClose();
	}

	$: if (open) {
		// Refresh tags when dialog opens to pick up any newly created tags
		void loadTags();
		name = initialValue?.name?.trim() ?? (initialValue?.id ? "" : "");
		description = initialValue?.description?.trim() ?? "";
		icon = normalizeViewIcon(initialValue?.icon);
		// Pre-populate with in:inbox,filtered for new views, use existing query for editing
		query = initialValue?.query?.trim() ?? (initialValue?.id ? "" : "in:inbox,filtered");
		// Reset auto-rule fields when opening dialog
		createAutoRule = false;
		ruleName = "";
		ruleDescription = "";
		ruleSkipInbox = true;
		ruleMarkRead = false;
		ruleStar = false;
		ruleArchive = false;
		ruleMute = false;
		ruleSelectedTags = [];
		ruleEnabled = true;
		ruleApplyToExisting = false;
	} else {
		name = "";
		description = "";
		icon = DEFAULT_VIEW_ICON;
		query = "in:inbox,filtered";
		createAutoRule = false;
	}
</script>

<Modal
	{open}
	title={initialValue?.id ? "Edit view" : "Add new view"}
	size="md"
	maxHeight="90vh"
	onClose={close}
>
	<form class="space-y-5" on:submit|preventDefault={submit}>
		<div class="rounded-xl bg-gray-50 dark:bg-gray-900/80 p-5 space-y-5">
			<div class="space-y-2">
				<div>
					<label class="text-sm font-medium text-gray-900 dark:text-gray-200" for="filter-name"
						>Name</label
					>
					<p class="text-xs text-gray-600 dark:text-gray-500 mt-1">
						Choose an emoji and a unique name for this view
					</p>
				</div>
				<div class="flex gap-3">
					<IconSelector value={icon} onChange={(newIcon) => (icon = newIcon)} isValid={iconValid} />
					<input
						id="filter-name"
						class="flex-1 rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-gray-200 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30"
						class:!border-rose-500={error}
						placeholder="My open PRs"
						bind:value={name}
						on:input={() => onClearError?.()}
						required
					/>
				</div>
				{#if !iconValid}
					<p class="text-xs text-rose-400">Icon must be a single emoji</p>
				{/if}
				{#if error}
					<p class="text-xs text-rose-400">{error}</p>
				{/if}
			</div>

			<div class="space-y-2">
				<div>
					<label
						class="text-sm font-medium text-gray-900 dark:text-gray-200"
						for="filter-description">Description</label
					>
					<p class="text-xs text-gray-600 dark:text-gray-500 mt-1">
						Optional extra details to show in the view list
					</p>
				</div>
				<input
					id="filter-description"
					class="w-full rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-gray-200 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30"
					bind:value={description}
					placeholder="Shows all open pull requests assigned to me"
					maxlength="50"
				/>
			</div>

			<div class="space-y-2">
				<div class="flex items-center justify-between gap-3">
					<div>
						<label class="text-sm font-medium text-gray-900 dark:text-gray-200" for="view-query"
							>Query</label
						>
					</div>
					<button
						type="button"
						class="text-xs text-blue-500 transition hover:text-blue-400 flex-shrink-0 cursor-pointer"
						on:click={() => (showQuerySyntaxModal = true)}
					>
						Query syntax
					</button>
				</div>
				<p class="text-xs text-gray-600 dark:text-gray-500 mb-2">
					The query defines what notifications are shown in the view
				</p>
				<QueryInput
					bind:value={query}
					onChange={handleQueryChange}
					placeholder="e.g., repo:owner/name type:issue urgent fix -author:dependabot"
					inputId="view-query"
					showValidation={true}
					showDropdownOnFocus={true}
				/>

				<p class="text-[10px] text-light text-gray-600 dark:text-gray-400 mt-1 pl-1">
					The default query includes <code class="text-gray-800 dark:text-gray-300"
						>in:inbox,filtered</code
					> to show both inbox and filtered notifications (includes notifications that skipped the inbox
					due to rule automation).
				</p>
			</div>

			<!-- Auto-rule creation section - only for new views -->
			{#if !initialValue?.id}
				<div class="space-y-3 pt-2 border-t border-gray-200 dark:border-gray-800">
					<label class="flex items-center gap-3 cursor-pointer">
						<input
							type="checkbox"
							bind:checked={createAutoRule}
							class="w-4 h-4 rounded border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 text-indigo-600 focus:ring-2 focus:ring-indigo-600/30 focus:ring-offset-0"
						/>
						<div>
							<div class="text-sm font-medium text-gray-900 dark:text-gray-200">
								Create auto-filtering rule for this view
							</div>
							<p class="text-xs text-gray-600 dark:text-gray-500 mt-0.5">
								Automatically process notifications that match this view's query
							</p>
						</div>
					</label>

					{#if createAutoRule}
						<div class="ml-7 pl-4 border-l-2 border-indigo-600/30">
							<RuleConfigFields
								bind:name={ruleName}
								bind:description={ruleDescription}
								bind:skipInbox={ruleSkipInbox}
								bind:markRead={ruleMarkRead}
								bind:star={ruleStar}
								bind:archive={ruleArchive}
								bind:mute={ruleMute}
								bind:selectedTags={ruleSelectedTags}
								bind:enabled={ruleEnabled}
								bind:applyToExisting={ruleApplyToExisting}
								{availableTags}
								showApplyToExisting={true}
								inline={true}
							/>
						</div>
					{/if}
				</div>
			{/if}
		</div>

		<div class="flex items-center justify-between gap-3 pt-2">
			{#if initialValue?.id && onDelete}
				<button
					type="button"
					class="rounded-full border border-gray-200 dark:border-gray-800 px-4 py-2 text-sm font-medium text-rose-600 dark:text-rose-300 transition hover:border-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/40 hover:text-rose-700 dark:hover:text-rose-200 cursor-pointer"
					on:click={() => onDelete?.()}
					disabled={saving}
				>
					Delete view
				</button>
			{:else}
				<span></span>
			{/if}
			<div class="flex items-center gap-3">
				<button
					type="button"
					class="rounded-full border border-transparent px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 transition hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
					on:click={close}
					disabled={saving}
				>
					Cancel
				</button>
				<button
					type="submit"
					class="rounded-full bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-indigo-700 disabled:opacity-50 disabled:hover:bg-indigo-600 cursor-pointer"
					disabled={saving || !isValid}
				>
					{saving ? "Saving…" : "Save view"}
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
