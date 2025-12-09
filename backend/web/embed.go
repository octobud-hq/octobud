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

// Package web embeds the frontend static files for production deployment.
//
// The dist/ directory is populated by running `npm run build` in the frontend
// directory, then copying the output to backend/web/dist/.
//
// Use the build-frontend target: make build-frontend
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns a filesystem containing the built frontend assets.
// The assets are rooted at "dist/", so callers typically use fs.Sub to strip that prefix.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
