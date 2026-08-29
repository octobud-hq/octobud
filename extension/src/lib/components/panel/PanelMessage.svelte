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

	// The desktop app's ApiErrorState is a max-w-2xl, p-8, text-2xl card that lists
	// "API Server / Worker / Database" as things to check. None of that fits a
	// ~360px panel, and none of it is the panel's actual failure mode — which is
	// almost always "the Octobud app isn't running". So this is a panel-native
	// component using the same tokens rather than a copy of that one.

	export let tone: "error" | "empty" = "empty";
	export let title: string;
	export let body: string = "";
	export let actionLabel: string = "";
	export let onAction: (() => void) | null = null;

	$: isError = tone === "error";
</script>

<div class="flex flex-1 items-center justify-center px-4 py-10">
	<div class="flex max-w-[16rem] flex-col items-center gap-3 text-center">
		<div
			class="flex h-11 w-11 items-center justify-center rounded-full {isError
				? 'bg-rose-500/10 ring-1 ring-rose-500/30'
				: 'bg-gray-500/10 ring-1 ring-gray-500/20'}"
		>
			{#if isError}
				<svg
					class="h-5 w-5 text-rose-500 dark:text-rose-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
					/>
				</svg>
			{:else}
				<svg
					class="h-5 w-5 text-gray-500 dark:text-gray-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
				</svg>
			{/if}
		</div>

		<div class="space-y-1">
			<p
				class="text-sm font-semibold {isError
					? 'text-rose-600 dark:text-rose-300'
					: 'text-gray-800 dark:text-gray-200'}"
			>
				{title}
			</p>
			{#if body}
				<p class="text-xs text-gray-600 dark:text-gray-400">{body}</p>
			{/if}
		</div>

		{#if actionLabel && onAction}
			<button
				type="button"
				class="rounded-full bg-indigo-600 px-3.5 py-1.5 text-xs font-semibold text-white transition hover:bg-indigo-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 dark:bg-indigo-500 dark:hover:bg-indigo-400"
				on:click={onAction}
			>
				{actionLabel}
			</button>
		{/if}
	</div>
</div>
