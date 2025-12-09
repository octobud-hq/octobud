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
</script>

<div class="space-y-6 text-sm">
	<!-- Basics Section -->
	<section class="space-y-3">
		<h4 class="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-400">
			Query Basics
		</h4>
		<div class="space-y-2">
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">field:value</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Basic filter syntax. Match notifications where the field contains the value (not exact
					match).
				</p>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-500">
					Example: <code class="text-indigo-600 dark:text-indigo-300">repo:traefik</code>
					matches
					<code class="text-indigo-600 dark:text-indigo-300">traefik/traefik</code>
					or
					<code class="text-indigo-600 dark:text-indigo-300">traefik/other</code>
				</p>
			</div>

			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">in:inbox</code>
				or
				<code class="text-indigo-600 dark:text-indigo-300">in:archive</code>
				or
				<code class="text-indigo-600 dark:text-indigo-300">in:snoozed</code>
				or
				<code class="text-indigo-600 dark:text-indigo-300">in:filtered</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Control which notifications to show. By default, queries search the inbox (excludes
					archived, snoozed, filtered, and muted). Use <code class="text-indigo-300"
						>in:filtered</code
					>
					to include notifications that skipped the inbox due to rule automation. Use
					<code class="text-indigo-300">in:anywhere</code> to search all notifications.
				</p>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-500">
					Example: <code class="text-indigo-600 dark:text-indigo-300">in:archive repo:cli/cli</code>
				</p>
			</div>

			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">repo:cli type:issue</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Combine multiple filters with spaces (implicit AND). All conditions must match.
				</p>
			</div>

			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">urgent fix</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Free text without colons searches across title, repo, author, and type (contains
					matching).
				</p>
			</div>
		</div>
	</section>

	<!-- Logical Operators Section -->
	<section class="space-y-3">
		<h4 class="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-400">
			Logical Operators
		</h4>
		<div class="space-y-2">
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">-field:value</code>
				or
				<code class="text-indigo-600 dark:text-indigo-300">NOT field:value</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Exclude matches. The dash (<code>-</code>) is a shortcut for
					<code>NOT</code>.
				</p>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-500">
					Example: <code class="text-indigo-600 dark:text-indigo-300">-author:dependabot</code>
				</p>
			</div>

			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">field:value1,value2</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Match any of the comma-separated values (implicit OR).
				</p>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-500">
					Example: <code class="text-indigo-600 dark:text-indigo-300"
						>reason:mention,review_requested</code
					>
				</p>
			</div>

			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">field1:value1 AND field2:value2</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Explicit AND operator (space between filters also means AND).
				</p>
			</div>

			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">field1:value1 OR field2:value2</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Match if either condition is true.
				</p>
			</div>

			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300"
					>(repo:cli/cli OR repo:other) type:issue</code
				>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Use parentheses to group conditions and control precedence.
				</p>
			</div>
		</div>
	</section>

	<!-- Available Fields Table -->
	<section class="space-y-3">
		<h4 class="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-400">
			All Supported Filters
		</h4>
		<div class="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
			<table class="w-full text-xs">
				<thead class="bg-gray-100/70 dark:bg-gray-800/70">
					<tr>
						<th class="px-3 py-2 text-left font-semibold text-gray-900 dark:text-gray-300"
							>Filter</th
						>
						<th class="px-3 py-2 text-left font-semibold text-gray-900 dark:text-gray-300"
							>Description</th
						>
						<th class="px-3 py-2 text-left font-semibold text-gray-900 dark:text-gray-300"
							>Values / Examples</th
						>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-200 dark:divide-gray-700">
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"><code class="text-indigo-600 dark:text-indigo-300">in:</code></td>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400">View context</td>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">inbox</code>,
							<code class="text-indigo-600 dark:text-indigo-300">archive</code>,
							<code class="text-indigo-600 dark:text-indigo-300">snoozed</code>,
							<code class="text-indigo-600 dark:text-indigo-300">filtered</code>,
							<code class="text-indigo-600 dark:text-indigo-300">anywhere</code>
						</td>
					</tr>
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"><code class="text-indigo-600 dark:text-indigo-300">is:</code></td>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400">Special status</td>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">read</code>,
							<code class="text-indigo-600 dark:text-indigo-300">unread</code>,
							<code class="text-indigo-600 dark:text-indigo-300">muted</code>,
							<code class="text-indigo-600 dark:text-indigo-300">filtered</code>
						</td>
					</tr>
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"
							><code class="text-indigo-600 dark:text-indigo-300">reason:</code></td
						>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400">Why you got it</td>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">mention</code>,
							<code class="text-indigo-600 dark:text-indigo-300">review_requested</code>,
							<code class="text-indigo-600 dark:text-indigo-300">assign</code>
						</td>
					</tr>
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"
							><code class="text-indigo-600 dark:text-indigo-300">type:</code></td
						>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400">Subject type</td>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">issue</code>,
							<code class="text-indigo-600 dark:text-indigo-300">pullrequest</code>,
							<code class="text-indigo-600 dark:text-indigo-300">release</code>
						</td>
					</tr>
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"
							><code class="text-indigo-600 dark:text-indigo-300">repo:</code></td
						>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400">Repository (contains match)</td>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">traefik</code>
							matches
							<code class="text-indigo-600 dark:text-indigo-300">traefik/traefik</code>
						</td>
					</tr>
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"
							><code class="text-indigo-600 dark:text-indigo-300">state:</code></td
						>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400">Issue/PR state</td>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">open</code>,
							<code class="text-indigo-600 dark:text-indigo-300">closed</code>
						</td>
					</tr>
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"
							><code class="text-indigo-600 dark:text-indigo-300">author:</code></td
						>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400"
							>Notification author (contains match)</td
						>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">octocat</code>
						</td>
					</tr>
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"><code class="text-indigo-600 dark:text-indigo-300">org:</code></td
						>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400">Organization (contains match)</td
						>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">github</code>
						</td>
					</tr>
					<tr class="bg-gray-50/30 dark:bg-gray-800/30">
						<td class="px-3 py-2"
							><code class="text-indigo-600 dark:text-indigo-300">tags:</code></td
						>
						<td class="px-3 py-2 text-gray-600 dark:text-gray-400">Tag slug (partial match)</td>
						<td class="px-3 py-2 text-gray-700 dark:text-gray-500">
							<code class="text-indigo-600 dark:text-indigo-300">urgent</code>,
							<code class="text-indigo-600 dark:text-indigo-300">bug-fix</code>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
		<div
			class="rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800/50 px-3 py-2"
		>
			<p class="text-xs text-blue-700 dark:text-blue-300">
				<span class="font-semibold">Shortcuts:</span>
				<code class="text-blue-600 dark:text-blue-200">-</code> for NOT • space for AND •
				<code class="text-blue-600 dark:text-blue-200">,</code> for OR (within a field)
			</p>
		</div>
	</section>

	<!-- Examples Section -->
	<section class="space-y-3">
		<h4 class="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-400">
			Query Examples
		</h4>
		<div class="space-y-2">
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">repo:traefik type:issue state:open</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Open issues in repositories containing "traefik" (e.g., traefik/traefik, traefik/other)
				</p>
			</div>
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300"
					>type:PullRequest state:open is:unread</code
				>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">Open, unread pull requests</p>
			</div>
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">in:archive org:github</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Archived notifications from any GitHub org repository
				</p>
			</div>
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300"
					>reason:review_requested -author:dependabot</code
				>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Review requests excluding Dependabot
				</p>
			</div>
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">-author:[bot]</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">Exclude all [bot] authors</p>
			</div>
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">reason:mention,comment state:open</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Mentions or comments on open items
				</p>
			</div>
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300"
					>(repo:cli/cli OR repo:desktop) AND type:issue</code
				>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Issues from either cli/cli or desktop repos
				</p>
			</div>
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">tags:urgent -tags:resolved</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Notifications tagged urgent but not resolved
				</p>
			</div>
			<div class="rounded-lg bg-gray-100/50 dark:bg-gray-800/50 px-3 py-2">
				<code class="text-indigo-600 dark:text-indigo-300">in:anywhere urgent security</code>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Search everywhere for "urgent" and "security" in title/repo/author
				</p>
			</div>
		</div>
	</section>

	<div
		class="rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50 px-3 py-2"
	>
		<p class="text-xs text-gray-600 dark:text-gray-400">
			<span class="font-medium text-gray-700 dark:text-gray-300">Tip:</span>
			Press
			<kbd
				class="rounded bg-gray-200 dark:bg-gray-800 px-1.5 py-0.5 text-xs text-gray-700 dark:text-gray-300"
				>Ctrl+Space</kbd
			>
			in the query input for autocomplete suggestions.
		</p>
	</div>
</div>
