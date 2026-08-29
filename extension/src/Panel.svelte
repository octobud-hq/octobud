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

	import { onMount, setContext } from "svelte";
	import { createPanelController } from "$lib/panel/panelController";
	import { PANEL_CONTROLLER_KEY } from "$lib/state/types";
	import { getBackendUrl, initBackendUrl } from "$lib/ext/backendUrl";
	import NotificationRow from "$lib/components/notification_view/NotificationRow.svelte";
	import CompactPagination from "$lib/components/shared/CompactPagination.svelte";
	import ViewSwitcher from "$lib/components/panel/ViewSwitcher.svelte";
	import PanelMessage from "$lib/components/panel/PanelMessage.svelte";
	import CustomSnoozeDialog from "$lib/components/panel/CustomSnoozeDialog.svelte";

	const controller = createPanelController();
	setContext(PANEL_CONTROLLER_KEY, controller);

	const { views, items, total, page, totalPages, loading, error, selectedSlug } = controller.stores;
	const { customSnoozeNotificationId } = controller.stores;

	const PAGE_SIZE = 30;

	let ready = false;

	onMount(async () => {
		await initBackendUrl();
		ready = true;
		await controller.actions.initialize();
	});

	function openApp() {
		window.open(getBackendUrl(), "_blank", "noopener,noreferrer");
	}

	function confirmCustomSnooze(until: string) {
		const target = $items.find(
			(item) => (item.githubId ?? item.id) === $customSnoozeNotificationId
		);
		controller.actions.closeCustomSnoozeDialog();
		if (target) {
			void controller.actions.snooze(target, until);
		}
	}

	$: rangeStart = $total === 0 ? 0 : ($page - 1) * PAGE_SIZE + 1;
	$: rangeEnd = Math.min($page * PAGE_SIZE, $total);
</script>

<ViewSwitcher
	views={$views}
	selectedSlug={$selectedSlug}
	busy={$loading}
	onSelectView={controller.actions.selectView}
	onOpenApp={openApp}
	onRefresh={() => void controller.actions.refresh()}
/>

<main class="flex min-h-0 flex-1 flex-col overflow-hidden bg-white dark:bg-gray-950">
	{#if !ready}
		<!-- Deliberately blank: initBackendUrl resolves in milliseconds against
		     localhost, and a spinner that flashes for one frame reads as jank. -->
		<div class="flex-1"></div>
	{:else if $error?.kind === "unreachable"}
		<PanelMessage
			tone="error"
			title="Can't reach Octobud"
			body={`Nothing is answering on ${getBackendUrl()}. Start the Octobud app and try again.`}
			actionLabel="Retry"
			onAction={() => void controller.actions.initialize()}
		/>
	{:else if $error?.kind === "not_connected"}
		<PanelMessage
			tone="error"
			title="GitHub isn't connected"
			body="Finish connecting your GitHub account in the Octobud app, then refresh."
			actionLabel="Open Octobud"
			onAction={openApp}
		/>
	{:else}
		<div class="flex items-center justify-between gap-2 px-3 pb-1 pt-2.5">
			<span class="truncate text-xs text-gray-500 dark:text-gray-500">
				{#if $total === 0}
					No notifications
				{:else if $total <= PAGE_SIZE}
					{$total}
					{$total === 1 ? "notification" : "notifications"}
				{:else}
					{rangeStart}–{rangeEnd} of {$total}
				{/if}
			</span>
			{#if $totalPages > 1}
				<CompactPagination
					page={$page}
					totalPages={$totalPages}
					onPrevious={() => void controller.actions.goToPage($page - 1)}
					onNext={() => void controller.actions.goToPage($page + 1)}
				/>
			{/if}
		</div>

		{#if $error?.kind === "request"}
			<div
				class="mx-3 mb-2 rounded-lg border border-amber-300 bg-amber-50 px-2.5 py-2 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
				role="alert"
			>
				{$error.message}
			</div>
		{/if}

		{#if $items.length === 0 && !$loading}
			<PanelMessage title="Nothing here yet" body="This view has no notifications right now." />
		{:else}
			<div class="flex-1 overflow-y-auto px-2 pb-4" style="scrollbar-gutter: stable;">
				<div class="space-y-2.5">
					{#each $items as notification (notification.id)}
						<NotificationRow {notification} />
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</main>

<CustomSnoozeDialog
	open={$customSnoozeNotificationId !== null}
	onConfirm={confirmCustomSnooze}
	onCancel={controller.actions.closeCustomSnoozeDialog}
/>
