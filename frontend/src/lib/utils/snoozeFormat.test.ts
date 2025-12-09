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

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { formatSnoozeMessage } from "./snoozeFormat";

describe("formatSnoozeMessage", () => {
	beforeEach(() => {
		vi.useFakeTimers();
		// Set current time to Monday, January 15, 2024 at 12:00 PM
		vi.setSystemTime(new Date("2024-01-15T12:00:00"));
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	describe("past snooze dates", () => {
		it('shows "Snoozed just now" for very recent snoozes', () => {
			const snoozedUntil = "2024-01-15T11:59:30";
			const snoozedAt = "2024-01-15T11:59:55";
			expect(formatSnoozeMessage(snoozedUntil, snoozedAt)).toBe("Snoozed just now");
		});

		it("shows minutes ago for recent snoozes", () => {
			const snoozedUntil = "2024-01-15T11:30:00";
			const snoozedAt = "2024-01-15T11:30:00";
			expect(formatSnoozeMessage(snoozedUntil, snoozedAt)).toBe("Snoozed 30m ago");
		});

		it("shows hours ago for older snoozes", () => {
			const snoozedUntil = "2024-01-15T09:00:00";
			const snoozedAt = "2024-01-15T09:00:00";
			expect(formatSnoozeMessage(snoozedUntil, snoozedAt)).toBe("Snoozed 3h ago");
		});

		it("shows days ago for snoozes more than a day old", () => {
			const snoozedUntil = "2024-01-13T12:00:00";
			const snoozedAt = "2024-01-13T12:00:00";
			expect(formatSnoozeMessage(snoozedUntil, snoozedAt)).toBe("Snoozed 2d ago");
		});
	});

	describe("future snooze dates", () => {
		it("shows time for today", () => {
			const snoozedUntil = "2024-01-15T18:00:00";
			expect(formatSnoozeMessage(snoozedUntil)).toBe("Snoozed until 6pm");
		});

		it("shows time with minutes for non-round hours", () => {
			const snoozedUntil = "2024-01-15T18:30:00";
			expect(formatSnoozeMessage(snoozedUntil)).toBe("Snoozed until 6:30pm");
		});

		it('shows "tomorrow" for tomorrow', () => {
			const snoozedUntil = "2024-01-16T09:00:00";
			expect(formatSnoozeMessage(snoozedUntil)).toBe("Snoozed until tomorrow");
		});

		it("shows day of week for dates within a week", () => {
			// Wednesday, January 17, 2024
			const snoozedUntil = "2024-01-17T09:00:00";
			expect(formatSnoozeMessage(snoozedUntil)).toBe("Snoozed until Wednesday");
		});

		it("shows date for dates more than a week away", () => {
			const snoozedUntil = "2024-01-25T09:00:00";
			expect(formatSnoozeMessage(snoozedUntil)).toBe("Snoozed until Jan 25");
		});
	});
});
