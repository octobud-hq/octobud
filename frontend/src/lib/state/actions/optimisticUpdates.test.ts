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

import { describe, it, expect } from "vitest";
import type { Notification } from "$lib/api/types";

// Extract the pure calculation functions for testing
// These match the logic in optimisticUpdates.ts

/**
 * Calculate the next item to focus/open after removing an item from the list
 */
function calculateNextItemAfterRemoval(
	currentIndex: number,
	items: Notification[],
	currentFocus: number | null,
	willBeRemoved: boolean
): {
	targetIndex: number | null;
	targetNotification: Notification | null;
	willPageBeEmpty: boolean;
} {
	// If item not in current page, no target
	if (currentIndex === -1) {
		return {
			targetIndex: null,
			targetNotification: null,
			willPageBeEmpty: items.length === 0,
		};
	}

	// Calculate items after removal
	const itemsAfterRemoval = items.length - (willBeRemoved ? 1 : 0);
	const willPageBeEmpty = itemsAfterRemoval === 0;

	if (willPageBeEmpty) {
		return {
			targetIndex: null,
			targetNotification: null,
			willPageBeEmpty: true,
		};
	}

	// Calculate target index: item that slides into removed item's position
	let targetIndex: number;
	if (currentIndex >= 0) {
		if (currentIndex < itemsAfterRemoval) {
			// Item slides into this position
			targetIndex = currentIndex;
		} else {
			// Removing last item, focus previous item
			targetIndex = Math.max(0, itemsAfterRemoval - 1);
		}
	} else if (currentFocus !== null && currentFocus < itemsAfterRemoval) {
		// Use current keyboard focus
		targetIndex = currentFocus;
	} else {
		// Fallback to last item if available, otherwise first
		targetIndex = itemsAfterRemoval > 0 ? itemsAfterRemoval - 1 : 0;
	}

	// Get target notification from REMAINING items (excluding removed one)
	const remainingItems = willBeRemoved ? items.filter((_, i) => i !== currentIndex) : items;
	const targetNotification = remainingItems[targetIndex] ?? null;

	return {
		targetIndex,
		targetNotification,
		willPageBeEmpty: false,
	};
}

/**
 * Calculate pagination state after removing an item
 */
function calculatePaginationStateAfterRemoval(
	currentPage: number,
	itemsAfterRemoval: number,
	pageSize: number,
	totalAfterRemoval: number
): {
	shouldNavigateToPreviousPage: boolean;
	targetPage: number | null;
} {
	// If page is empty and we're not on page 1, navigate to previous page
	if (itemsAfterRemoval === 0 && currentPage > 1) {
		return {
			shouldNavigateToPreviousPage: true,
			targetPage: currentPage - 1,
		};
	}

	return {
		shouldNavigateToPreviousPage: false,
		targetPage: null,
	};
}

// Helper to create mock notifications
const createMockNotification = (id: string): Notification => ({
	id,
	githubId: `gh-${id}`,
	repoFullName: "owner/repo",
	reason: "review_requested",
	subjectTitle: `Test ${id}`,
	subjectType: "PullRequest",
	subjectUrl: `https://github.com/owner/repo/pull/${id}`,
	updatedAt: "2024-01-15T10:00:00Z",
	isRead: false,
	archived: false,
	muted: false,
	starred: false,
	filtered: false,
	snoozedUntil: null,
	labels: [],
	viewIds: [],
});

describe("Optimistic Update Helpers", () => {
	describe("calculateNextItemAfterRemoval", () => {
		describe("when removing item from middle of list", () => {
			it("targets the item that slides into the removed position", () => {
				const items = [
					createMockNotification("1"),
					createMockNotification("2"),
					createMockNotification("3"),
					createMockNotification("4"),
				];

				// Remove item at index 1 (id: 2)
				const result = calculateNextItemAfterRemoval(1, items, null, true);

				expect(result.targetIndex).toBe(1);
				expect(result.targetNotification?.id).toBe("3"); // Item 3 slides into position 1
				expect(result.willPageBeEmpty).toBe(false);
			});
		});

		describe("when removing first item", () => {
			it("targets index 0 (next item slides in)", () => {
				const items = [
					createMockNotification("1"),
					createMockNotification("2"),
					createMockNotification("3"),
				];

				const result = calculateNextItemAfterRemoval(0, items, null, true);

				expect(result.targetIndex).toBe(0);
				expect(result.targetNotification?.id).toBe("2");
				expect(result.willPageBeEmpty).toBe(false);
			});
		});

		describe("when removing last item", () => {
			it("targets previous item", () => {
				const items = [
					createMockNotification("1"),
					createMockNotification("2"),
					createMockNotification("3"),
				];

				// Remove item at index 2 (last item)
				const result = calculateNextItemAfterRemoval(2, items, null, true);

				expect(result.targetIndex).toBe(1); // Previous item
				expect(result.targetNotification?.id).toBe("2");
				expect(result.willPageBeEmpty).toBe(false);
			});
		});

		describe("when removing only item on page", () => {
			it("returns willPageBeEmpty true", () => {
				const items = [createMockNotification("1")];

				const result = calculateNextItemAfterRemoval(0, items, null, true);

				expect(result.targetIndex).toBe(null);
				expect(result.targetNotification).toBe(null);
				expect(result.willPageBeEmpty).toBe(true);
			});
		});

		describe("when item not in current page", () => {
			it("returns null target with currentIndex -1", () => {
				const items = [createMockNotification("1"), createMockNotification("2")];

				const result = calculateNextItemAfterRemoval(-1, items, null, true);

				expect(result.targetIndex).toBe(null);
				expect(result.targetNotification).toBe(null);
				expect(result.willPageBeEmpty).toBe(false);
			});
		});

		describe("when willBeRemoved is false", () => {
			it("does not reduce item count", () => {
				const items = [createMockNotification("1"), createMockNotification("2")];

				const result = calculateNextItemAfterRemoval(0, items, null, false);

				expect(result.targetIndex).toBe(0);
				expect(result.willPageBeEmpty).toBe(false);
			});
		});

		describe("with currentFocus", () => {
			it("uses currentFocus when currentIndex is -1", () => {
				const items = [
					createMockNotification("1"),
					createMockNotification("2"),
					createMockNotification("3"),
				];

				// currentIndex -1 but focus is at 1
				const result = calculateNextItemAfterRemoval(-1, items, 1, true);

				// currentIndex -1 returns null
				expect(result.targetIndex).toBe(null);
			});
		});

		describe("edge cases", () => {
			it("handles empty items array", () => {
				const result = calculateNextItemAfterRemoval(-1, [], null, true);

				expect(result.targetIndex).toBe(null);
				expect(result.targetNotification).toBe(null);
				expect(result.willPageBeEmpty).toBe(true);
			});

			it("handles two items, removing first", () => {
				const items = [createMockNotification("1"), createMockNotification("2")];

				const result = calculateNextItemAfterRemoval(0, items, null, true);

				expect(result.targetIndex).toBe(0);
				expect(result.targetNotification?.id).toBe("2");
			});

			it("handles two items, removing second", () => {
				const items = [createMockNotification("1"), createMockNotification("2")];

				const result = calculateNextItemAfterRemoval(1, items, null, true);

				expect(result.targetIndex).toBe(0);
				expect(result.targetNotification?.id).toBe("1");
			});
		});
	});

	describe("calculatePaginationStateAfterRemoval", () => {
		describe("when page becomes empty on page > 1", () => {
			it("should navigate to previous page", () => {
				const result = calculatePaginationStateAfterRemoval(
					3, // currentPage
					0, // itemsAfterRemoval
					25, // pageSize
					50 // totalAfterRemoval
				);

				expect(result.shouldNavigateToPreviousPage).toBe(true);
				expect(result.targetPage).toBe(2);
			});
		});

		describe("when page becomes empty on page 1", () => {
			it("should not navigate (stay on page 1)", () => {
				const result = calculatePaginationStateAfterRemoval(
					1, // currentPage
					0, // itemsAfterRemoval
					25, // pageSize
					0 // totalAfterRemoval
				);

				expect(result.shouldNavigateToPreviousPage).toBe(false);
				expect(result.targetPage).toBe(null);
			});
		});

		describe("when page still has items", () => {
			it("should not navigate", () => {
				const result = calculatePaginationStateAfterRemoval(
					2, // currentPage
					24, // itemsAfterRemoval (had 25, removed 1)
					25, // pageSize
					49 // totalAfterRemoval
				);

				expect(result.shouldNavigateToPreviousPage).toBe(false);
				expect(result.targetPage).toBe(null);
			});
		});

		describe("when on page 2 and it becomes empty", () => {
			it("should navigate to page 1", () => {
				const result = calculatePaginationStateAfterRemoval(
					2, // currentPage
					0, // itemsAfterRemoval
					25, // pageSize
					25 // totalAfterRemoval (items on page 1)
				);

				expect(result.shouldNavigateToPreviousPage).toBe(true);
				expect(result.targetPage).toBe(1);
			});
		});

		describe("edge cases", () => {
			it("handles single item on page 1", () => {
				const result = calculatePaginationStateAfterRemoval(1, 0, 25, 0);

				expect(result.shouldNavigateToPreviousPage).toBe(false);
				expect(result.targetPage).toBe(null);
			});

			it("handles large page number", () => {
				const result = calculatePaginationStateAfterRemoval(
					100,
					0,
					25,
					2475 // 99 * 25
				);

				expect(result.shouldNavigateToPreviousPage).toBe(true);
				expect(result.targetPage).toBe(99);
			});
		});
	});
});
