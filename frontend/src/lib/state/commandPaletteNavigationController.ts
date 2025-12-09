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

import { writable, derived } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import { tick } from "svelte";

export interface NavigationState {
	rootCommandIndex: number;
	viewCommandIndex: number;
	lastRootQuery: string;
	lastViewQuery: string;
}

export interface CommandPaletteNavigationControllerOptions {
	isOpen: Readable<boolean>;
	rootCommandCount: Readable<number>;
	viewMatchCount: Readable<number>;
	activeCommandId: Readable<string | null>;
	unknownCommandToken: Readable<string | null>;
	rootQuery: Readable<string>;
	viewQuery: Readable<string>;
	containerElement: () => HTMLElement | null;
}

export interface CommandPaletteNavigationControllerStores {
	rootCommandIndex: Writable<number>;
	viewCommandIndex: Writable<number>;
}

export interface CommandPaletteNavigationControllerActions {
	moveRootCommandSelection: (delta: number) => void;
	moveViewCommandSelection: (delta: number) => void;
	handleRootCommandPointer: (index: number) => void;
	handleViewPointer: (index: number) => void;
	scrollToRootCommand: (index: number) => void;
	scrollToView: (index: number) => void;
	resetNavigation: () => void;
}

export interface CommandPaletteNavigationController {
	stores: CommandPaletteNavigationControllerStores;
	actions: CommandPaletteNavigationControllerActions;
}

/**
 * Creates a controller for managing keyboard navigation within the command palette
 */
export function createCommandPaletteNavigationController(
	options: CommandPaletteNavigationControllerOptions
): CommandPaletteNavigationController {
	const {
		isOpen,
		rootCommandCount,
		viewMatchCount,
		activeCommandId,
		unknownCommandToken,
		rootQuery,
		viewQuery,
		containerElement,
	} = options;

	const rootCommandIndex = writable<number>(0);
	const viewCommandIndex = writable<number>(0);
	const lastRootQuery = writable<string>("");
	const lastViewQuery = writable<string>("");

	// Reset indices when palette is closed
	isOpen.subscribe(($isOpen) => {
		if (!$isOpen) {
			rootCommandIndex.set(0);
			lastRootQuery.set("");
			viewCommandIndex.set(0);
			lastViewQuery.set("");
		}
	});

	// Reset root command index when root query changes
	derived(
		[rootQuery, activeCommandId, unknownCommandToken],
		([$rootQuery, $activeCommandId, $unknownCommandToken]) => {
			if (!$activeCommandId && !$unknownCommandToken) {
				return $rootQuery;
			}
			return null;
		}
	).subscribe((currentQuery) => {
		if (currentQuery !== null) {
			let lastQuery = "";
			lastRootQuery.subscribe((value) => {
				lastQuery = value;
			})();

			if (currentQuery !== lastQuery) {
				rootCommandIndex.set(0);
				lastRootQuery.set(currentQuery);
			}
		}
	});

	// Reset view command index when switching to view mode
	activeCommandId.subscribe(($activeCommandId) => {
		if ($activeCommandId !== "view") {
			viewCommandIndex.set(0);
		}
	});

	// Reset view command index when view query changes
	derived([viewQuery, activeCommandId], ([$viewQuery, $activeCommandId]) => {
		if ($activeCommandId === "view") {
			return $viewQuery;
		}
		return null;
	}).subscribe((currentQuery) => {
		if (currentQuery !== null) {
			let lastQuery = "";
			lastViewQuery.subscribe((value) => {
				lastQuery = value;
			})();

			if (currentQuery !== lastQuery) {
				viewCommandIndex.set(0);
				lastViewQuery.set(currentQuery);
			}
		}
	});

	// Clamp root command index to valid range
	derived(
		[rootCommandCount, rootCommandIndex, activeCommandId, unknownCommandToken],
		([$rootCommandCount, $rootCommandIndex, $activeCommandId, $unknownCommandToken]) => {
			if (!$activeCommandId && !$unknownCommandToken) {
				if ($rootCommandCount === 0) {
					return -1;
				}
				if ($rootCommandIndex < 0 || $rootCommandIndex >= $rootCommandCount) {
					return Math.max(0, Math.min($rootCommandIndex, $rootCommandCount - 1));
				}
			} else if ($rootCommandIndex !== 0) {
				return 0;
			}
			return $rootCommandIndex;
		}
	).subscribe((clampedIndex) => {
		rootCommandIndex.update((current) => {
			if (current !== clampedIndex) {
				return clampedIndex;
			}
			return current;
		});
	});

	// Clamp view command index to valid range
	derived(
		[viewMatchCount, viewCommandIndex, activeCommandId],
		([$viewMatchCount, $viewCommandIndex, $activeCommandId]) => {
			if ($activeCommandId === "view") {
				if ($viewMatchCount === 0) {
					return -1;
				}
				if ($viewCommandIndex < 0 || $viewCommandIndex >= $viewMatchCount) {
					return Math.max(0, Math.min($viewCommandIndex, $viewMatchCount - 1));
				}
			}
			return $viewCommandIndex;
		}
	).subscribe((clampedIndex) => {
		viewCommandIndex.update((current) => {
			if (current !== clampedIndex) {
				return clampedIndex;
			}
			return current;
		});
	});

	function moveRootCommandSelection(delta: number) {
		let count = 0;
		rootCommandCount.subscribe((value) => {
			count = value;
		})();

		if (count === 0) {
			return;
		}

		rootCommandIndex.update((current) => {
			const baseIndex = current < 0 ? 0 : current;
			return (baseIndex + delta + count) % count;
		});
	}

	function moveViewCommandSelection(delta: number) {
		let count = 0;
		viewMatchCount.subscribe((value) => {
			count = value;
		})();

		if (count === 0) {
			return;
		}

		viewCommandIndex.update((current) => {
			if (current === -1) {
				return delta > 0 ? 0 : count - 1;
			}
			let nextIndex = current + delta;
			if (nextIndex < 0) {
				nextIndex = count - 1;
			} else if (nextIndex >= count) {
				nextIndex = 0;
			}
			return nextIndex;
		});
	}

	function handleRootCommandPointer(index: number) {
		let count = 0;
		rootCommandCount.subscribe((value) => {
			count = value;
		})();

		if (index < 0 || index >= count) {
			return;
		}
		rootCommandIndex.set(index);
	}

	function handleViewPointer(index: number) {
		let count = 0;
		viewMatchCount.subscribe((value) => {
			count = value;
		})();

		if (index < 0 || index >= count) {
			return;
		}
		viewCommandIndex.set(index);
	}

	function scrollToRootCommand(index: number) {
		let $isOpen = false;
		isOpen.subscribe((value) => {
			$isOpen = value;
		})();

		let $activeCommandId: string | null = null;
		activeCommandId.subscribe((value) => {
			$activeCommandId = value;
		})();

		let $unknownCommandToken: string | null = null;
		unknownCommandToken.subscribe((value) => {
			$unknownCommandToken = value;
		})();

		if ($isOpen && index >= 0 && !$activeCommandId && !$unknownCommandToken) {
			void tick().then(() => {
				const container = containerElement();
				const element = container?.querySelector(`[data-root-command-index="${index}"]`);
				if (element) {
					element.scrollIntoView({
						block: "nearest",
						behavior: "smooth",
					});
				}
			});
		}
	}

	function scrollToView(index: number) {
		let $isOpen = false;
		isOpen.subscribe((value) => {
			$isOpen = value;
		})();

		let $activeCommandId: string | null = null;
		activeCommandId.subscribe((value) => {
			$activeCommandId = value;
		})();

		if ($isOpen && $activeCommandId === "view" && index >= 0) {
			void tick().then(() => {
				const container = containerElement();
				const element = container?.querySelector(`[data-view-index="${index}"]`);
				if (element) {
					element.scrollIntoView({
						block: "nearest",
						behavior: "smooth",
					});
				}
			});
		}
	}

	// Auto-scroll when indices change
	rootCommandIndex.subscribe((index) => {
		scrollToRootCommand(index);
	});

	viewCommandIndex.subscribe((index) => {
		scrollToView(index);
	});

	function resetNavigation() {
		rootCommandIndex.set(0);
		viewCommandIndex.set(0);
		lastRootQuery.set("");
		lastViewQuery.set("");
	}

	return {
		stores: {
			rootCommandIndex,
			viewCommandIndex,
		},
		actions: {
			moveRootCommandSelection,
			moveViewCommandSelection,
			handleRootCommandPointer,
			handleViewPointer,
			scrollToRootCommand,
			scrollToView,
			resetNavigation,
		},
	};
}
