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

import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

export default {
	preprocess: vitePreprocess(),
	compilerOptions: {
		// The rest of the codebase is written in Svelte 4 syntax (`export let`,
		// `$:`, `on:click`). Runes mode would reject it, so stay in legacy mode
		// and keep ported components diffable against frontend/.
		runes: false,
	},
};
