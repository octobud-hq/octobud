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

	import { createEventDispatcher, onMount, onDestroy } from "svelte";

	const dispatch = createEventDispatcher<{
		resize: { deltaX: number };
	}>();

	let isResizing = false;
	let startX = 0;

	function handleMouseDown(e: MouseEvent) {
		isResizing = true;
		startX = e.clientX;
		e.preventDefault();
	}

	function handleMouseMove(e: MouseEvent) {
		if (!isResizing) return;

		const deltaX = e.clientX - startX;
		dispatch("resize", { deltaX });
		startX = e.clientX;
	}

	function handleMouseUp() {
		isResizing = false;
	}

	onMount(() => {
		if (typeof window !== "undefined") {
			window.addEventListener("mousemove", handleMouseMove);
			window.addEventListener("mouseup", handleMouseUp);
		}
	});

	onDestroy(() => {
		if (typeof window !== "undefined") {
			window.removeEventListener("mousemove", handleMouseMove);
			window.removeEventListener("mouseup", handleMouseUp);
		}
	});
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
	class="w-1 cursor-col-resize bg-gray-200 dark:bg-gray-800 hover:bg-indigo-500 transition-colors flex-shrink-0"
	on:mousedown={handleMouseDown}
	role="separator"
	aria-orientation="vertical"
	aria-label="Resize panes"
	tabindex="0"
	on:keydown={(e) => {
		if (e.key === "ArrowLeft") {
			dispatch("resize", { deltaX: -10 });
		} else if (e.key === "ArrowRight") {
			dispatch("resize", { deltaX: 10 });
		}
	}}
></div>
