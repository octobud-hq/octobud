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

	import NotificationToolbar from "./NotificationToolbar.svelte";
	import type { Tag } from "$lib/api/types";
	import type { NotificationToolbarComponent } from "$lib/types/components";

	// Toolbar props
	export let combinedQuery: string;
	export let canUpdateView: boolean;
	export let onChangeQuery: (value: string) => void;
	export let onSubmitQuery: (value: string) => void;
	export let onClear: () => void;
	export let onUpdateView: () => void;
	export let onSaveAsNewView: () => void;
	export let page: number;
	export let totalPages: number;
	export let onPrevious: () => void;
	export let onNext: () => void;
	export let multiselectMode: boolean;
	export let onToggleMultiselect: () => void;
	export let splitModeEnabled: boolean;
	export let onToggleSplitMode: () => void;
	export let tags: Tag[] = [];

	// Export toolbar ref for parent access
	let toolbarComponent: NotificationToolbarComponent | null = null;
	export function focusSearchInput() {
		return toolbarComponent?.focusSearchInput() ?? false;
	}

	// Bulk selection bar methods - proxied through toolbar
	export function openBulkTagDropdown() {
		return toolbarComponent?.openBulkTagDropdown() ?? false;
	}
	export function closeBulkTagDropdown() {
		return toolbarComponent?.closeBulkTagDropdown() ?? false;
	}
	export function getBulkTagDropdownOpen(): boolean {
		return toolbarComponent?.getBulkTagDropdownOpen() ?? false;
	}
	export function openBulkSnoozeDropdown() {
		return toolbarComponent?.openBulkSnoozeDropdown() ?? false;
	}
	export function closeBulkSnoozeDropdown() {
		return toolbarComponent?.closeBulkSnoozeDropdown() ?? false;
	}
	export function getBulkSnoozeDropdownOpen(): boolean {
		return toolbarComponent?.getBulkSnoozeDropdownOpen() ?? false;
	}
</script>

<div class="pl-4 pr-2 pb-5 -mb-5 bg-transparent dark:bg-transparent">
	<NotificationToolbar
		bind:this={toolbarComponent}
		{combinedQuery}
		{canUpdateView}
		{onChangeQuery}
		{onSubmitQuery}
		{onClear}
		{onUpdateView}
		{onSaveAsNewView}
		{page}
		{totalPages}
		{onPrevious}
		{onNext}
		{multiselectMode}
		{onToggleMultiselect}
		{splitModeEnabled}
		{onToggleSplitMode}
		{tags}
	/>
</div>
