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
 * Shared by the generator and its test. Kept free of any top-level filesystem or
 * `import.meta.url` work so it imports cleanly under Vitest's transform.
 */

/** Every icon name `utils/notificationIcons.ts` asks `getIconPath` for. */
export function requestedIconNames(source) {
	return [...source.matchAll(/getIconPath\("([^"]+)"\)/g)].map((match) => match[1]).sort();
}
