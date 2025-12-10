//go:build !darwin

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
)

// service is the no-op implementation for non-macOS platforms.
type service struct{}

// NewService creates a new OS actions service.
// On non-macOS platforms, this returns a no-op implementation.
func NewService() Service {
	return &service{}
}

// RestartApp is a no-op on non-macOS platforms.
func (s *service) RestartApp(ctx context.Context, baseURL string) error {
	return fmt.Errorf("restart not supported on this platform")
}

// ActivateBrowserTab is a no-op on non-macOS platforms.
func (s *service) ActivateBrowserTab(ctx context.Context, browser string, url string, navigateTo string) (bool, error) {
	return false, nil
}

// GetInstalledAppVersion is not supported on non-macOS platforms.
func (s *service) GetInstalledAppVersion(ctx context.Context) (string, error) {
	return "", fmt.Errorf("installed app version not supported on this platform")
}
