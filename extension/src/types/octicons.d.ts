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
 * `@primer/octicons` ships no type declarations. `utils/notificationIcons.ts` is
 * vendored byte-identically from frontend/, so the shim goes here rather than as
 * an edit to the copy.
 *
 * Only the shape that file actually uses is declared: an icon registry keyed by
 * name, each entry exposing SVG markup per height.
 */
declare module "@primer/octicons" {
	interface OcticonHeight {
		width: number;
		height: number;
		path: string;
	}

	interface Octicon {
		name: string;
		keywords: string[];
		heights: Record<string, OcticonHeight>;
	}

	const octicons: Record<string, Octicon>;
	export default octicons;
}
