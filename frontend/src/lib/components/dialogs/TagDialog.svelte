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

	import { createEventDispatcher } from "svelte";
	import type { Tag, TagInput } from "$lib/api/tags";
	import { createTag, updateTag } from "$lib/api/tags";
	import { toastStore } from "$lib/stores/toastStore";
	import Modal from "$lib/components/shared/Modal.svelte";
	import { invalidateAll } from "$app/navigation";

	export let open = false;
	export let tag: Tag | null = null; // null for create, tag object for edit
	export let onDelete: (() => void) | null = null;

	const dispatch = createEventDispatcher();

	// Preset color palette (Tailwind-inspired, tag-friendly colors)
	const colorPalette = [
		"#ef4444", // red-500
		"#f59e0b", // amber-500
		"#eab308", // yellow-500
		"#22c55e", // green-500
		"#10b981", // emerald-500
		"#14b8a6", // teal-500
		"#06b6d4", // cyan-500
		"#3b82f6", // blue-500
		"#6366f1", // indigo-500
		"#8b5cf6", // violet-500
		"#d946ef", // fuchsia-500
		"#ec4899", // pink-500
	];

	let name = "";
	let color = "";
	let description = "";
	let saving = false;

	$: isEditMode = tag !== null;
	$: isValid = name.trim();

	$: if (open) {
		if (tag) {
			// Edit mode - populate form
			name = tag.name;
			color = tag.color || "";
			description = tag.description || "";
		} else {
			// Create mode - reset form
			name = "";
			color = colorPalette[8]; // Default to indigo
			description = "";
		}
	}

	function selectColor(selectedColor: string) {
		color = selectedColor;
	}

	function handleClose() {
		open = false;
		dispatch("close");
	}

	async function handleSave() {
		if (!isValid || saving) return;

		saving = true;
		try {
			const input: TagInput = {
				name: name.trim(),
				color: color.trim() || undefined,
				description: description.trim() || undefined,
			};

			if (isEditMode && tag) {
				await updateTag(tag.id, input);
				toastStore.show("Tag updated successfully", "success");
			} else {
				await createTag(input);
				toastStore.show("Tag created successfully", "success");
			}

			await invalidateAll();
			handleClose();
		} catch (error) {
			toastStore.show(`Failed to save tag: ${error}`, "error");
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!isEditMode || !tag) return;
		onDelete?.();
	}

	function handleSubmit() {
		void handleSave();
	}
</script>

<Modal {open} title={isEditMode ? "Edit Tag" : "New Tag"} size="sm" onClose={handleClose}>
	<form class="space-y-4" on:submit|preventDefault={handleSubmit}>
		<div class="rounded-xl bg-gray-50 dark:bg-gray-900/80 p-5 space-y-4">
			<!-- Name -->
			<div class="space-y-2">
				<label for="tag-name" class="text-sm font-medium text-gray-900 dark:text-gray-200">
					Name <span class="text-red-400">*</span>
				</label>
				<!-- svelte-ignore a11y_autofocus -->
				<input
					id="tag-name"
					bind:value={name}
					type="text"
					placeholder="e.g., urgent, bug, feature"
					class="w-full rounded-lg border border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-gray-200 placeholder-gray-500 dark:placeholder-gray-500 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30"
					required
					autofocus
				/>
			</div>

			<!-- Color -->
			<div class="space-y-3">
				<label for="tag-color" class="text-sm font-medium text-gray-900 dark:text-gray-200"
					>Color</label
				>

				<!-- Color Indicator and Hex Input -->
				<div class="flex gap-2 items-center">
					<div
						class="w-10 h-10 rounded-full border-2 border-gray-300 dark:border-gray-700 flex-shrink-0"
						style="background-color: {color || '#6366f1'}"
					></div>
					<input
						id="tag-color"
						bind:value={color}
						type="text"
						placeholder="#6366f1"
						class="flex-1 rounded-lg border border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-gray-200 placeholder-gray-500 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30 font-mono"
					/>
				</div>

				<!-- Color Palette Grid -->
				<div class="grid grid-cols-6 gap-2">
					{#each colorPalette as paletteColor (paletteColor)}
						<button
							type="button"
							on:click={() => selectColor(paletteColor)}
							class="relative h-9 rounded-lg border-2 transition-all hover:scale-105 cursor-pointer {color.toLowerCase() ===
							paletteColor.toLowerCase()
								? 'border-white'
								: 'border-transparent hover:border-gray-400 dark:hover:border-gray-500'}"
							style="background-color: {paletteColor}"
							title={paletteColor}
						></button>
					{/each}
				</div>
			</div>

			<!-- Description -->
			<div class="space-y-2">
				<label for="tag-description" class="text-sm font-medium text-gray-900 dark:text-gray-200">
					Description
				</label>
				<textarea
					id="tag-description"
					bind:value={description}
					rows="2"
					placeholder="Optional description for this tag"
					class="w-full rounded-lg border border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-gray-200 placeholder-gray-500 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30 resize-none"
				></textarea>
			</div>
		</div>

		<!-- Footer actions -->
		<div class="flex items-center justify-between gap-3 pt-2">
			<div>
				{#if isEditMode && onDelete}
					<button
						type="button"
						on:click={handleDelete}
						disabled={saving}
						class="px-4 py-2 text-sm font-medium text-red-400 hover:text-red-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
					>
						Delete
					</button>
				{/if}
			</div>
			<div class="flex gap-2">
				<button
					type="button"
					on:click={handleClose}
					disabled={saving}
					class="px-4 py-2 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
				>
					Cancel
				</button>
				<button
					type="submit"
					disabled={!isValid || saving}
					class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
				>
					{saving ? "Saving..." : isEditMode ? "Save" : "Create"}
				</button>
			</div>
		</div>
	</form>
</Modal>
