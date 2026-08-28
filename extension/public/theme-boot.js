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

// Applies the stored theme before first paint, mirroring the inline script in
// frontend/src/app.html. It has to be a classic synchronous script reading
// localStorage: storage.local is async, which is a guaranteed flash of the
// wrong theme every time the panel opens.
(function () {
	try {
		var theme = localStorage.getItem("octobud:theme");
		if (theme === "light" || theme === "dark") {
			document.documentElement.classList.toggle("dark", theme === "dark");
		} else {
			document.documentElement.classList.add("dark");
		}
	} catch (error) {
		void error;
		document.documentElement.classList.add("dark");
	}
})();
