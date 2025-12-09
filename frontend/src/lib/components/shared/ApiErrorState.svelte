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

	export let errorMessage: string | null = null;
</script>

<div class="flex flex-1 items-center justify-center">
	<div
		class="mx-auto max-w-2xl rounded-2xl border border-rose-300 dark:border-rose-800/50 bg-white dark:bg-gray-900 p-8 shadow-xl"
	>
		<div class="flex flex-col items-center gap-6 text-center">
			<!-- Error Icon -->
			<div
				class="flex h-16 w-16 items-center justify-center rounded-full bg-rose-500/10 ring-2 ring-rose-500/30"
			>
				<svg
					class="h-8 w-8 text-rose-500 dark:text-rose-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
					/>
				</svg>
			</div>

			<!-- Error Message -->
			<div class="space-y-2">
				<h2 class="text-2xl font-bold text-rose-600 dark:text-rose-100">
					Unable to Connect to API
				</h2>
				{#if errorMessage}
					<p class="text-gray-700 dark:text-gray-200">
						{errorMessage}
					</p>
				{:else}
					<p class="text-gray-600 dark:text-gray-300">
						The frontend cannot reach the backend server. Please verify that all services are
						running:
					</p>
				{/if}
			</div>

			<!-- Requirements List -->
			<div class="w-full space-y-2 rounded-lg bg-gray-50 dark:bg-gray-800/50 p-4 text-left">
				<p
					class="mb-3 text-sm font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-400"
				>
					Required Services
				</p>
				<ul class="space-y-2 text-sm text-gray-700 dark:text-gray-300">
					<li class="flex items-start gap-2">
						<span class="mt-0.5 text-rose-500 dark:text-rose-400">•</span>
						<span
							><strong class="text-gray-900 dark:text-gray-200">API Server:</strong>
							Make sure the backend server is running</span
						>
					</li>
					<li class="flex items-start gap-2">
						<span class="mt-0.5 text-rose-500 dark:text-rose-400">•</span>
						<span
							><strong class="text-gray-900 dark:text-gray-200">Worker:</strong> Ensure the background
							worker is active</span
						>
					</li>
					<li class="flex items-start gap-2">
						<span class="mt-0.5 text-rose-500 dark:text-rose-400">•</span>
						<span
							><strong class="text-gray-900 dark:text-gray-200">Database:</strong> Verify the database
							is accessible</span
						>
					</li>
				</ul>
			</div>

			<!-- Documentation Link -->
			<a
				href="https://github.com/octobud-hq/octobud#quick-start"
				target="_blank"
				rel="noopener noreferrer"
				class="inline-flex items-center gap-2 rounded-lg bg-gray-600 dark:bg-gray-500 px-4 py-2 text-sm font-semibold text-white dark:text-gray-200 transition hover:bg-gray-700 dark:hover:bg-gray-400"
			>
				<span>View Setup Documentation</span>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
					/>
				</svg>
			</a>
		</div>
	</div>
</div>
