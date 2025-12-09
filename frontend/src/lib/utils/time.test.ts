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
import { parseTimestamp, formatRelativeShort } from "./time";

describe("parseTimestamp", () => {
	it("returns null for null input", () => {
		expect(parseTimestamp(null)).toBeNull();
	});

	it("returns null for undefined input", () => {
		expect(parseTimestamp(undefined)).toBeNull();
	});

	it("returns null for empty string", () => {
		expect(parseTimestamp("")).toBeNull();
	});

	it("returns null for invalid date string", () => {
		expect(parseTimestamp("not-a-date")).toBeNull();
	});

	it("parses valid ISO date string", () => {
		const result = parseTimestamp("2024-01-15T10:30:00Z");
		expect(result).toBeInstanceOf(Date);
		expect(result?.toISOString()).toBe("2024-01-15T10:30:00.000Z");
	});

	it("parses date string without time", () => {
		const result = parseTimestamp("2024-01-15");
		expect(result).toBeInstanceOf(Date);
	});
});

describe("formatRelativeShort", () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date("2024-01-15T12:00:00Z"));
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it("formats seconds ago", () => {
		const date = new Date("2024-01-15T11:59:30Z");
		expect(formatRelativeShort(date)).toBe("30s ago");
	});

	it("formats minutes ago", () => {
		const date = new Date("2024-01-15T11:55:00Z");
		expect(formatRelativeShort(date)).toBe("5m ago");
	});

	it("formats hours ago", () => {
		const date = new Date("2024-01-15T09:00:00Z");
		expect(formatRelativeShort(date)).toBe("3h ago");
	});

	it("formats days ago", () => {
		const date = new Date("2024-01-13T12:00:00Z");
		expect(formatRelativeShort(date)).toBe("2d ago");
	});

	it("formats weeks ago", () => {
		const date = new Date("2024-01-01T12:00:00Z");
		expect(formatRelativeShort(date)).toBe("2w ago");
	});

	it("formats future dates", () => {
		const date = new Date("2024-01-15T13:00:00Z");
		expect(formatRelativeShort(date)).toBe("1h from now");
	});
});
