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
import type { NotificationTimelineItem } from "$lib/api/types";
import {
	groupConsecutiveEvents,
	isTimelineGroup,
	getDisplayItemTimestamp,
} from "./timelineGrouping";
import type { TimelineGroup, SummaryPart } from "./timelineGrouping";

/** Flatten summary parts to plain text for easy assertion. */
function summaryText(parts: SummaryPart[]): string {
	return parts.map((p) => (p.isMention ? `@${p.text}` : p.text)).join("");
}

function makeItem(
	type: string,
	timestamp: string,
	extra?: Partial<NotificationTimelineItem>
): NotificationTimelineItem {
	return {
		type,
		id: `${type}-${timestamp}`,
		timestamp,
		author: { login: "testuser", avatarUrl: "" },
		...extra,
	};
}

describe("groupConsecutiveEvents", () => {
	it("returns empty array for empty input", () => {
		expect(groupConsecutiveEvents([])).toEqual([]);
	});

	it("does not group a single bundleable item", () => {
		const items = [makeItem("labeled", "2025-01-01T00:00:00Z")];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(1);
		expect(isTimelineGroup(result[0])).toBe(false);
	});

	it("groups consecutive labeled events", () => {
		const items = [
			makeItem("labeled", "2025-01-01T00:01:00Z", {
				label: { name: "bug", color: "d73a4a" },
			}),
			makeItem("labeled", "2025-01-01T00:02:00Z", {
				label: { name: "enhancement", color: "a2eeef" },
			}),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(1);
		expect(isTimelineGroup(result[0])).toBe(true);
		const group = result[0] as TimelineGroup;
		expect(group.groupType).toBe("labeled");
		expect(group.items).toHaveLength(2);
		expect(summaryText(group.summary)).toBe("added labels ");
		expect(group.timestamp).toBe("2025-01-01T00:02:00Z");
	});

	it("does not group labeled and unlabeled together", () => {
		const items = [
			makeItem("labeled", "2025-01-01T00:01:00Z", {
				label: { name: "bug", color: "d73a4a" },
			}),
			makeItem("unlabeled", "2025-01-01T00:02:00Z", {
				label: { name: "wontfix", color: "ffffff" },
			}),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(2);
		expect(result.every((r) => !isTimelineGroup(r))).toBe(true);
	});

	it("groups consecutive assigned events", () => {
		const items = [
			makeItem("assigned", "2025-01-01T00:01:00Z", { assignee: "alice" }),
			makeItem("assigned", "2025-01-01T00:02:00Z", { assignee: "bob" }),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(1);
		expect(isTimelineGroup(result[0])).toBe(true);
		const group = result[0] as TimelineGroup;
		expect(group.groupType).toBe("assigned");
		expect(summaryText(group.summary)).toBe("assigned @alice and @bob");
	});

	it("falls back to count for assigned events without assignee field", () => {
		const items = [
			makeItem("assigned", "2025-01-01T00:01:00Z"),
			makeItem("assigned", "2025-01-01T00:02:00Z"),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(1);
		const group = result[0] as TimelineGroup;
		expect(summaryText(group.summary)).toBe("assigned 2");
	});

	it("groups consecutive review_requested events", () => {
		const items = [
			makeItem("review_requested", "2025-01-01T00:01:00Z", {
				requestedReviewer: "alice",
			}),
			makeItem("review_requested", "2025-01-01T00:02:00Z", {
				requestedReviewer: "bob",
			}),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(1);
		expect(isTimelineGroup(result[0])).toBe(true);
		const group = result[0] as TimelineGroup;
		expect(group.groupType).toBe("review_requested");
		expect(summaryText(group.summary)).toBe("requested reviews from @alice and @bob");
	});

	it("does not group non-consecutive events of same type", () => {
		const items = [
			makeItem("labeled", "2025-01-01T00:01:00Z"),
			makeItem("commented", "2025-01-01T00:02:00Z"),
			makeItem("labeled", "2025-01-01T00:03:00Z"),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(3);
		expect(result.every((r) => !isTimelineGroup(r))).toBe(true);
	});

	it("does not group events of different bundle categories", () => {
		const items = [
			makeItem("labeled", "2025-01-01T00:01:00Z"),
			makeItem("assigned", "2025-01-01T00:02:00Z"),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(2);
		expect(result.every((r) => !isTimelineGroup(r))).toBe(true);
	});

	it("handles mixed groups and ungrouped items", () => {
		const items = [
			makeItem("commented", "2025-01-01T00:01:00Z"),
			makeItem("labeled", "2025-01-01T00:02:00Z"),
			makeItem("labeled", "2025-01-01T00:03:00Z"),
			makeItem("merged", "2025-01-01T00:04:00Z"),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(3);
		expect(isTimelineGroup(result[0])).toBe(false); // commented
		expect(isTimelineGroup(result[1])).toBe(true); // label group
		expect(isTimelineGroup(result[2])).toBe(false); // merged
	});

	it("does not group consecutive events from different actors", () => {
		const items = [
			makeItem("labeled", "2025-01-01T00:01:00Z", {
				author: { login: "alice", avatarUrl: "" },
				label: { name: "bug", color: "d73a4a" },
			}),
			makeItem("labeled", "2025-01-01T00:02:00Z", {
				author: { login: "bob", avatarUrl: "" },
				label: { name: "enhancement", color: "a2eeef" },
			}),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(2);
		expect(result.every((r) => !isTimelineGroup(r))).toBe(true);
	});

	it("does not group non-bundleable events", () => {
		const items = [
			makeItem("committed", "2025-01-01T00:01:00Z"),
			makeItem("committed", "2025-01-01T00:02:00Z"),
			makeItem("head_ref_force_pushed", "2025-01-01T00:03:00Z"),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(3);
		expect(result.every((r) => !isTimelineGroup(r))).toBe(true);
	});
});

describe("getDisplayItemTimestamp", () => {
	it("returns timestamp from regular item", () => {
		const item = makeItem("commented", "2025-01-01T00:01:00Z");
		expect(getDisplayItemTimestamp(item)).toBe("2025-01-01T00:01:00Z");
	});

	it("returns newest timestamp from group", () => {
		const items = [
			makeItem("labeled", "2025-01-01T00:01:00Z"),
			makeItem("labeled", "2025-01-01T00:03:00Z"),
		];
		const result = groupConsecutiveEvents(items);
		expect(result).toHaveLength(1);
		expect(getDisplayItemTimestamp(result[0])).toBe("2025-01-01T00:03:00Z");
	});
});
