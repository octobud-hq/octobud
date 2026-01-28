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
import { getActionIconConfig, getActionLabel } from "./actionIcons";
import type { UndoableActionType } from "./types";

describe("actionIcons", () => {
	describe("getActionIconConfig", () => {
		const actionTypes: UndoableActionType[] = [
			"archive",
			"unarchive",
			"mute",
			"unmute",
			"star",
			"unstar",
			"markRead",
			"markUnread",
			"snooze",
			"unsnooze",
			"assignTag",
			"removeTag",
		];

		it("returns valid icon config for all action types", () => {
			for (const actionType of actionTypes) {
				const config = getActionIconConfig(actionType);

				expect(config).toBeDefined();
				expect(config.paths).toBeDefined();
				expect(Array.isArray(config.paths)).toBe(true);
				expect(config.paths.length).toBeGreaterThan(0);
				expect(config.viewBox).toBeDefined();
				expect(typeof config.useStroke).toBe("boolean");
			}
		});

		it("returns stroke-based icons for archive actions", () => {
			const archiveConfig = getActionIconConfig("archive");
			expect(archiveConfig.useStroke).toBe(true);
			expect(archiveConfig.strokeWidth).toBeDefined();
		});

		it("returns fill-based icons for unarchive", () => {
			const unarchiveConfig = getActionIconConfig("unarchive");
			expect(unarchiveConfig.useStroke).toBe(false);
		});

		it("returns stroke-based icons for mute/unmute", () => {
			expect(getActionIconConfig("mute").useStroke).toBe(true);
			expect(getActionIconConfig("unmute").useStroke).toBe(true);
		});

		it("returns stroke-based icons for star/unstar", () => {
			expect(getActionIconConfig("star").useStroke).toBe(true);
			expect(getActionIconConfig("unstar").useStroke).toBe(true);
		});

		it("returns fill-based icons for mail (read/unread)", () => {
			expect(getActionIconConfig("markRead").useStroke).toBe(false);
			expect(getActionIconConfig("markUnread").useStroke).toBe(false);
		});

		it("returns stroke-based icons for snooze", () => {
			expect(getActionIconConfig("snooze").useStroke).toBe(true);
			expect(getActionIconConfig("unsnooze").useStroke).toBe(true);
		});

		it("returns fill-based icons for tags", () => {
			expect(getActionIconConfig("assignTag").useStroke).toBe(false);
			expect(getActionIconConfig("removeTag").useStroke).toBe(false);
		});
	});

	describe("getActionLabel", () => {
		it("returns correct labels for all action types", () => {
			expect(getActionLabel("archive")).toBe("Archived");
			expect(getActionLabel("unarchive")).toBe("Unarchived");
			expect(getActionLabel("mute")).toBe("Muted");
			expect(getActionLabel("unmute")).toBe("Unmuted");
			expect(getActionLabel("star")).toBe("Starred");
			expect(getActionLabel("unstar")).toBe("Unstarred");
			expect(getActionLabel("markRead")).toBe("Marked read");
			expect(getActionLabel("markUnread")).toBe("Marked unread");
			expect(getActionLabel("snooze")).toBe("Snoozed");
			expect(getActionLabel("unsnooze")).toBe("Unsnoozed");
			expect(getActionLabel("assignTag")).toBe("Added tag");
			expect(getActionLabel("removeTag")).toBe("Removed tag");
		});

		it("returns fallback for unknown action types", () => {
			// Cast to bypass TypeScript check for testing purposes
			expect(getActionLabel("unknownAction" as UndoableActionType)).toBe("Action");
		});
	});
});
