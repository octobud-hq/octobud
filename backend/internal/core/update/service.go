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

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/version"
)

const (
	// DefaultGitHubRepo is the default GitHub repository for Octobud
	DefaultGitHubRepo = "octobud-hq/octobud"
	// GitHubReleasesAPI is the base URL for GitHub Releases API
	GitHubReleasesAPI = "https://api.github.com/repos"
)

// Service provides update checking functionality
type Service struct {
	httpClient *http.Client
	logger     *zap.Logger
	repoOwner  string
	repoName   string
}

// NewService creates a new update service
func NewService(logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:    logger,
		repoOwner: "octobud-hq",
		repoName:  "octobud",
	}
}

// CheckForUpdates checks for available updates by querying GitHub Releases API
// Returns UpdateInfo if a newer version is available, nil otherwise
func (s *Service) CheckForUpdates(ctx context.Context, includePrereleases bool) (*UpdateInfo, error) {
	currentVersion := version.Get()

	// Query GitHub Releases API
	// If including prereleases, get all releases and filter; otherwise use /latest endpoint
	url := fmt.Sprintf("%s/%s/releases/latest", GitHubReleasesAPI, DefaultGitHubRepo)
	if includePrereleases {
		// For prereleases, get all releases and find the latest (including prereleases)
		url = fmt.Sprintf("%s/%s/releases", GitHubReleasesAPI, DefaultGitHubRepo)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to identify the application
	req.Header.Set("User-Agent", "Octobud/"+currentVersion)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	// Handle 404 for /releases/latest endpoint (no releases exist)
	if resp.StatusCode == http.StatusNotFound && !includePrereleases {
		s.logger.Debug("No releases found in repository",
			zap.String("current", currentVersion),
		)
		return nil, nil // No releases available, not an error
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if includePrereleases {
		// Parse array of releases and get the first (latest) one
		var releases []GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, fmt.Errorf("failed to decode releases: %w", err)
		}
		if len(releases) == 0 {
			s.logger.Debug("No releases found in repository",
				zap.String("current", currentVersion),
			)
			return nil, nil
		}
		release = releases[0]
	} else {
		// Parse single latest release
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, fmt.Errorf("failed to decode release: %w", err)
		}
	}

	// Strip 'v' prefix from tag for comparison
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// Compare versions
	if !IsNewer(latestVersion, currentVersion) {
		s.logger.Debug("No update available",
			zap.String("current", currentVersion),
			zap.String("latest", latestVersion),
		)
		return nil, nil
	}

	// Find appropriate asset for current platform
	asset, err := s.GetPlatformAsset(release.Assets)
	if err != nil {
		s.logger.Warn("No asset found for platform, but update is available",
			zap.Error(err),
			zap.String("latest", latestVersion),
		)
		// Still return update info even if no asset found
		return &UpdateInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion,
			ReleaseName:    release.Name,
			ReleaseNotes:   release.Body,
			PublishedAt:    release.PublishedAt,
			IsPrerelease:   release.Prerelease,
		}, nil
	}

	return &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		ReleaseName:    release.Name,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
		DownloadURL:    asset.BrowserDownloadURL,
		AssetName:      asset.Name,
		AssetSize:      asset.Size,
		IsPrerelease:   release.Prerelease,
	}, nil
}

// GetPlatformAsset finds the appropriate release asset for the current platform
func (s *Service) GetPlatformAsset(assets []ReleaseAsset) (*ReleaseAsset, error) {
	var platform string
	var arch string

	// Detect platform and architecture
	os := runtime.GOOS
	switch os {
	case "darwin":
		platform = "macos"
	case "windows":
		platform = "windows"
	case "linux":
		platform = "linux"
	default:
		return nil, fmt.Errorf("unsupported platform: %s", os)
	}

	arch = runtime.GOARCH
	// Normalize architecture names
	if arch == "arm64" {
		arch = "arm64"
	} else if arch == "amd64" {
		arch = "amd64"
	} else if arch == "386" {
		arch = "386"
	}

	// Try to find matching asset
	// Priority: platform-arch.pkg > platform.pkg > .pkg (for macOS)
	patterns := []string{
		fmt.Sprintf("%s-%s", platform, arch), // e.g., macos-arm64
		platform,                             // e.g., macos
		"",                                   // any .pkg
	}

	for _, pattern := range patterns {
		for _, asset := range assets {
			name := strings.ToLower(asset.Name)
			// Check if asset name contains pattern
			if pattern == "" || strings.Contains(name, pattern) {
				// Prefer .pkg for macOS, .exe/.msi for Windows
				if platform == "macos" && strings.HasSuffix(name, ".pkg") {
					return &asset, nil
				}
				if platform == "windows" && (strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".msi")) {
					return &asset, nil
				}
				if platform == "linux" && (strings.HasSuffix(name, ".deb") || strings.HasSuffix(name, ".rpm") || strings.HasSuffix(name, ".AppImage")) {
					return &asset, nil
				}
			}
		}
	}

	// Fallback: return first asset if available
	if len(assets) > 0 {
		return &assets[0], nil
	}

	return nil, fmt.Errorf("no asset found for platform %s/%s", platform, arch)
}
