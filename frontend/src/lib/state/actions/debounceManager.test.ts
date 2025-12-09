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
import { createDebounceManager, SEARCH_DEBOUNCE_MS } from "./debounceManager";

describe("DebounceManager", () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	describe("setSearchDebounce", () => {
		it("calls function after debounce delay", () => {
			const manager = createDebounceManager();
			const fn = vi.fn();

			manager.setSearchDebounce(fn);

			expect(fn).not.toHaveBeenCalled();

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(fn).toHaveBeenCalledTimes(1);
		});

		it("cancels previous debounce when called again", () => {
			const manager = createDebounceManager();
			const fn1 = vi.fn();
			const fn2 = vi.fn();

			manager.setSearchDebounce(fn1);
			vi.advanceTimersByTime(100); // Partial delay
			manager.setSearchDebounce(fn2);

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(fn1).not.toHaveBeenCalled();
			expect(fn2).toHaveBeenCalledTimes(1);
		});

		it("resets timer when called multiple times", () => {
			const manager = createDebounceManager();
			const fn = vi.fn();

			manager.setSearchDebounce(fn);
			vi.advanceTimersByTime(200);
			manager.setSearchDebounce(fn);
			vi.advanceTimersByTime(200);
			manager.setSearchDebounce(fn);

			expect(fn).not.toHaveBeenCalled();

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(fn).toHaveBeenCalledTimes(1);
		});
	});

	describe("setQueryDebounce", () => {
		it("calls function after debounce delay", () => {
			const manager = createDebounceManager();
			const fn = vi.fn();

			manager.setQueryDebounce(fn);

			expect(fn).not.toHaveBeenCalled();

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(fn).toHaveBeenCalledTimes(1);
		});

		it("cancels previous debounce when called again", () => {
			const manager = createDebounceManager();
			const fn1 = vi.fn();
			const fn2 = vi.fn();

			manager.setQueryDebounce(fn1);
			vi.advanceTimersByTime(100);
			manager.setQueryDebounce(fn2);

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(fn1).not.toHaveBeenCalled();
			expect(fn2).toHaveBeenCalledTimes(1);
		});
	});

	describe("search and query debounces are independent", () => {
		it("does not interfere with each other", () => {
			const manager = createDebounceManager();
			const searchFn = vi.fn();
			const queryFn = vi.fn();

			manager.setSearchDebounce(searchFn);
			manager.setQueryDebounce(queryFn);

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(searchFn).toHaveBeenCalledTimes(1);
			expect(queryFn).toHaveBeenCalledTimes(1);
		});

		it("canceling search does not affect query", () => {
			const manager = createDebounceManager();
			const searchFn1 = vi.fn();
			const searchFn2 = vi.fn();
			const queryFn = vi.fn();

			manager.setSearchDebounce(searchFn1);
			manager.setQueryDebounce(queryFn);
			vi.advanceTimersByTime(100);
			manager.setSearchDebounce(searchFn2); // Cancel searchFn1

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(searchFn1).not.toHaveBeenCalled();
			expect(searchFn2).toHaveBeenCalledTimes(1);
			expect(queryFn).toHaveBeenCalledTimes(1);
		});
	});

	describe("clearAll", () => {
		it("cancels both search and query debounces", () => {
			const manager = createDebounceManager();
			const searchFn = vi.fn();
			const queryFn = vi.fn();

			manager.setSearchDebounce(searchFn);
			manager.setQueryDebounce(queryFn);

			manager.clearAll();

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(searchFn).not.toHaveBeenCalled();
			expect(queryFn).not.toHaveBeenCalled();
		});

		it("handles clearAll when no debounces are pending", () => {
			const manager = createDebounceManager();

			expect(() => manager.clearAll()).not.toThrow();
		});

		it("can set new debounces after clearAll", () => {
			const manager = createDebounceManager();
			const fn1 = vi.fn();
			const fn2 = vi.fn();

			manager.setSearchDebounce(fn1);
			manager.clearAll();
			manager.setSearchDebounce(fn2);

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(fn1).not.toHaveBeenCalled();
			expect(fn2).toHaveBeenCalledTimes(1);
		});
	});

	describe("multiple manager instances", () => {
		it("are independent", () => {
			const manager1 = createDebounceManager();
			const manager2 = createDebounceManager();
			const fn1 = vi.fn();
			const fn2 = vi.fn();

			manager1.setSearchDebounce(fn1);
			manager2.setSearchDebounce(fn2);

			manager1.clearAll();

			vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS);

			expect(fn1).not.toHaveBeenCalled();
			expect(fn2).toHaveBeenCalledTimes(1);
		});
	});
});
