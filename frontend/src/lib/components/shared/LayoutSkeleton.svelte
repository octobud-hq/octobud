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

	// Loading screen shown before hydration
	// Displays the logo centered on a clean background with a squish animation
</script>

<div class="h-screen w-full flex items-center justify-center bg-gray-50 dark:bg-gray-950">
	<!-- Logo with colorful squircle background -->
	<div
		class="logo-squish flex h-20 w-20 items-center justify-center rounded-3xl bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500"
		style="box-shadow: 0 10px 40px -10px rgba(99, 102, 241, 0.5), 0 4px 12px -2px rgba(139, 92, 246, 0.3);"
	>
		<img src="/baby_octo.svg" alt="Octobud" class="h-12 w-12" />
	</div>
</div>

<style>
	@keyframes squish {
		0% {
			transform: scale(0.7);
			opacity: 0;
		}
		50% {
			transform: scale(1.05);
		}
		100% {
			transform: scale(1);
			opacity: 1;
		}
	}

	.logo-squish {
		animation: squish 0.4s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
	}
</style>
