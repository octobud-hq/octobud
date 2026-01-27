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
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// APIError represents an error from the GitHub API with optional retry information.
// It captures response headers that indicate when the client should retry.
type APIError struct {
	StatusCode     int            // HTTP status code
	Message        string         // Error message from response body
	RetryAfter     *time.Duration // Parsed from Retry-After header (seconds to wait)
	RateLimitReset *time.Time     // Parsed from x-ratelimit-reset header (UTC timestamp)
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("github: API status %d: %s", e.StatusCode, e.Message)
}

// GetRetryDelay returns the recommended duration to wait before retrying.
// It prefers RetryAfter if set, otherwise calculates from RateLimitReset.
// Returns nil if no retry information is available.
func (e *APIError) GetRetryDelay() *time.Duration {
	if e.RetryAfter != nil {
		return e.RetryAfter
	}
	if e.RateLimitReset != nil {
		delay := time.Until(*e.RateLimitReset)
		if delay > 0 {
			return &delay
		}
	}
	return nil
}

// IsRateLimitError returns true if this is a rate limit error (429 or 403 with rate limit).
func (e *APIError) IsRateLimitError() bool {
	return e.StatusCode == 429 || (e.StatusCode == 403 && e.RateLimitReset != nil)
}

// IsServerError returns true if this is a server error (5xx).
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// GetRetryDelayFromError extracts the retry delay from an error if it's an APIError.
// Returns nil if the error is not an APIError or has no retry information.
func GetRetryDelayFromError(err error) *time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.GetRetryDelay()
	}
	return nil
}

// IsRetriableError determines if an error from GitHub API calls should be retried.
// Returns true for transient errors that might succeed on retry:
// - Network errors (timeout, connection refused, etc.)
// - HTTP 429 (rate limit)
// - HTTP 5xx (server errors)
// Returns false for permanent errors that won't succeed on retry:
// - HTTP 403 (forbidden - org restrictions, unless rate limited)
// - HTTP 404 (not found - deleted PR/issue)
// - HTTP 401 (unauthorized - auth issues)
func IsRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our structured APIError type
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return isRetriableStatusCode(apiErr.StatusCode, apiErr.RateLimitReset != nil)
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

	// Parse error message for HTTP status codes (legacy path for non-APIError errors)
	// Format: "github: API status %d: %s" or similar patterns containing "status <code>"
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

	return isRetriableStatusCode(statusCode, false)
}

// isRetriableStatusCode determines if an HTTP status code indicates a retriable error.
// The isRateLimited parameter indicates if rate limit headers were present (for 403 responses).
func isRetriableStatusCode(statusCode int, isRateLimited bool) bool {
	switch statusCode {
	case 429: // Rate limit - definitely retriable
		return true
	case 500, 502, 503, 504: // Server errors - retriable
		return true
	case 403:
		// 403 is retriable if it's due to rate limiting, otherwise permanent
		return isRateLimited
	case 404, 401: // Permanent errors - not retriable
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
