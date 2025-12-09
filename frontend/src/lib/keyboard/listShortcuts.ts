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

/**
 * Keyboard Shortcut Handler using Command Registry Pattern
 *
 * Maps keyboard combinations to command names, which are executed via the command registry.
 */

import { executeCommand, type CommandContext } from "./commandRegistry";

function isFormFieldActive(active: Element | null): boolean {
	if (!active) {
		return false;
	}

	if (active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement) {
		return true;
	}

	return active instanceof HTMLElement && active.isContentEditable;
}

function isInteractiveElement(active: Element | null): boolean {
	if (!active) {
		return false;
	}
	if (isFormFieldActive(active)) {
		return true;
	}
	return (
		active instanceof HTMLButtonElement ||
		active instanceof HTMLAnchorElement ||
		active instanceof HTMLSelectElement ||
		active instanceof HTMLLabelElement
	);
}

function consumeEvent(event: KeyboardEvent) {
	event.preventDefault();
	event.stopPropagation();
	event.stopImmediatePropagation();
}

/**
 * Register keyboard shortcuts that map to commands in the command registry.
 * The context provides state getters that commands can use.
 */
export function registerListShortcuts(context: CommandContext): () => void {
	let lastGPressTime = 0;
	const DOUBLE_TAP_DELAY_MS = 500; // Time window for double-tap detection

	const handleKeydown = (event: KeyboardEvent) => {
		const active = document.activeElement;

		if (event.metaKey || event.ctrlKey) {
			const lowerKey = event.key.toLowerCase();
			if (lowerKey === "k") {
				// Cmd+K: Open palette with >view prepopulated
				// Cmd+Shift+K: Open palette empty
				const command = event.shiftKey ? "openPaletteEmpty" : "openPaletteView";
				const handled = executeCommand(command, context);
				if (handled) {
					consumeEvent(event);
				}
				return;
			}

			if (lowerKey === "b") {
				// Cmd/Ctrl+B: Toggle sidebar collapse (if in multiselect mode, use bulk palette)
				const multiselectActive = context.getMultiselectActive?.() ?? false;
				const command = multiselectActive ? "openPaletteBulk" : "toggleSidebar";
				const handled = executeCommand(command, context);
				if (handled) {
					consumeEvent(event);
				}
				return;
			}

			return;
		}

		if (event.altKey) {
			return;
		}

		const detailOpen = context.getDetailOpen();

		// Handle Escape in detail view to close it
		if (detailOpen && event.key === "Escape") {
			if (isFormFieldActive(active)) {
				return;
			}
			if (event.repeat) {
				return;
			}

			// Check if snooze dropdown is open - close it first
			const snoozeDropdownOpen = context.getSnoozeDropdownOpen?.() ?? false;
			if (snoozeDropdownOpen) {
				if (executeCommand("closeSnoozeDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check if tag dropdown is open - close it first
			const tagDropdownOpen = context.getTagDropdownOpen?.() ?? false;
			if (tagDropdownOpen) {
				if (executeCommand("closeTagDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check if bulk tag dropdown is open - close it first
			const bulkTagDropdownOpen = context.getBulkTagDropdownOpen?.() ?? false;
			if (bulkTagDropdownOpen) {
				if (executeCommand("closeBulkTagDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Close detail view with Escape
			if (executeCommand("toggleFocused", context)) {
				consumeEvent(event);
			}
			return;
		}

		// Handle Space in detail view (can toggle detail closed while keeping list item selected)
		if (detailOpen && (event.key === " " || event.key === "spacebar")) {
			if (isFormFieldActive(active)) {
				return;
			}
			if (event.repeat) {
				return;
			}
			if (isInteractiveElement(active)) {
				return;
			}
			if (executeCommand("toggleFocused", context)) {
				consumeEvent(event);
			}
			return;
		}

		// Handle action shortcuts when detail is open (e, r, m, s, z, t, i, o)
		// Note: Detail can be open in two scenarios:
		// 1. Split mode: detail is always open, list is visible - check multiselect first
		// 2. Single mode: user manually opened detail, list is hidden - always use focused notification
		if (detailOpen && !isFormFieldActive(active) && !event.repeat) {
			const key = event.key.toLowerCase();
			const splitViewMode = context.getSplitViewMode?.() ?? false;
			// Only check multiselect in split mode (where list is visible)
			// In single mode, list is hidden so multiselect selections shouldn't be acted upon
			const multiselectActive = splitViewMode ? (context.getMultiselectActive?.() ?? false) : false;
			const hasSelection = splitViewMode ? (context.getHasSelection?.() ?? false) : false;

			if (key === "e") {
				// Archive/unarchive
				// Check if Shift is pressed for unarchive
				if (event.shiftKey) {
					// Shift+E: Unarchive
					if (multiselectActive && hasSelection) {
						if (executeCommand("bulkUnarchive", context)) {
							consumeEvent(event);
						}
					}
					// No single item unarchive action currently
				} else {
					// E: Archive
					// In multiselect mode with selections, use bulk action
					if (multiselectActive && hasSelection) {
						if (executeCommand("bulkArchive", context)) {
							consumeEvent(event);
						}
					} else {
						// Otherwise, use single item action
						if (executeCommand("markFocusedArchive", context)) {
							consumeEvent(event);
						}
					}
				}
				return;
			}

			if (key === "r") {
				// Mark read/unread
				if (event.shiftKey) {
					// Shift+R: Mark unread (bulk, multiselect only)
					if (multiselectActive && hasSelection) {
						if (executeCommand("bulkMarkUnread", context)) {
							consumeEvent(event);
						}
					}
				} else {
					// R: Mark read
					// In multiselect mode with selections, use bulk action
					if (multiselectActive && hasSelection) {
						if (executeCommand("bulkMarkRead", context)) {
							consumeEvent(event);
						}
					} else {
						// Otherwise, use single item action
						if (executeCommand("markFocusedRead", context)) {
							consumeEvent(event);
						}
					}
				}
				return;
			}

			if (key === "m") {
				// Mute/unmute
				// In multiselect mode with selections, use bulk action
				if (multiselectActive && hasSelection) {
					if (event.shiftKey) {
						// Shift+M: bulk unmute
						if (executeCommand("bulkUnmute", context)) {
							consumeEvent(event);
						}
					} else {
						// M: bulk mute
						if (executeCommand("bulkMute", context)) {
							consumeEvent(event);
						}
					}
				} else {
					// Otherwise, use single item action
					if (executeCommand("markFocusedMute", context)) {
						consumeEvent(event);
					}
				}
				return;
			}

			if (key === "s") {
				// Star/unstar
				// In multiselect mode with selections, use bulk action
				if (multiselectActive && hasSelection) {
					if (event.shiftKey) {
						// Shift+S: bulk unstar
						if (executeCommand("bulkUnstar", context)) {
							consumeEvent(event);
						}
					} else {
						// S: bulk star
						if (executeCommand("bulkStar", context)) {
							consumeEvent(event);
						}
					}
				} else {
					// Otherwise, use single item action (toggle star)
					if (executeCommand("markFocusedStar", context)) {
						consumeEvent(event);
					}
				}
				return;
			}

			if (key === "z") {
				// Snooze - check if dropdowns are already open
				const snoozeDropdownOpen = context.getSnoozeDropdownOpen?.() ?? false;
				const bulkSnoozeDropdownOpen = context.getBulkSnoozeDropdownOpen?.() ?? false;

				if (snoozeDropdownOpen) {
					// If individual snooze dropdown is open, close it
					if (executeCommand("closeSnoozeDropdown", context)) {
						consumeEvent(event);
					}
					return;
				}

				if (bulkSnoozeDropdownOpen) {
					// If bulk snooze dropdown is open, close it
					if (executeCommand("closeBulkSnoozeDropdown", context)) {
						consumeEvent(event);
					}
					return;
				}

				// Check if we're in multiselect mode with selections
				if (multiselectActive && hasSelection) {
					// Open bulk snooze dropdown
					if (executeCommand("openBulkSnoozeDropdown", context)) {
						consumeEvent(event);
					}
					return;
				}

				// Otherwise, open individual snooze for focused notification
				if (executeCommand("openFocusedSnoozeDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			if (key === "t") {
				// Tags - handle tag dropdown for focused notification or bulk tag in multiselect
				// Check if tag dropdown is already open
				const tagDropdownOpen = context.getTagDropdownOpen?.() ?? false;
				const bulkTagDropdownOpen = context.getBulkTagDropdownOpen?.() ?? false;

				if (tagDropdownOpen) {
					// If individual tag dropdown is open, close it
					if (executeCommand("closeTagDropdown", context)) {
						consumeEvent(event);
					}
					return;
				}

				if (bulkTagDropdownOpen) {
					// If bulk tag dropdown is open, close it
					if (executeCommand("closeBulkTagDropdown", context)) {
						consumeEvent(event);
					}
					return;
				}

				// Check if we're in multiselect mode with selections
				if (multiselectActive && hasSelection) {
					// Open bulk tag dropdown
					if (executeCommand("openBulkTagDropdown", context)) {
						consumeEvent(event);
					}
					return;
				}

				// Otherwise, open individual tag for focused notification
				if (executeCommand("openFocusedTagDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			if (key === "i") {
				// Move to inbox (unfilter)
				// In multiselect mode with selections, use bulk action
				if (multiselectActive && hasSelection) {
					if (executeCommand("bulkUnfilter", context)) {
						consumeEvent(event);
					}
				} else {
					// Otherwise, use single item action
					if (executeCommand("markFocusedUnfilter", context)) {
						consumeEvent(event);
					}
				}
				return;
			}

			if (key === "o") {
				// Open in GitHub - always use focused notification (no bulk action for this)
				if (executeCommand("openFocusedInGithub", context)) {
					consumeEvent(event);
				}
				return;
			}
		}

		// J/K navigation works the same whether detail is open or not
		// The handler will decide whether to auto-open based on reading pane state

		if (event.key === "Escape") {
			// Check if snooze dropdown is open - close it first
			const snoozeDropdownOpen = context.getSnoozeDropdownOpen?.() ?? false;
			if (snoozeDropdownOpen) {
				if (executeCommand("closeSnoozeDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check if tag dropdown is open - close it first
			const tagDropdownOpen = context.getTagDropdownOpen?.() ?? false;
			if (tagDropdownOpen) {
				if (executeCommand("closeTagDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check if bulk snooze dropdown is open - close it first
			const bulkSnoozeDropdownOpen = context.getBulkSnoozeDropdownOpen?.() ?? false;
			if (bulkSnoozeDropdownOpen) {
				if (executeCommand("closeBulkSnoozeDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check if bulk tag dropdown is open - close it first
			const bulkTagDropdownOpen = context.getBulkTagDropdownOpen?.() ?? false;
			if (bulkTagDropdownOpen) {
				if (executeCommand("closeBulkTagDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check if any dialogs/modals are open - they should handle Escape first
			if (context.isAnyDialogOpen()) {
				return;
			}

			// Check if focus is in a form field - inputs should handle Escape first
			if (isFormFieldActive(active)) {
				return;
			}

			// In multiselect mode: first clear selection, second press exits multiselect
			const multiselectActive = context.getMultiselectActive?.() ?? false;
			const hasSelection = context.getHasSelection?.() ?? false;

			if (multiselectActive && hasSelection) {
				// First Escape: clear selection
				if (executeCommand("clearSelection", context)) {
					consumeEvent(event);
					return;
				}
			}

			if (multiselectActive && !hasSelection) {
				// Second Escape: exit multiselect mode
				if (executeCommand("toggleMultiselectMode", context)) {
					consumeEvent(event);
					return;
				}
			}

			// Reset keyboard focus if it's set
			if (executeCommand("resetFocus", context)) {
				consumeEvent(event);
			}
			return;
		}

		// Block all other shortcuts if any dialog/modal is open (except Escape handled above)
		if (context.isAnyDialogOpen()) {
			return;
		}

		if (isFormFieldActive(active)) {
			return;
		}

		if (event.key === "/") {
			consumeEvent(event);
			executeCommand("focusViewSearch", context);
			return;
		}

		// Handle F key to toggle filter dropdown
		const key = event.key.toLowerCase();

		// Handle P key for display/reading pane toggle
		if (key === "p" && event.shiftKey) {
			// Shift+P: toggle reading pane (display mode)
			if (executeCommand("toggleSplitMode", context)) {
				consumeEvent(event);
			}
			return;
		}

		// Handle H key to toggle shortcuts modal
		if (key === "h") {
			if (executeCommand("toggleShortcutsModal", context)) {
				consumeEvent(event);
			}
			return;
		}

		// Handle V key to toggle multiselect mode or open bulk palette
		if (key === "v") {
			// Don't activate multiselect mode if detail is open in list mode (not split view)
			const splitViewMode = context.getSplitViewMode?.() ?? false;
			if (detailOpen && !splitViewMode && !event.shiftKey) {
				return;
			}

			if (event.shiftKey) {
				// Shift+V: Open bulk command palette
				if (executeCommand("openPaletteBulk", context)) {
					consumeEvent(event);
				}
			} else {
				// V: Toggle multiselect mode
				if (executeCommand("toggleMultiselectMode", context)) {
					consumeEvent(event);
				}
			}
			return;
		}

		// Handle Z key to toggle snooze dropdown for focused notification or bulk snooze in multiselect
		if (key === "z") {
			// Check if snooze dropdown is already open
			const snoozeDropdownOpen = context.getSnoozeDropdownOpen?.() ?? false;
			const bulkSnoozeDropdownOpen = context.getBulkSnoozeDropdownOpen?.() ?? false;
			const multiselectActive = context.getMultiselectActive?.() ?? false;
			const hasSelection = context.getHasSelection?.() ?? false;

			if (snoozeDropdownOpen) {
				// If individual snooze dropdown is open, close it
				if (executeCommand("closeSnoozeDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			if (bulkSnoozeDropdownOpen) {
				// If bulk snooze dropdown is open, close it
				if (executeCommand("closeBulkSnoozeDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check if we're in multiselect mode with selections
			if (multiselectActive && hasSelection) {
				// Open bulk snooze dropdown
				if (executeCommand("openBulkSnoozeDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Otherwise, open individual snooze for focused notification
			if (executeCommand("openFocusedSnoozeDropdown", context)) {
				consumeEvent(event);
			}
			return;
		}

		// Handle number keys 1-5 when snooze dropdown is open
		// Let the dropdown handle these keys itself
		const snoozeDropdownOpen = context.getSnoozeDropdownOpen?.() ?? false;
		const bulkSnoozeDropdownOpen = context.getBulkSnoozeDropdownOpen?.() ?? false;
		if ((snoozeDropdownOpen || bulkSnoozeDropdownOpen) && key >= "1" && key <= "5") {
			// Don't consume the event - let the SnoozeDropdown component handle it
			return;
		}

		// Multiselect-specific shortcuts
		const multiselectActive = context.getMultiselectActive?.() ?? false;
		const hasSelection = context.getHasSelection?.() ?? false;

		// Handle X key to toggle selection of focused item (multiselect mode only)
		if (key === "x" && multiselectActive) {
			if (executeCommand("toggleFocusedSelection", context)) {
				consumeEvent(event);
			}
			return;
		}

		// Handle A key to cycle select all modes (multiselect mode only)
		if (key === "a" && multiselectActive) {
			if (executeCommand("cycleSelectAll", context)) {
				consumeEvent(event);
			}
			return;
		}

		// Handle G key (vim-style navigation to last item)
		if (key === "g") {
			if (event.shiftKey) {
				// Shift+G: jump to last notification (last page)
				if (executeCommand("focusLast", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check for double-tap 'g' (gg)
			const now = Date.now();
			const timeSinceLastG = now - lastGPressTime;

			if (timeSinceLastG < DOUBLE_TAP_DELAY_MS) {
				// Second 'g' within the time window - jump to first notification
				lastGPressTime = 0; // Reset to avoid triple-tap
				if (executeCommand("focusFirst", context)) {
					consumeEvent(event);
				}
				return;
			} else {
				// First 'g' press - start timer
				lastGPressTime = now;
				consumeEvent(event);
				return;
			}
		}

		// Handle [ and ] for page navigation
		// Don't allow page navigation if detail is open in list mode (not split view)
		const splitViewMode = context.getSplitViewMode?.() ?? false;
		if (key === "[") {
			// Skip in list mode with detail open
			if (detailOpen && !splitViewMode) {
				return;
			}
			// [: jump to previous page (focus last item)
			if (executeCommand("goToPreviousPage", context)) {
				consumeEvent(event);
			}
			return;
		}

		if (key === "]") {
			// Skip in list mode with detail open
			if (detailOpen && !splitViewMode) {
				return;
			}
			// ]: jump to next page (focus first item)
			if (executeCommand("goToNextPage", context)) {
				consumeEvent(event);
			}
			return;
		}

		if (key === "j") {
			if (event.shiftKey) {
				// Shift+J: navigate to next view (down the view list)
				if (executeCommand("navigateNextView", context)) {
					consumeEvent(event);
				}
			} else {
				// J: navigate to next notification
				// Close snooze dropdown if open
				const snoozeDropdownOpen = context.getSnoozeDropdownOpen?.() ?? false;
				if (snoozeDropdownOpen) {
					// This will be handled by clicking outside
					return;
				}
				if (executeCommand("focusNext", context)) {
					consumeEvent(event);
				}
			}
		} else if (key === "k") {
			if (event.shiftKey) {
				// Shift+K: navigate to previous view (up the view list)
				if (executeCommand("navigatePreviousView", context)) {
					consumeEvent(event);
				}
			} else {
				// K: navigate to previous notification
				// Close snooze dropdown if open
				const snoozeDropdownOpen = context.getSnoozeDropdownOpen?.() ?? false;
				if (snoozeDropdownOpen) {
					// This will be handled by clicking outside
					return;
				}
				if (executeCommand("focusPrevious", context)) {
					consumeEvent(event);
				}
			}
		} else if (key === " " || key === "spacebar") {
			if (event.repeat) {
				return;
			}
			if (executeCommand("toggleFocused", context)) {
				consumeEvent(event);
			}
		} else if (key === "e") {
			// Check if Shift is pressed for unarchive
			if (event.shiftKey) {
				// Shift+E: Unarchive
				if (multiselectActive && hasSelection) {
					if (executeCommand("bulkUnarchive", context)) {
						consumeEvent(event);
					}
				}
				// No single item unarchive action currently
			} else {
				// E: Archive
				// In multiselect mode with selections, use bulk action
				if (multiselectActive && hasSelection) {
					if (executeCommand("bulkArchive", context)) {
						consumeEvent(event);
					}
				} else {
					// Otherwise, use single item action
					if (executeCommand("markFocusedArchive", context)) {
						consumeEvent(event);
					}
				}
			}
		} else if (key === "r") {
			// Mark read/unread
			if (event.shiftKey) {
				// Shift+R: Mark unread (bulk, multiselect only)
				if (multiselectActive && hasSelection) {
					if (executeCommand("bulkMarkUnread", context)) {
						consumeEvent(event);
					}
				}
			} else {
				// R: Mark read
				// In multiselect mode with selections, use bulk action
				if (multiselectActive && hasSelection) {
					if (executeCommand("bulkMarkRead", context)) {
						consumeEvent(event);
					}
				} else {
					// Otherwise, use single item action
					if (executeCommand("markFocusedRead", context)) {
						consumeEvent(event);
					}
				}
			}
		} else if (key === "m") {
			// Mute/unmute
			// In multiselect mode with selections, use bulk action
			if (multiselectActive && hasSelection) {
				if (event.shiftKey) {
					// Shift+M: bulk unmute
					if (executeCommand("bulkUnmute", context)) {
						consumeEvent(event);
					}
				} else {
					// M: bulk mute
					if (executeCommand("bulkMute", context)) {
						consumeEvent(event);
					}
				}
			} else {
				// Otherwise, use single item action
				if (executeCommand("markFocusedMute", context)) {
					consumeEvent(event);
				}
			}
		} else if (key === "o") {
			if (executeCommand("openFocusedInGithub", context)) {
				consumeEvent(event);
			}
		} else if (key === "s") {
			// Star/unstar shortcut
			// In multiselect mode with selections, use bulk action
			if (multiselectActive && hasSelection) {
				if (event.shiftKey) {
					// Shift+S: bulk unstar
					if (executeCommand("bulkUnstar", context)) {
						consumeEvent(event);
					}
				} else {
					// S: bulk star
					if (executeCommand("bulkStar", context)) {
						consumeEvent(event);
					}
				}
			} else {
				// Otherwise, use single item action (toggle star)
				if (executeCommand("markFocusedStar", context)) {
					consumeEvent(event);
				}
			}
		} else if (key === "z") {
			// Snooze - open snooze dropdown for focused notification
			if (executeCommand("openFocusedSnoozeDropdown", context)) {
				consumeEvent(event);
			}
		} else if (key === "t") {
			// Tags - handle tag dropdown for focused notification or bulk tag in multiselect
			// Check if tag dropdown is already open
			const tagDropdownOpen = context.getTagDropdownOpen?.() ?? false;
			const bulkTagDropdownOpen = context.getBulkTagDropdownOpen?.() ?? false;
			const multiselectActive = context.getMultiselectActive?.() ?? false;
			const hasSelection = context.getHasSelection?.() ?? false;

			if (tagDropdownOpen) {
				// If individual tag dropdown is open, close it
				if (executeCommand("closeTagDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			if (bulkTagDropdownOpen) {
				// If bulk tag dropdown is open, close it
				if (executeCommand("closeBulkTagDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Check if we're in multiselect mode with selections
			if (multiselectActive && hasSelection) {
				// Open bulk tag dropdown
				if (executeCommand("openBulkTagDropdown", context)) {
					consumeEvent(event);
				}
				return;
			}

			// Otherwise, open individual tag for focused notification
			if (executeCommand("openFocusedTagDropdown", context)) {
				consumeEvent(event);
			}
		} else if (key === "i") {
			// Move to inbox (unfilter)
			// In multiselect mode with selections, use bulk action
			if (multiselectActive && hasSelection) {
				if (executeCommand("bulkUnfilter", context)) {
					consumeEvent(event);
				}
			} else {
				// Otherwise, use single item action
				if (executeCommand("markFocusedUnfilter", context)) {
					consumeEvent(event);
				}
			}
		}
	};

	const listenerOptions = { capture: true } as const;
	window.addEventListener("keydown", handleKeydown, listenerOptions);
	return () => {
		window.removeEventListener("keydown", handleKeydown, listenerOptions);
	};
}
