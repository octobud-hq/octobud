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

package db

import (
	"database/sql/driver"
	"encoding/json"
)

// NullRawMessage represents a json.RawMessage that may be null.
// This replaces pqtype.NullRawMessage for SQLite-only operation.
type NullRawMessage struct {
	RawMessage json.RawMessage
	Valid      bool
}

// Scan implements the sql.Scanner interface.
func (n *NullRawMessage) Scan(value interface{}) error {
	if value == nil {
		n.RawMessage, n.Valid = nil, false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case []byte:
		n.RawMessage = json.RawMessage(v)
	case string:
		n.RawMessage = json.RawMessage(v)
	default:
		n.RawMessage = nil
	}
	return nil
}

// Value implements the driver.Valuer interface.
func (n NullRawMessage) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return []byte(n.RawMessage), nil
}

// MarshalJSON implements json.Marshaler.
func (n NullRawMessage) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.RawMessage, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *NullRawMessage) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		n.RawMessage = nil
		return nil
	}
	n.Valid = true
	n.RawMessage = json.RawMessage(data)
	return nil
}

// StorageStats contains notification counts by state for storage management
type StorageStats struct {
	TotalCount    int64
	ArchivedCount int64
	StarredCount  int64
	SnoozedCount  int64
	UnreadCount   int64
	TaggedCount   int64
}

// CleanupParams contains parameters for cleaning up old notifications
type CleanupParams struct {
	ProtectStarred bool
	ProtectTagged  bool
	CutoffDate     string // ISO8601 format
	BatchSize      int64
}
