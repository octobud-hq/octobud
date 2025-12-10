//go:build darwin

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

package osactions

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// appPath is the path to the installed Octobud application on macOS.
const appPath = "/Applications/Octobud.app"

// service is the macOS implementation using AppleScript.
type service struct{}

// NewService creates a new OS actions service for macOS.
func NewService() Service {
	return &service{}
}

// RestartApp quits the current application and launches the installed app.
// The app's startup code will handle checking for existing tabs.
func (s *service) RestartApp(ctx context.Context, baseURL string) error {
	// Simple restart - the app will check for existing tabs on startup
	script := fmt.Sprintf(`tell application "Octobud"
	quit
end tell
delay 1
tell application "%s"
	activate
end tell`, appPath)

	// Run in background so HTTP response can return
	//nolint:gosec // G204: osascript is a standard macOS utility, the script variable is controlled
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start restart script: %w", err)
	}

	// Don't wait for completion - let it run in background
	// The script will handle the quit and relaunch
	return nil
}

// ActivateBrowserTab attempts to find and activate a browser tab with the given URL.
// If navigateTo is provided and a tab is found, it will navigate that tab to navigateTo.
func (s *service) ActivateBrowserTab(
	ctx context.Context,
	browser string,
	url string,
	navigateTo string,
) (bool, error) {
	switch strings.ToLower(browser) {
	case "safari":
		return s.activateSafariTab(ctx, url, navigateTo)
	case "chrome":
		return s.activateChromeTab(ctx, url, navigateTo)
	default:
		return false, fmt.Errorf("unsupported browser: %s", browser)
	}
}

// activateSafariTab uses AppleScript to find and activate any Safari tab with the URL.
func (s *service) activateSafariTab(
	ctx context.Context,
	url,
	navigateTo string,
) (bool, error) {
	// Escape the URL for AppleScript
	escapedURL := strings.ReplaceAll(url, "\\", "\\\\")
	escapedURL = strings.ReplaceAll(escapedURL, "\"", "\\\"")

	var navigateScript string
	if navigateTo != "" {
		escapedNavigateTo := strings.ReplaceAll(navigateTo, "\\", "\\\\")
		escapedNavigateTo = strings.ReplaceAll(escapedNavigateTo, "\"", "\\\"")
		navigateScript = fmt.Sprintf(`
						set URL of t to "%s"`, escapedNavigateTo)
	} else {
		navigateScript = ""
	}

	// AppleScript to check Safari for existing tabs with URL and activate if found
	script := fmt.Sprintf(`
		tell application "System Events"
			if not (exists process "Safari") then
				return "false"
			end if
		end tell
		
		tell application "Safari"
			activate
			repeat with w in windows
				repeat with t in tabs of w
					if URL of t starts with "%s" then
						set current tab of w to t
						set index of w to 1%s
						return "true"
					end if
				end repeat
			end repeat
		end tell
		return "false"
	`, escapedURL, navigateScript)

	//nolint:gosec // G204: osascript is a standard macOS utility, the script variable is controlled
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("safari AppleScript error: %w", err)
	}

	result := strings.TrimSpace(string(output))
	return result == "true", nil
}

// activateChromeTab uses AppleScript to find and activate any Chrome tab with the URL.
func (s *service) activateChromeTab(
	ctx context.Context,
	url,
	navigateTo string,
) (bool, error) {
	// Escape the URL for AppleScript
	escapedURL := strings.ReplaceAll(url, "\\", "\\\\")
	escapedURL = strings.ReplaceAll(escapedURL, "\"", "\\\"")

	var navigateScript string
	if navigateTo != "" {
		escapedNavigateTo := strings.ReplaceAll(navigateTo, "\\", "\\\\")
		escapedNavigateTo = strings.ReplaceAll(escapedNavigateTo, "\"", "\\\"")
		navigateScript = fmt.Sprintf(`
						set URL of t to "%s"`, escapedNavigateTo)
	} else {
		navigateScript = ""
	}

	// AppleScript to check Chrome for existing tabs and activate if found
	// Note: We use a counter for tab index since "get index of t" doesn't work reliably in Chrome
	script := fmt.Sprintf(`
		tell application "System Events"
			if not (exists process "Google Chrome") then
				return "false"
			end if
		end tell
		
		tell application "Google Chrome"
			activate
			repeat with w in windows
				set tabIndex to 1
				repeat with t in tabs of w
					if URL of t starts with "%s" then
						set active tab index of w to tabIndex
						set index of w to 1%s
						return "true"
					end if
					set tabIndex to tabIndex + 1
				end repeat
			end repeat
		end tell
		return "false"
	`, escapedURL, navigateScript)

	//nolint:gosec // G204: osascript is a standard macOS utility, the script variable is controlled
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("chrome AppleScript error: %w", err)
	}

	result := strings.TrimSpace(string(output))
	return result == "true", nil
}

// GetInstalledAppVersion reads the version from the installed app's Info.plist.
func (s *service) GetInstalledAppVersion(ctx context.Context) (string, error) {
	plistPath := fmt.Sprintf("%s/Contents/Info.plist", appPath)
	//nolint:gosec // G304: plistPath is constructed from a constant, not user input
	data, err := os.ReadFile(plistPath)
	if err != nil {
		return "", fmt.Errorf("failed to read Info.plist: %w", err)
	}

	// Parse plist XML to extract CFBundleShortVersionString
	version, err := parsePlistVersion(data)
	if err != nil {
		return "", fmt.Errorf("failed to parse plist: %w", err)
	}

	return version, nil
}

// parsePlistVersion extracts CFBundleShortVersionString from plist XML
func parsePlistVersion(data []byte) (string, error) {
	// Simple regex approach to extract version
	// Pattern: <key>CFBundleShortVersionString</key><string>VERSION</string>
	re := regexp.MustCompile(`<key>CFBundleShortVersionString</key>\s*<string>([^<]+)</string>`)
	matches := re.FindSubmatch(data)
	if len(matches) < 2 {
		return "", fmt.Errorf("version not found in plist")
	}
	return string(matches[1]), nil
}
