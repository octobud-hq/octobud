// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A GENERAL PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"database/sql"
	"encoding/json"
)

// User represents a user in the system
// Identity is based on GitHub - no local username/password
type User struct {
	ID                int64
	GithubUserID      string // GitHub user ID (the primary identity)
	GithubUsername    string // GitHub username (for display)
	SyncSettings      *SyncSettings
	RetentionSettings *RetentionSettings
	UpdateSettings    *UpdateSettings
	MutedUntil        sql.NullTime // When notifications are muted until (null if not muted)
}

// SyncSettings represents the user's sync configuration
type SyncSettings struct {
	InitialSyncDays       *int `json:"initialSyncDays,omitempty"`     // Number of days to sync (null = unlimited)
	InitialSyncMaxCount   *int `json:"initialSyncMaxCount,omitempty"` // Maximum notifications (null = no limit)
	InitialSyncUnreadOnly bool `json:"initialSyncUnreadOnly"`         // Only sync unread notifications initially
	SetupCompleted        bool `json:"setupCompleted"`                // Whether setup was completed
}

// ToJSON converts SyncSettings to JSON bytes
func (s *SyncSettings) ToJSON() (json.RawMessage, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// SyncSettingsFromJSON creates SyncSettings from JSON bytes
func SyncSettingsFromJSON(data json.RawMessage) (*SyncSettings, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var settings SyncSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// RetentionSettings represents the user's data retention configuration
type RetentionSettings struct {
	Enabled        bool   `json:"enabled"`                 // false = never auto-delete
	RetentionDays  int    `json:"retentionDays"`           // 30, 60, 90, 180, 365 (0 = never)
	ProtectStarred bool   `json:"protectStarred"`          // default: true
	ProtectTagged  bool   `json:"protectTagged"`           // default: true
	LastCleanupAt  string `json:"lastCleanupAt,omitempty"` // ISO8601 timestamp of last cleanup
}

// DefaultRetentionSettings returns the default retention settings (auto-cleanup disabled)
func DefaultRetentionSettings() *RetentionSettings {
	return &RetentionSettings{
		Enabled:        false,
		RetentionDays:  90,
		ProtectStarred: true,
		ProtectTagged:  true,
	}
}

// ToJSON converts RetentionSettings to JSON bytes
func (s *RetentionSettings) ToJSON() (json.RawMessage, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// RetentionSettingsFromJSON creates RetentionSettings from JSON bytes
func RetentionSettingsFromJSON(data json.RawMessage) (*RetentionSettings, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var settings RetentionSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateSettings represents the user's auto-update configuration
type UpdateSettings struct {
	Enabled            bool   `json:"enabled"`                  // Enable/disable auto-update checking (default: true)
	CheckFrequency     string `json:"checkFrequency"`           // "on_startup", "daily", "weekly", "never" (default: "daily")
	IncludePrereleases bool   `json:"includePrereleases"`       // Include beta/RC releases (default: false)
	LastCheckedAt      string `json:"lastCheckedAt,omitempty"`  // ISO8601 timestamp of last check
	DismissedUntil     string `json:"dismissedUntil,omitempty"` // ISO8601 timestamp - when to stop showing banner (if dismissed)
}

// DefaultUpdateSettings returns the default update settings (auto-update enabled)
func DefaultUpdateSettings() *UpdateSettings {
	return &UpdateSettings{
		Enabled:            true,
		CheckFrequency:     "daily",
		IncludePrereleases: false,
	}
}

// ToJSON converts UpdateSettings to JSON bytes
func (s *UpdateSettings) ToJSON() (json.RawMessage, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// UpdateSettingsFromJSON creates UpdateSettings from JSON bytes
func UpdateSettingsFromJSON(data json.RawMessage) (*UpdateSettings, error) {
	if len(data) == 0 {
		return DefaultUpdateSettings(), nil // Return default if no settings found
	}
	var settings UpdateSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}
