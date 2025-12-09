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

// Package models provides type helpers for the models package.
package models

import (
	"database/sql"
	"strings"
	"time"
)

// NullStringPtr converts sql.NullString to *string
func NullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

// NullInt64Ptr converts sql.NullInt64 to *int64
func NullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

// NullInt32ToInt64Ptr converts sql.NullInt32 to *int64 (converting int32 to int64)
func NullInt32ToInt64Ptr(value sql.NullInt32) *int64 {
	if !value.Valid {
		return nil
	}
	v := int64(value.Int32)
	return &v
}

// NullBoolPtr converts sql.NullBool to *bool
func NullBoolPtr(value sql.NullBool) *bool {
	if !value.Valid {
		return nil
	}
	v := value.Bool
	return &v
}

// NullTimePtr converts sql.NullTime to *time.Time
func NullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time
	return &v
}

// StringPtrToNull converts *string to sql.NullString
func StringPtrToNull(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{Valid: false}
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: v, Valid: true}
}

// IsUniqueViolation checks if an error is a unique constraint violation
// Works with SQLite UNIQUE constraint errors
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// SQLite unique constraint violations
	return strings.Contains(errStr, "UNIQUE constraint failed") ||
		strings.Contains(errStr, "SQLITE_CONSTRAINT_UNIQUE")
}

// StrPtr returns a pointer to the given string
func StrPtr(s string) *string {
	return &s
}

// SQLNullString converts string to sql.NullString (empty strings become invalid)
func SQLNullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

// SQLNullStringPtr converts *string to sql.NullString
func SQLNullStringPtr(value *string) sql.NullString {
	if value == nil || strings.TrimSpace(*value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

// SQLNullInt64 converts int64 to sql.NullInt64 (zero values become invalid)
func SQLNullInt64(value int64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

// SQLNullInt64Ptr converts *int64 to sql.NullInt64
func SQLNullInt64Ptr(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

// SQLNullBoolPtr converts *bool to sql.NullBool
func SQLNullBoolPtr(value *bool) sql.NullBool {
	if value == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *value, Valid: true}
}

// SQLNullTime converts *time.Time to sql.NullTime
func SQLNullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

// SQLNullTimeFromISO converts an ISO 8601 string to sql.NullTime
func SQLNullTimeFromISO(value *string) sql.NullTime {
	if value == nil || *value == "" {
		return sql.NullTime{}
	}
	t, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.UTC(), Valid: true}
}
