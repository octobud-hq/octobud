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
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package tray provides system tray (menu bar) support for macOS.
package tray

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/assets"
	"github.com/octobud-hq/octobud/backend/internal/api/navigation"
	"github.com/octobud-hq/octobud/backend/internal/osactions"
)

// Status colors using Unicode circles
const (
	statusGreen  = "🟢"
	statusYellow = "🟡"
	statusRed    = "🔴"
)

// Tray manages the macOS menu bar icon and menu.
type Tray struct {
	baseURL string
	onQuit  func()
	logger  *zap.Logger

	mu          sync.RWMutex
	unreadCount int
	lastSync    time.Time
	syncing     bool

	// Navigation broadcaster for SSE events
	navBroadcaster *navigation.Broadcaster

	// OS actions service for browser tab activation
	osActionsSvc osactions.Service

	// Menu items (for updating)
	mStatus      *systray.MenuItem
	mInbox       *systray.MenuItem
	mStarred     *systray.MenuItem
	mSnoozed     *systray.MenuItem
	mMute        *systray.MenuItem
	mMute30m     *systray.MenuItem
	mMute1h      *systray.MenuItem
	mMuteRestDay *systray.MenuItem
	mUnmute      *systray.MenuItem
	mSettings    *systray.MenuItem
	mQuit        *systray.MenuItem
}

// RunWithApp starts the system tray and calls onAppReady when ready.
// This function blocks and MUST be called from the main goroutine on macOS.
// The onAppReady callback should start your application logic in a goroutine.
// baseURL is the URL to open in browser (e.g., "http://localhost:8808" or "http://localhost:5173").
// logger is used for logging tray actions. If nil, a no-op logger will be used.
func RunWithApp(baseURL string, logger *zap.Logger, onAppReady func(t *Tray), onAppExit func()) {
	if logger == nil {
		logger = zap.NewNop()
	}
	osActionsSvc := osactions.NewService()
	t := &Tray{
		baseURL:      baseURL,
		logger:       logger,
		osActionsSvc: osActionsSvc,
	}

	systray.Run(func() {
		t.onReady()
		if onAppReady != nil {
			onAppReady(t)
		}
	}, func() {
		t.onExit()
		if onAppExit != nil {
			onAppExit()
		}
	})
}

// Quit stops the system tray.
func (t *Tray) Quit() {
	systray.Quit()
}

func (t *Tray) onReady() {
	// Set the icon
	systray.SetIcon(assets.TrayIcon)
	systray.SetTooltip("Octobud")

	// Status line (non-clickable)
	t.mStatus = systray.AddMenuItem(statusGreen+" Octobud is running", "")
	t.mStatus.Disable()

	systray.AddSeparator()

	// View menu items
	t.mInbox = systray.AddMenuItem("Inbox", "Open Inbox")
	t.mStarred = systray.AddMenuItem("Starred", "Open Starred notifications")
	t.mSnoozed = systray.AddMenuItem("Snoozed", "Open Snoozed notifications")

	systray.AddSeparator()

	// Mute desktop notifications submenu
	t.mMute = systray.AddMenuItem(
		"Mute desktop notifications",
		"Temporarily mute desktop notifications",
	)
	t.mMute30m = t.mMute.AddSubMenuItem("30 minutes", "Mute for 30 minutes")
	t.mMute1h = t.mMute.AddSubMenuItem("1 hour", "Mute for 1 hour")
	t.mMuteRestDay = t.mMute.AddSubMenuItem("Rest of day", "Mute until end of day")
	t.mUnmute = t.mMute.AddSubMenuItem("Unmute", "Unmute notifications")
	t.mUnmute.Disable() // Disabled by default, enabled when muted

	// Settings
	t.mSettings = systray.AddMenuItem("Settings", "Open Octobud settings")

	systray.AddSeparator()

	// Quit (at the bottom)
	t.mQuit = systray.AddMenuItem("Quit", "Quit Octobud")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-t.mInbox.ClickedCh:
				t.logger.Info("Menu action: Open Inbox")
				t.openBrowser(t.baseURL + "/views/inbox")
			case <-t.mStarred.ClickedCh:
				t.logger.Info("Menu action: Open Starred")
				t.openBrowser(t.baseURL + "/views/starred")
			case <-t.mSnoozed.ClickedCh:
				t.logger.Info("Menu action: Open Snoozed")
				t.openBrowser(t.baseURL + "/views/snoozed")
			case <-t.mMute30m.ClickedCh:
				t.logger.Info("Menu action: Mute for 30 minutes")
				t.setMute("30m")
			case <-t.mMute1h.ClickedCh:
				t.logger.Info("Menu action: Mute for 1 hour")
				t.setMute("1h")
			case <-t.mMuteRestDay.ClickedCh:
				t.logger.Info("Menu action: Mute for rest of day")
				t.setMute("rest_of_day")
			case <-t.mUnmute.ClickedCh:
				t.logger.Info("Menu action: Unmute")
				t.clearMute()
			case <-t.mSettings.ClickedCh:
				t.logger.Info("Menu action: Open Settings")
				t.openBrowser(t.baseURL + "/settings")
			case <-t.mQuit.ClickedCh:
				t.logger.Info("Menu action: Quit")
				if t.onQuit != nil {
					t.onQuit()
				}
				systray.Quit()
				return
			}
		}
	}()

	// Periodically check mute status to update menu
	go t.updateMuteStatus()
}

func (t *Tray) onExit() {
	// Cleanup if needed
}

// SetOnQuit sets the quit callback.
func (t *Tray) SetOnQuit(onQuit func()) {
	t.onQuit = onQuit
}

// SetNavigationBroadcaster sets the navigation broadcaster for sending navigation events.
func (t *Tray) SetNavigationBroadcaster(broadcaster *navigation.Broadcaster) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.navBroadcaster = broadcaster
}

// SetUnreadCount updates the unread notification count displayed in the menu.
func (t *Tray) SetUnreadCount(count int) {
	t.mu.Lock()
	t.unreadCount = count
	t.mu.Unlock()

	t.updateOpenTitle()

	// Update tooltip with unread count
	if count > 0 {
		systray.SetTooltip(fmt.Sprintf("Octobud (%d unread)", count))
	} else {
		systray.SetTooltip("Octobud")
	}
}

// SetLastSync updates the last sync time displayed in the menu.
func (t *Tray) SetLastSync(syncTime time.Time) {
	t.mu.Lock()
	t.lastSync = syncTime
	t.mu.Unlock()

	t.updateStatusText()
}

// SetSyncing updates the syncing state.
func (t *Tray) SetSyncing(syncing bool) {
	t.mu.Lock()
	t.syncing = syncing
	t.mu.Unlock()

	t.updateStatusText()
}

func (t *Tray) updateOpenTitle() {
	if t.mInbox == nil {
		return
	}

	t.mu.RLock()
	count := t.unreadCount
	t.mu.RUnlock()

	if count == 0 {
		t.mInbox.SetTitle("Inbox")
	} else {
		t.mInbox.SetTitle(fmt.Sprintf("Inbox (%d)", count))
	}
}

func (t *Tray) updateStatusText() {
	if t.mStatus == nil {
		return
	}

	t.mu.RLock()
	syncing := t.syncing
	lastSync := t.lastSync
	t.mu.RUnlock()

	var status string
	var color string

	switch {
	case syncing:
		color = statusYellow
		status = "Syncing..."
	case lastSync.IsZero():
		color = statusYellow
		status = "Octobud is running"
	default:
		elapsed := time.Since(lastSync)
		switch {
		case elapsed < 5*time.Minute:
			color = statusGreen
			status = "Octobud is running"
		case elapsed < 30*time.Minute:
			color = statusYellow
			status = fmt.Sprintf("Last sync: %s ago", formatDuration(elapsed))
		default:
			color = statusRed
			status = fmt.Sprintf("Last sync: %s ago", formatDuration(elapsed))
		}
	}

	t.mStatus.SetTitle(color + " " + status)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min"
		}
		return fmt.Sprintf("%d mins", mins)
	}

	hours := int(d.Hours())
	if hours == 1 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", hours)
}

func (t *Tray) openBrowser(url string) {
	t.logger.Info("Opening browser", zap.String("url", url))
	ctx := context.Background()
	var err error
	var tabFound bool
	switch runtime.GOOS {
	case "darwin":
		// Try to activate existing tab first, then fall back to opening new tab
		tabFound = t.activateExistingTab(ctx, url)
		if tabFound {
			t.logger.Info("Successfully activated existing tab")
			// Broadcast navigation event so frontend can navigate within the tab
			t.broadcastNavigation(ctx, url)
		} else {
			t.logger.Info("No existing tab found, opening new tab")
			err = exec.CommandContext(ctx, "open", url).Start()
			if err != nil {
				t.logger.Error("Failed to open new tab", zap.Error(err))
			} else {
				t.logger.Info("Successfully opened new tab")
				// Broadcast navigation event for newly opened tab
				t.broadcastNavigation(ctx, url)
			}
		}
	case "linux":
		err = exec.CommandContext(ctx, "xdg-open", url).Start()
	case "windows":
		err = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	if err != nil {
		// Log error but don't crash
		t.logger.Error("Failed to open browser", zap.Error(err))
	}
}

// broadcastNavigation sends a navigation event to connected frontend clients.
func (t *Tray) broadcastNavigation(ctx context.Context, url string) {
	t.mu.RLock()
	broadcaster := t.navBroadcaster
	t.mu.RUnlock()

	if broadcaster == nil {
		t.logger.Warn(
			"Navigation broadcaster is nil, cannot send navigation event",
			zap.String("url", url),
		)
		return
	}

	// Extract path from full URL for navigation
	// e.g., "http://localhost:5173/views/starred" -> "/views/starred"
	navURL := url
	if strings.HasPrefix(url, t.baseURL) {
		navURL = strings.TrimPrefix(url, t.baseURL)
		if navURL == "" {
			navURL = "/"
		}
	}

	broadcaster.Broadcast(ctx, navURL)
}

// activateExistingTab tries to find any Octobud tab (matching baseURL).
// Returns true if a tab was found and activated, false otherwise.
// Navigation to the specific URL is handled via SSE after activation.
func (t *Tray) activateExistingTab(ctx context.Context, url string) bool {
	t.logger.Info("Attempting to find existing Octobud tab", zap.String("target_url", url))

	if t.osActionsSvc == nil {
		t.logger.Debug("OS actions service not available, skipping tab activation")
		return false
	}

	// Try Safari first (don't navigate - keep current route, navigation handled via SSE)
	found, err := t.osActionsSvc.ActivateBrowserTab(ctx, "safari", url, "")
	if err != nil {
		t.logger.Debug("Failed to activate Safari tab", zap.Error(err))
	} else if found {
		t.logger.Info("Found and activated Octobud tab in Safari")
		return true
	}

	// Try Chrome (don't navigate - keep current route, navigation handled via SSE)
	found, err = t.osActionsSvc.ActivateBrowserTab(ctx, "chrome", url, "")
	if err != nil {
		t.logger.Debug("Failed to activate Chrome tab", zap.Error(err))
	} else if found {
		t.logger.Info("Found and activated Octobud tab in Chrome")
		return true
	}

	t.logger.Info("No existing Octobud tab found")
	return false
}

// setMute sets the mute status via API
func (t *Tray) setMute(duration string) {
	t.logger.Info("Setting mute duration", zap.String("duration", duration))
	apiURL := t.baseURL + "/api/user/mute"
	reqBody := map[string]string{"duration": duration}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.logger.Error("Failed to marshal mute request", zap.Error(err))
		return
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		"PUT",
		apiURL,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.logger.Error("Failed to create mute request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.logger.Error("Failed to set mute (network error)", zap.Error(err))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.logger.Error("Error closing response body", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			body = []byte(fmt.Sprintf("failed to read body: %v", err))
		}
		t.logger.Error(
			"Failed to set mute",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return
	}

	t.logger.Info("Successfully set mute", zap.String("duration", duration))
	// Update menu state
	t.updateMuteMenuState(true)
}

// clearMute clears the mute status via API
func (t *Tray) clearMute() {
	t.logger.Info("Clearing mute")
	apiURL := t.baseURL + "/api/user/mute"
	req, err := http.NewRequestWithContext(context.Background(), "DELETE", apiURL, http.NoBody)
	if err != nil {
		t.logger.Error("Failed to create unmute request", zap.Error(err))
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.logger.Error("Failed to clear mute (network error)", zap.Error(err))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.logger.Error("Error closing response body", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			body = []byte(fmt.Sprintf("failed to read body: %v", err))
		}
		t.logger.Error(
			"Failed to clear mute",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return
	}

	t.logger.Info("Successfully cleared mute")
	// Update menu state
	t.updateMuteMenuState(false)
}

// updateMuteStatus periodically checks mute status and updates menu
func (t *Tray) updateMuteStatus() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Check immediately
	t.checkMuteStatus()

	for range ticker.C {
		t.checkMuteStatus()
	}
}

// checkMuteStatus checks the current mute status and updates menu
func (t *Tray) checkMuteStatus() {
	apiURL := t.baseURL + "/api/user/mute-status"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		t.logger.Error("Failed to create mute status request", zap.Error(err))
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Don't log periodic check failures - they're expected if server isn't ready
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.logger.Warn("failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.logger.Debug("Failed to check mute status", zap.Int("status_code", resp.StatusCode))
		return
	}

	var status struct {
		IsMuted    bool   `json:"isMuted"`
		MutedUntil string `json:"mutedUntil,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.logger.Error("Failed to decode mute status", zap.Error(err))
		return
	}

	t.updateMuteMenuState(status.IsMuted)
}

// updateMuteMenuState updates the mute menu items based on mute status
func (t *Tray) updateMuteMenuState(isMuted bool) {
	if t.mMute == nil {
		return
	}

	if isMuted {
		t.mMute30m.Disable()
		t.mMute1h.Disable()
		t.mMuteRestDay.Disable()
		t.mUnmute.Enable()
	} else {
		t.mMute30m.Enable()
		t.mMute1h.Enable()
		t.mMuteRestDay.Enable()
		t.mUnmute.Disable()
	}
}
