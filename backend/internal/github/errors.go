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

package github

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"syscall"
)

// IsRetriableError determines if an error from GitHub API calls should be retried.
// Returns true for transient errors that might succeed on retry:
// - Network errors (timeout, connection refused, etc.)
// - HTTP 429 (rate limit)
// - HTTP 5xx (server errors)
// Returns false for permanent errors that won't succeed on retry:
// - HTTP 403 (forbidden - org restrictions)
// - HTTP 404 (not found - deleted PR/issue)
// - HTTP 401 (unauthorized - auth issues)
func IsRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
		return true // Connection errors are retriable
	}

	// Check for connection refused and other syscall errors
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		// Connection refused, connection reset, etc. are retriable
		return true
	}

	// Check for URL errors (often wrap network errors)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// If it's a timeout or temporary error, retry
		if urlErr.Timeout() || urlErr.Temporary() {
			return true
		}
		// Recursively check the underlying error
		return IsRetriableError(urlErr.Err)
	}

	// Parse error message for HTTP status codes
	// FetchSubjectRaw returns: "github: subject status %d: %s"
	errStr := err.Error()

	// Look for "status" followed by a number
	statusIdx := strings.Index(errStr, "status")
	if statusIdx == -1 {
		// No status code in error, check if it's a generic network error
		// Most other errors are likely network-related and should be retriable
		return true
	}

	// Extract status code from error message
	// Format: "github: subject status 403: ..."
	parts := strings.Fields(errStr[statusIdx:])
	if len(parts) < 2 {
		return true // Can't parse, assume retriable
	}

	// parts[1] might be "403:" so strip any trailing colon
	statusStr := strings.TrimSuffix(parts[1], ":")
	statusCode, parseErr := strconv.Atoi(statusStr)
	if parseErr != nil {
		return true // Can't parse status code, assume retriable
	}

	// Classify based on status code
	switch statusCode {
	case 429: // Rate limit - definitely retriable
		return true
	case 500, 502, 503, 504: // Server errors - retriable
		return true
	case 403, 404, 401: // Permanent errors - not retriable
		return false
	default:
		// Unknown status code - be conservative and retry
		// 4xx errors (except 403, 404, 401) might be retriable (e.g., 408 Request Timeout)
		if statusCode >= 500 {
			return true // All 5xx are retriable
		}
		if statusCode == 408 { // Request Timeout
			return true
		}
		// Other 4xx errors - assume not retriable
		return false
	}
}
