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

	import type { NotificationTimelineItem } from "$lib/api/types";
	import Comment from "./Comment.svelte";
	import TimelineCommit from "./TimelineCommit.svelte";
	import TimelineEvent from "./TimelineEvent.svelte";

	export let item: NotificationTimelineItem;
	export let showThread: boolean = true;
	export let isLastItem: boolean = false;

	// Determine which component to use based on event type
	$: component = getComponent(item.type);

	function getComponent(type: string) {
		// Events with body content use Comment component
		if (type === "comment" || type === "commented" || type === "reviewed") {
			return Comment;
		}
		// Commits use TimelineCommit component
		if (type === "committed") {
			return TimelineCommit;
		}
		// All other events use generic TimelineEvent
		return TimelineEvent;
	}
</script>

<svelte:component this={component} {item} {showThread} {isLastItem} />
