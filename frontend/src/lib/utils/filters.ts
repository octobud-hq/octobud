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

export function normalizeQuickFilter(filter: NotificationViewFilter): NotificationViewFilter {
	const field = (filter.field ?? "").trim();
	const value = (filter.value ?? "").trim();
	const operator = (filter.operator ?? "equals") as NotificationViewFilter["operator"];

	return {
		field,
		operator,
		value,
	};
}

export function isQuickFilterActive(filter: NotificationViewFilter): boolean {
	return filter.field.length > 0 && filter.value.length > 0;
}
