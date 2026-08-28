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

// `vitest/config` re-exports Vite's defineConfig with the `test` key typed.
import { defineConfig } from "vitest/config";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";

// The panel bundle is the only thing Vite builds. `public/background.js` is
// dependency-free vanilla JS and is copied through verbatim, which keeps the
// Chrome service worker and the Firefox event page on the same single file
// without asking Rollup to emit two module formats from one build.
export default defineConfig({
	plugins: [svelte(), tailwindcss()],
	resolve: {
		alias: {
			$lib: fileURLToPath(new URL("./src/lib", import.meta.url)),
		},
	},
	build: {
		outDir: "dist/build",
		emptyOutDir: true,
		// Extensions load from file:// style origins where long-term caching is
		// meaningless, and stable names make the packaged zip easier to diff.
		rollupOptions: {
			input: fileURLToPath(new URL("./panel.html", import.meta.url)),
			output: {
				entryFileNames: "assets/[name].js",
				chunkFileNames: "assets/[name].js",
				assetFileNames: "assets/[name].[ext]",
			},
		},
	},
	test: {
		include: ["test/**/*.{test,spec}.ts"],
		environment: "jsdom",
		globals: true,
		setupFiles: ["./test/setup.ts"],
	},
});
