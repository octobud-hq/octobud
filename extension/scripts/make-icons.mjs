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
 * Regenerates public/icons/* from the app's own favicon, so the toolbar button
 * is recognisably the same mark as the desktop app rather than a lookalike.
 *
 * Committed output — this only needs re-running if frontend/static/favicon.png
 * changes. Requires ImageMagick, which is why it is not wired into `npm run
 * build`: a missing system dependency should not break an ordinary build.
 */

import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";

const SOURCE = fileURLToPath(new URL("../../frontend/static/favicon.png", import.meta.url));
const OUT_DIR = fileURLToPath(new URL("../public/icons", import.meta.url));
const SIZES = [16, 32, 48, 128];

if (!existsSync(SOURCE)) {
	throw new Error(`Source icon not found at ${SOURCE}`);
}

mkdirSync(OUT_DIR, { recursive: true });

for (const size of SIZES) {
	const output = `${OUT_DIR}/icon-${size}.png`;
	// The favicon is 404x402; pad to a square first so the resize doesn't
	// distort, keeping the transparent background.
	execFileSync("convert", [
		SOURCE,
		"-background",
		"none",
		"-gravity",
		"center",
		"-extent",
		"404x404",
		"-resize",
		`${size}x${size}`,
		output,
	]);
	console.log(`Wrote ${output}`);
}
