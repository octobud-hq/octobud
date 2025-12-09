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

import { redirect } from "@sveltejs/kit";
import { fetchViews } from "$lib/api/views";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ fetch, url }) => {
	const views = await fetchViews(fetch);
	const defaultView = views.find((view) => view.isDefault) ?? null;

	// If no views are available, still redirect to inbox (system view)
	// The UI will show "no views" message in the sidebar
	const targetSlug = defaultView?.slug ?? views[0]?.slug ?? "inbox";

	const search = url.search;

	throw redirect(307, `/views/${encodeURIComponent(targetSlug)}${search}`);
};
