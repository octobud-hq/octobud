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
 * Generates src/lib/octicons-subset.ts.
 *
 * `@primer/octicons` is a 4 MB package whose default export is one big object,
 * so importing it pulls every icon into the bundle — about a megabyte of JS for
 * the twelve 16px paths the notification row actually draws. The panel is
 * re-parsed every time the sidebar opens, so that waste is worth removing.
 *
 * The icon names are scraped out of utils/notificationIcons.ts rather than
 * hard-coded here, so the subset cannot drift from what that file requests. It
 * is vendored byte-identically from frontend/ (see src/lib/VENDORED.md), which
 * is why the trimming happens behind a build alias instead of as an edit to it.
 */

import octicons from "@primer/octicons";
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath, pathToFileURL } from "node:url";

import { requestedIconNames } from "./octicon-names.mjs";

const SOURCE = fileURLToPath(new URL("../src/lib/utils/notificationIcons.ts", import.meta.url));
const OUTPUT = fileURLToPath(new URL("../src/lib/octicons-subset.ts", import.meta.url));

function generate() {
	const names = [...new Set(requestedIconNames(readFileSync(SOURCE, "utf8")))];
	if (names.length === 0) {
		throw new Error(
			`No getIconPath("…") calls found in ${SOURCE}; refusing to emit an empty subset.`
		);
	}

	const missing = names.filter((name) => !octicons[name]?.heights?.["16"]);
	if (missing.length > 0) {
		throw new Error(`@primer/octicons has no 16px variant for: ${missing.join(", ")}`);
	}

	const entries = names
		.map((name) => {
			const variant = octicons[name].heights["16"];
			// Octicons keys each variant by its height and omits the `height` field
			// itself, so take it from the key.
			const width = variant.width ?? 16;
			const height = variant.height ?? 16;
			return `\t"${name}": {\n\t\tname: "${name}",\n\t\tkeywords: [],\n\t\theights: { "16": { width: ${width}, height: ${height}, path: ${JSON.stringify(variant.path)} } },\n\t},`;
		})
		.join("\n");

	writeFileSync(OUTPUT, render(entries), "utf8");
	console.log(`Wrote ${names.length} icons to src/lib/octicons-subset.ts: ${names.join(", ")}`);
}

function render(entries) {
	return `// Copyright (C) 2025 Austin Beattie
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

// GENERATED FILE — do not edit. Run \`npm run octicons\` to regenerate.
//
// The 16px octicons that utils/notificationIcons.ts draws. Vite aliases
// "@primer/octicons" to this module so the panel bundles ~4 KB of icon data
// instead of the full 4 MB package. See scripts/build-octicons-subset.mjs.

interface OcticonHeight {
\twidth: number;
\theight: number;
\tpath: string;
}

interface Octicon {
\tname: string;
\tkeywords: string[];
\theights: Record<string, OcticonHeight>;
}

const octicons: Record<string, Octicon> = {
${entries}
};

export default octicons;
`;
}

// Only generate when run directly, so the test can import `requestedIconNames`
// without rewriting the committed file as a side effect.
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
	generate();
}
