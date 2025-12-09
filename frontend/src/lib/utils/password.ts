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

// Password validation constants - must match backend/internal/core/auth/password.go
export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_LENGTH = 128;

export interface PasswordValidationResult {
	valid: boolean;
	error?: string;
}

/**
 * Validates password strength
 * Returns validation result with error message if invalid
 */
export function validatePasswordStrength(password: string): PasswordValidationResult {
	const length = [...password].length; // Count Unicode characters, not bytes

	if (length < MIN_PASSWORD_LENGTH) {
		return {
			valid: false,
			error: `Password must be at least ${MIN_PASSWORD_LENGTH} characters long (got ${length})`,
		};
	}

	if (length > MAX_PASSWORD_LENGTH) {
		return {
			valid: false,
			error: `Password must be no more than ${MAX_PASSWORD_LENGTH} characters long (got ${length})`,
		};
	}

	return { valid: true };
}
