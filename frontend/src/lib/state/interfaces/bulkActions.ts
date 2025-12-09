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
 * Bulk Operation Actions
 * Actions for performing operations on multiple notifications
 */
export interface BulkActions {
	bulkMarkRead: () => Promise<void>;
	bulkMarkUnread: () => Promise<void>;
	bulkArchive: () => Promise<void>;
	bulkUnarchive: () => Promise<void>;
	bulkMute: () => Promise<void>;
	bulkUnmute: () => Promise<void>;
	bulkSnooze: (snoozedUntil: string) => Promise<void>;
	bulkUnsnooze: () => Promise<void>;
	bulkStar: () => Promise<void>;
	bulkUnstar: () => Promise<void>;
	bulkUnfilter: () => Promise<void>;
	bulkAssignTag: (tagId: string) => Promise<void>;
	bulkRemoveTag: (tagId: string) => Promise<void>;
	markAllRead: () => Promise<void>;
	markAllUnread: () => Promise<void>;
	markAllArchive: () => Promise<void>;
}
