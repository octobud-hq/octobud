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

// Package update provides functionality for checking and managing application updates.
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
	// PlatformWindows is the platform identifier for Windows
	PlatformWindows = "windows"
	// PlatformLinux is the platform identifier for Linux
	PlatformLinux = "linux"
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
// Returns Info if a newer version is available, nil otherwise
func (s *Service) CheckForUpdates(
	ctx context.Context,
	includePrereleases bool,
) (*Info, error) {
	currentVersion := version.Get()

	// Query GitHub Releases API
	// If including prereleases, get all releases and filter; otherwise use /latest endpoint
	url := fmt.Sprintf("%s/%s/releases/latest", GitHubReleasesAPI, DefaultGitHubRepo)
	if includePrereleases {
		// For prereleases, get all releases and find the latest (including prereleases)
		url = fmt.Sprintf("%s/%s/releases", GitHubReleasesAPI, DefaultGitHubRepo)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to identify the application
	req.Header.Set("User-Agent", "Octobud/"+currentVersion)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.logger.Warn("failed to close response body", zap.Error(closeErr))
		}
	}()

	// Handle 404 for /releases/latest endpoint (no releases exist)
	if resp.StatusCode == http.StatusNotFound && !includePrereleases {
		s.logger.Info("No releases found in repository",
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
		if decodeErr := json.NewDecoder(resp.Body).Decode(&releases); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode releases: %w", decodeErr)
		}
		if len(releases) == 0 {
			s.logger.Info("No releases found in repository",
				zap.String("current", currentVersion),
			)
			return nil, nil
		}
		release = releases[0]
	} else {
		// Parse single latest release
		if decodeErr := json.NewDecoder(resp.Body).Decode(&release); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode release: %w", decodeErr)
		}
	}

	// Strip 'v' prefix from tag for comparison
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	// Normalize current version (strip 'v' prefix if present)
	normalizedCurrentVersion := strings.TrimPrefix(currentVersion, "v")

	// Log the comparison for debugging (Info level so it's visible)
	s.logger.Info("Comparing versions for update check",
		zap.String("current", currentVersion),
		zap.String("normalizedCurrent", normalizedCurrentVersion),
		zap.String("latest", latestVersion),
		zap.String("latestTag", release.TagName),
		zap.Bool("isPrerelease", release.Prerelease),
		zap.String("releaseName", release.Name),
	)

	// Compare versions
	comparisonResult := CompareVersions(latestVersion, normalizedCurrentVersion)
	if comparisonResult <= 0 {
		s.logger.Info("No update available - current version is up to date",
			zap.String("current", currentVersion),
			zap.String("normalizedCurrent", normalizedCurrentVersion),
			zap.String("latest", latestVersion),
			zap.String("latestTag", release.TagName),
			zap.Int("comparisonResult", comparisonResult),
		)
		return nil, nil
	}

	s.logger.Info("Update available",
		zap.String("current", currentVersion),
		zap.String("normalizedCurrent", normalizedCurrentVersion),
		zap.String("latest", latestVersion),
		zap.String("latestTag", release.TagName),
	)

	// Find appropriate asset for current platform
	asset, err := s.GetPlatformAsset(release.Assets)
	if err != nil {
		s.logger.Warn("No asset found for platform, but update is available",
			zap.Error(err),
			zap.String("latest", latestVersion),
		)
		// Still return update info even if no asset found
		return &Info{
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion,
			ReleaseName:    release.Name,
			ReleaseNotes:   release.Body,
			PublishedAt:    release.PublishedAt,
			IsPrerelease:   release.Prerelease,
		}, nil
	}

	return &Info{
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
	platform, arch := detectPlatform()
	if platform == "" {
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Try to find matching asset
	// Priority: platform-arch.pkg > platform.pkg > .pkg (for macOS)
	patterns := []string{
		fmt.Sprintf("%s-%s", platform, arch), // e.g., macos-arm64
		platform,                             // e.g., macos
		"",                                   // any .pkg
	}

	asset := s.findAssetByPatterns(assets, patterns, platform)
	if asset != nil {
		return asset, nil
	}

	// Fallback: return first asset if available
	if len(assets) > 0 {
		return &assets[0], nil
	}

	return nil, fmt.Errorf("no asset found for platform %s/%s", platform, arch)
}

// detectPlatform detects the current platform and architecture
func detectPlatform() (platform, arch string) {
	os := runtime.GOOS
	switch os {
	case "darwin":
		platform = "macos"
	case "windows":
		platform = PlatformWindows
	case "linux":
		platform = PlatformLinux
	default:
		return "", ""
	}

	arch = runtime.GOARCH
	// Normalize architecture names
	switch arch {
	case "arm64", "amd64", "386":
		// Already normalized
	default:
		// Keep as-is for other architectures
	}

	return platform, arch
}

// findAssetByPatterns finds an asset matching the given patterns and platform
func (s *Service) findAssetByPatterns(
	assets []ReleaseAsset,
	patterns []string,
	platform string,
) *ReleaseAsset {
	for _, pattern := range patterns {
		for _, asset := range assets {
			name := strings.ToLower(asset.Name)
			// Check if asset name contains pattern
			if pattern == "" || strings.Contains(name, pattern) {
				if s.matchesPlatformAsset(name, platform) {
					return &asset
				}
			}
		}
	}
	return nil
}

// matchesPlatformAsset checks if an asset name matches the expected format for the platform
func (s *Service) matchesPlatformAsset(name, platform string) bool {
	if platform == "macos" && strings.HasSuffix(name, ".pkg") {
		return true
	}
	if platform == PlatformWindows &&
		(strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".msi")) {
		return true
	}
	if platform == PlatformLinux &&
		(strings.HasSuffix(name, ".deb") || strings.HasSuffix(name, ".rpm") || strings.HasSuffix(name, ".AppImage")) {
		return true
	}
	return false
}
