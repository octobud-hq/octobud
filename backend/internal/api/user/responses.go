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

package user

// UserResponse represents the current user information
// Identity is based on GitHub - no local username/password
type UserResponse struct {
	GithubUserID   string                `json:"githubUserId,omitempty"`
	GithubUsername string                `json:"githubUsername,omitempty"`
	SyncSettings   *SyncSettingsResponse `json:"syncSettings,omitempty"`
}

// SyncSettingsResponse represents the user's sync settings
type SyncSettingsResponse struct {
	InitialSyncDays       *int `json:"initialSyncDays,omitempty"`
	InitialSyncMaxCount   *int `json:"initialSyncMaxCount,omitempty"`
	InitialSyncUnreadOnly bool `json:"initialSyncUnreadOnly"`
	SetupCompleted        bool `json:"setupCompleted"`
}

// SyncSettingsRequest represents the request to update sync settings
type SyncSettingsRequest struct {
	InitialSyncDays       *int `json:"initialSyncDays,omitempty"`
	InitialSyncMaxCount   *int `json:"initialSyncMaxCount,omitempty"`
	InitialSyncUnreadOnly bool `json:"initialSyncUnreadOnly"`
	SetupCompleted        bool `json:"setupCompleted"`
}

// SyncOlderRequest represents the request to sync older notifications
type SyncOlderRequest struct {
	Days       int     `json:"days"`                 // Required: number of days to sync back
	MaxCount   *int    `json:"maxCount,omitempty"`   // Optional: maximum notifications to sync
	UnreadOnly bool    `json:"unreadOnly,omitempty"` // Optional: only sync unread notifications
	BeforeDate *string `json:"beforeDate,omitempty"` // Optional: override the "before" date (RFC3339 format)
}

// SyncStateResponse represents the sync state information for the frontend
type SyncStateResponse struct {
	OldestNotificationSyncedAt *string `json:"oldestNotificationSyncedAt,omitempty"`
	InitialSyncCompletedAt     *string `json:"initialSyncCompletedAt,omitempty"`
}

// GitHubStatusResponse represents the current GitHub connection status
type GitHubStatusResponse struct {
	Connected      bool   `json:"connected"`
	Source         string `json:"source"` // "none", "stored", or "oauth"
	MaskedToken    string `json:"maskedToken,omitempty"`
	GitHubUsername string `json:"githubUsername,omitempty"`
}

// SetGitHubTokenRequest represents the request to set a GitHub token
type SetGitHubTokenRequest struct {
	Token string `json:"token"`
}

// SetGitHubTokenResponse represents the response after setting a token
type SetGitHubTokenResponse struct {
	GitHubUsername string `json:"githubUsername"`
}

// MuteStatusResponse represents the current mute status
type MuteStatusResponse struct {
	MutedUntil *string `json:"mutedUntil,omitempty"` // RFC3339 timestamp, null if not muted
	IsMuted    bool    `json:"isMuted"`
}

// SetMuteRequest represents the request to set mute
type SetMuteRequest struct {
	Duration string `json:"duration"` // "30m", "1h", or "rest_of_day"
}

// UpdateSettingsResponse represents the user's update settings
type UpdateSettingsResponse struct {
	Enabled            bool   `json:"enabled"`
	CheckFrequency     string `json:"checkFrequency"` // "on_startup", "daily", "weekly", "never"
	IncludePrereleases bool   `json:"includePrereleases"`
	LastCheckedAt      string `json:"lastCheckedAt,omitempty"`
	DismissedUntil     string `json:"dismissedUntil,omitempty"`
}

// UpdateSettingsRequest represents the request to update update settings
type UpdateSettingsRequest struct {
	Enabled            *bool   `json:"enabled,omitempty"`
	CheckFrequency     *string `json:"checkFrequency,omitempty"`
	IncludePrereleases *bool   `json:"includePrereleases,omitempty"`
	DismissedUntil     *string `json:"dismissedUntil,omitempty"`
}

// UpdateCheckResponse represents the response from checking for updates
type UpdateCheckResponse struct {
	UpdateAvailable bool   `json:"updateAvailable"`
	CurrentVersion  string `json:"currentVersion,omitempty"`
	LatestVersion   string `json:"latestVersion,omitempty"`
	ReleaseName     string `json:"releaseName,omitempty"`
	ReleaseNotes    string `json:"releaseNotes,omitempty"`
	PublishedAt     string `json:"publishedAt,omitempty"`
	DownloadURL     string `json:"downloadUrl,omitempty"`
	AssetName       string `json:"assetName,omitempty"`
	AssetSize       int64  `json:"assetSize,omitempty"`
	IsPrerelease    bool   `json:"isPrerelease,omitempty"`
}

// VersionResponse represents the current version of the application
type VersionResponse struct {
	Version string `json:"version"`
}
