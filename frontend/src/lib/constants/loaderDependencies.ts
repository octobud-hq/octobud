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
 * SvelteKit load dependency keys. (Route module files may only export load-related
 * symbols, so these live here rather than next to the loaders that declare them.)
 */

/**
 * The sidebar data (views and tags with their unread counts), owned by the root layout
 * loader. Anything that creates, edits, or deletes a view or tag, or changes a view's
 * default repositories, must `invalidate(VIEWS_DEPENDENCY)` so the loaders see fresh data.
 */
export const VIEWS_DEPENDENCY = "app:views";
