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

import eslintPluginSvelte from "eslint-plugin-svelte";
import prettier from "eslint-config-prettier";
import tsParser from "@typescript-eslint/parser";
import globals from "globals";

export default [
	{
		ignores: ["**/node_modules/**", "**/dist/**", "**/web-ext-artifacts/**"],
	},
	{
		files: ["**/*.{js,mjs,cjs}"],
		languageOptions: {
			ecmaVersion: "latest",
			sourceType: "module",
			globals: { ...globals.node, ...globals.browser, ...globals.webextensions },
		},
	},
	{
		files: ["**/*.ts"],
		languageOptions: {
			parser: tsParser,
			ecmaVersion: "latest",
			sourceType: "module",
			parserOptions: { project: "./tsconfig.json" },
			globals: { ...globals.browser, ...globals.webextensions },
		},
	},
	...eslintPluginSvelte.configs["flat/recommended"],
	{
		files: ["**/*.svelte"],
		languageOptions: {
			parserOptions: { parser: tsParser },
			globals: { ...globals.browser, ...globals.webextensions },
		},
		rules: {
			"svelte/no-at-html-tags": "warn",
			"svelte/valid-compile": "error",
		},
	},
	prettier,
].flat();
