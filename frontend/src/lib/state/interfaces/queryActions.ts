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

import type { NotificationViewFilter } from "$lib/api/types";

/**
 * Query Actions
 * Actions related to search queries and filters
 */
export interface QueryActions {
	handleSearchInput: (value: string) => void;
	handleQuickQueryChange: (value: string) => void;
	addQuickFilter: () => void;
	updateQuickFilter: (index: number, patch: Partial<NotificationViewFilter>) => void;
	removeQuickFilter: (index: number) => void;
	clearFilters: () => Promise<void>;
}
