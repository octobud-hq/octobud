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

import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { requestedIconNames } from "../scripts/octicon-names.mjs";
import subset from "../src/lib/octicons-subset";

/**
 * The build aliases "@primer/octicons" to the generated subset. If the vendored
 * notificationIcons.ts starts asking for an icon the subset doesn't carry, the
 * row silently renders an empty <svg> — no error, just a missing glyph. Catch it
 * here instead.
 */
describe("octicons subset", () => {
	// Resolved from the Vitest root rather than `import.meta.url`, which this file
	// does not reliably get as a file: URL once the .mjs import is transformed.
	const source = readFileSync(resolve("src/lib/utils/notificationIcons.ts"), "utf8");
	const requested = [...new Set(requestedIconNames(source))];

	it("finds the icon names the vendored module requests", () => {
		expect(requested.length).toBeGreaterThan(0);
	});

	it.each(requested)("includes a 16px path for %s", (name) => {
		expect(
			subset[name]?.heights["16"]?.path,
			`octicons-subset.ts is missing "${name}". Run \`npm run octicons\`.`
		).toBeTruthy();
	});

	it("carries nothing beyond what is requested", () => {
		expect(Object.keys(subset).sort()).toEqual([...requested].sort());
	});
});
