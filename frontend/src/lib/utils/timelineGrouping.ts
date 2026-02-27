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

export type SummaryPart = { text: string; isMention?: boolean };

export type GroupType =
	| "labeled"
	| "unlabeled"
	| "assigned"
	| "unassigned"
	| "review_requested"
	| "review_request_removed";

export interface TimelineGroup {
	_isGroup: true;
	groupType: GroupType;
	items: NotificationTimelineItem[];
	summary: SummaryPart[];
	timestamp: string;
	actor?: { login: string; avatarUrl: string };
}

export type TimelineDisplayItem = NotificationTimelineItem | TimelineGroup;

export function isTimelineGroup(item: TimelineDisplayItem): item is TimelineGroup {
	return "_isGroup" in item && item._isGroup === true;
}

const bundleableTypes: Set<string> = new Set([
	"labeled",
	"unlabeled",
	"assigned",
	"unassigned",
	"review_requested",
	"review_request_removed",
]);

function getItemTimestamp(item: NotificationTimelineItem): string | undefined {
	return item.timestamp || item.createdAt || item.submittedAt || item.updatedAt;
}

/** Resolve the actor from either author or actor field (backend sends author). */
function getItemActor(
	item: NotificationTimelineItem
): { login: string; avatarUrl: string } | undefined {
	return item.author || item.actor;
}

function mentionList(names: string[]): SummaryPart[] {
	if (names.length === 0) return [];
	const parts: SummaryPart[] = [{ text: names[0], isMention: true }];
	for (let i = 1; i < names.length; i++) {
		parts.push({ text: i === names.length - 1 ? " and " : ", " });
		parts.push({ text: names[i], isMention: true });
	}
	return parts;
}

function getGroupSummary(group: TimelineGroup): SummaryPart[] {
	const { items, groupType } = group;
	switch (groupType) {
		case "labeled":
			return [{ text: `added label${items.length > 1 ? "s" : ""} ` }];
		case "unlabeled":
			return [{ text: `removed label${items.length > 1 ? "s" : ""} ` }];
		case "assigned": {
			const names = items.map((i) => i.assignee).filter(Boolean) as string[];
			if (names.length > 0) {
				return [{ text: "assigned " }, ...mentionList(names)];
			}
			return [{ text: `assigned ${items.length}` }];
		}
		case "unassigned": {
			const names = items.map((i) => i.assignee).filter(Boolean) as string[];
			if (names.length > 0) {
				return [{ text: "unassigned " }, ...mentionList(names)];
			}
			return [{ text: `unassigned ${items.length}` }];
		}
		case "review_requested": {
			const names = items.map((i) => i.requestedReviewer).filter(Boolean) as string[];
			if (names.length > 0) {
				return [
					{ text: `requested review${items.length > 1 ? "s" : ""} from ` },
					...mentionList(names),
				];
			}
			return [{ text: `requested ${items.length} review${items.length > 1 ? "s" : ""}` }];
		}
		case "review_request_removed": {
			const names = items.map((i) => i.requestedReviewer).filter(Boolean) as string[];
			if (names.length > 0) {
				return [
					{ text: `removed review request${items.length > 1 ? "s" : ""} from ` },
					...mentionList(names),
				];
			}
			return [{ text: `removed ${items.length} review request${items.length > 1 ? "s" : ""}` }];
		}
		default:
			return [{ text: `${items.length} events` }];
	}
}

export function groupConsecutiveEvents(items: NotificationTimelineItem[]): TimelineDisplayItem[] {
	if (items.length === 0) return [];

	const result: TimelineDisplayItem[] = [];
	let currentGroupItems: NotificationTimelineItem[] = [];
	let currentType: string | null = null;

	function flushGroup() {
		if (currentGroupItems.length === 0) return;
		if (currentGroupItems.length === 1) {
			result.push(currentGroupItems[0]);
		} else {
			const newestTimestamp =
				getItemTimestamp(currentGroupItems[currentGroupItems.length - 1]) || "";
			const actor = getItemActor(currentGroupItems[0]);
			const group: TimelineGroup = {
				_isGroup: true,
				groupType: currentType as GroupType,
				items: currentGroupItems,
				summary: [],
				timestamp: newestTimestamp,
				actor,
			};
			group.summary = getGroupSummary(group);
			result.push(group);
		}
		currentGroupItems = [];
		currentType = null;
	}

	for (const item of items) {
		if (bundleableTypes.has(item.type)) {
			const actorLogin = getItemActor(item)?.login;
			const currentActorLogin =
				currentGroupItems.length > 0 ? getItemActor(currentGroupItems[0])?.login : undefined;
			if (currentType === item.type && actorLogin === currentActorLogin) {
				currentGroupItems.push(item);
			} else {
				flushGroup();
				currentGroupItems = [item];
				currentType = item.type;
			}
		} else {
			flushGroup();
			result.push(item);
		}
	}
	flushGroup();

	return result;
}

export function getDisplayItemTimestamp(item: TimelineDisplayItem): string | undefined {
	if (isTimelineGroup(item)) {
		return item.timestamp;
	}
	return getItemTimestamp(item);
}
