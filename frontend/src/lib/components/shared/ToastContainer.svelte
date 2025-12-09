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

	import { toastStore } from "$lib/stores/toastStore";
	import { fly, fade } from "svelte/transition";
	import { quintOut } from "svelte/easing";

	$: toasts = $toastStore.toasts;

	function getToastStyles(type: string) {
		switch (type) {
			case "success":
				return "bg-emerald-500/90 text-white border-emerald-400";
			case "error":
				return "bg-rose-500/90 text-white border-rose-400";
			case "warning":
				return "bg-amber-500/90 text-white border-amber-400";
			case "info":
			default:
				return "bg-indigo-500 text-white border-indigo-500";
		}
	}

	function getIconPath(type: string) {
		switch (type) {
			case "success":
				return "M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z";
			case "error":
				return "M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z";
			case "warning":
				return "M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z";
			case "info":
			default:
				return "M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z";
		}
	}
</script>

<div
	class="pointer-events-none fixed right-4 top-4 z-[200] flex flex-col gap-3"
	aria-live="polite"
	aria-atomic="true"
>
	{#each toasts as toast (toast.id)}
		<div
			in:fly={{ duration: 300, x: 300, easing: quintOut }}
			out:fade={{ duration: 200 }}
			class={`pointer-events-auto flex items-center gap-3 rounded-xl border px-4 py-3 shadow-2xl backdrop-blur-sm ${getToastStyles(toast.type)}`}
			role="alert"
		>
			<svg
				class="h-5 w-5 flex-shrink-0"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				aria-hidden="true"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d={getIconPath(toast.type)}
				/>
			</svg>
			<span class="text-sm font-medium">{toast.message}</span>
			<button
				type="button"
				class="ml-2 flex h-6 w-6 items-center justify-center rounded-lg transition hover:bg-white/20"
				on:click={() => toastStore.dismiss(toast.id)}
				aria-label="Dismiss notification"
			>
				<span class="text-lg leading-none" aria-hidden="true">×</span>
			</button>
		</div>
	{/each}
</div>
