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

import type { PageLoad } from "./$types";
import { fetchRules } from "$lib/api/rules";
import { fetchTags } from "$lib/api/tags";
import { ApiError } from "$lib/api/fetch";

// Swallow 401 (user disconnected) so Settings stays reachable for reconnect.
// Re-throw anything else so real backend errors still surface to the user.
function swallowUnauthorized<T>(fallback: T): (err: unknown) => T {
	return (err: unknown) => {
		if (err instanceof ApiError && err.status === 401) {
			return fallback;
		}
		throw err;
	};
}

export const load: PageLoad = async ({ fetch }) => {
	const [rules, tags] = await Promise.all([
		fetchRules(fetch).catch(swallowUnauthorized<Awaited<ReturnType<typeof fetchRules>>>([])),
		fetchTags(fetch).catch(swallowUnauthorized<Awaited<ReturnType<typeof fetchTags>>>([])),
	]);
	return { rules, tags };
};
