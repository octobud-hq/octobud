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

// Bell off icon path extracted from bell_off.svg
// This shows a bell with a slash through it (for mute action when notification is unmuted)
export const bellOffIconPath =
	"M9 17V18C9 19.6569 10.3431 21 12 21C13.6569 21 15 19.6569 15 18V17M9 17L15 17M9 17L5.00022 17.0002C4.48738 17.0002 4.06449 16.6141 4.00673 16.1168L4 15.9998V15.4141C4 15.1489 4.10544 14.8949 4.29297 14.7073L4.80371 14.1963C4.92939 14.0706 5 13.9003 5 13.7225V9.99986C5 8.15821 5.7112 6.48267 6.87393 5.23291M15 17L18.9999 17M5 3L21 19M18.9995 12.999L19.0004 10C19.0004 6.13402 15.8665 3 12.0005 3L11.7598 3.00406C10.9548 3.03125 10.1845 3.19404 9.4707 3.47081";

// Bell on icon path extracted from bell_on.svg
// This is a regular bell (for unmute action when notification is muted)
export const bellOnIconPath =
	"M15 17V18C15 19.6569 13.6569 21 12 21C10.3431 21 9 19.6569 9 18V17M15 17H9M15 17H18.5905C18.973 17 19.1652 17 19.3201 16.9478C19.616 16.848 19.8475 16.6156 19.9473 16.3198C19.9997 16.1643 19.9997 15.9715 19.9997 15.5859C19.9997 15.4172 19.9995 15.3329 19.9863 15.2524C19.9614 15.1004 19.9024 14.9563 19.8126 14.8312C19.7651 14.7651 19.7048 14.7048 19.5858 14.5858L19.1963 14.1963C19.0706 14.0706 19 13.9001 19 13.7224V10C19 6.134 15.866 2.99999 12 3C8.13401 3.00001 5 6.13401 5 10V13.7224C5 13.9002 4.92924 14.0706 4.80357 14.1963L4.41406 14.5858C4.29476 14.7051 4.23504 14.765 4.1875 14.8312C4.09766 14.9564 4.03815 15.1004 4.0132 15.2524C4 15.3329 4 15.4172 4 15.586C4 15.9715 4 16.1642 4.05245 16.3197C4.15225 16.6156 4.3848 16.848 4.68066 16.9478C4.83556 17 5.02701 17 5.40956 17H9";

/**
 * Get the appropriate mute icon data based on whether the notification is muted
 * Returns an object with path string and whether to use stroke or fill
 */
export function getMuteIcon(isMuted: boolean): { path: string; useStroke: boolean } {
	if (isMuted) {
		// Notification is muted: show bell ON (unmute action) - uses stroke
		return { path: bellOnIconPath, useStroke: true };
	}
	// Notification is not muted: show bell OFF (mute action) - uses stroke
	return { path: bellOffIconPath, useStroke: true };
}
