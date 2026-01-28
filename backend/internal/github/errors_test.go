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
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error is not retriable",
			err:      nil,
			expected: false,
		},
		{
			name:     "network timeout is retriable",
			err:      &net.OpError{Err: &timeoutError{}},
			expected: true,
		},
		{
			name:     "connection refused is retriable",
			err:      &net.OpError{Err: syscall.ECONNREFUSED},
			expected: true,
		},
		{
			name:     "URL error wrapping timeout is retriable",
			err:      &url.Error{Err: &timeoutError{}},
			expected: true,
		},
		{
			name:     "URL error wrapping network error is retriable",
			err:      &url.Error{Err: errors.New("connection refused")},
			expected: true,
		},
		{
			name:     "HTTP 429 rate limit is retriable",
			err:      errors.New("github: subject status 429: rate limit exceeded"),
			expected: true,
		},
		{
			name:     "HTTP 500 server error is retriable",
			err:      errors.New("github: subject status 500: internal server error"),
			expected: true,
		},
		{
			name:     "HTTP 502 bad gateway is retriable",
			err:      errors.New("github: subject status 502: bad gateway"),
			expected: true,
		},
		{
			name:     "HTTP 503 service unavailable is retriable",
			err:      errors.New("github: subject status 503: service unavailable"),
			expected: true,
		},
		{
			name:     "HTTP 504 gateway timeout is retriable",
			err:      errors.New("github: subject status 504: gateway timeout"),
			expected: true,
		},
		{
			name:     "HTTP 408 request timeout is retriable",
			err:      errors.New("github: subject status 408: request timeout"),
			expected: true,
		},
		{
			name:     "HTTP 403 forbidden is not retriable",
			err:      errors.New("github: subject status 403: forbidden"),
			expected: false,
		},
		{
			name:     "HTTP 404 not found is not retriable",
			err:      errors.New("github: subject status 404: not found"),
			expected: false,
		},
		{
			name:     "HTTP 401 unauthorized is not retriable",
			err:      errors.New("github: subject status 401: unauthorized"),
			expected: false,
		},
		{
			name:     "HTTP 400 bad request is not retriable",
			err:      errors.New("github: subject status 400: bad request"),
			expected: false,
		},
		{
			name:     "HTTP 422 unprocessable entity is not retriable",
			err:      errors.New("github: subject status 422: unprocessable entity"),
			expected: false,
		},
		{
			name:     "error without status code defaults to retriable",
			err:      errors.New("some network error"),
			expected: true,
		},
		{
			name:     "malformed error message defaults to retriable",
			err:      errors.New("github: subject status abc: invalid"),
			expected: true,
		},
		{
			name:     "URL error wrapping network error is retriable",
			err:      &url.Error{Err: &net.OpError{Err: syscall.ECONNREFUSED}},
			expected: true,
		},
		{
			name:     "URL error wrapping timeout is retriable",
			err:      &url.Error{Err: &timeoutError{}},
			expected: true,
		},
		{
			name:     "any 5xx error is retriable",
			err:      errors.New("github: subject status 507: insufficient storage"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetriableError(tt.err)
			require.Equal(
				t,
				tt.expected,
				result,
				"IsRetriableError(%v) = %v, want %v",
				tt.err,
				result,
				tt.expected,
			)
		})
	}
}

// timeoutError is a simple error that implements net.Error with Timeout() returning true
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return false }

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 429,
		Message:    "rate limit exceeded",
	}
	require.Equal(t, "github: API status 429: rate limit exceeded", err.Error())
}

func TestAPIError_GetRetryDelay(t *testing.T) {
	t.Run("prefers RetryAfter when set", func(t *testing.T) {
		// Use a far-future RateLimitReset to avoid timing sensitivity
		futureReset := time.Now().Add(10 * time.Minute)
		err := &APIError{
			StatusCode:     429,
			Message:        "rate limited",
			RetryAfter:     durationPtr(60 * time.Second),
			RateLimitReset: &futureReset,
		}
		result := err.GetRetryDelay()
		require.NotNil(t, result)
		require.Equal(t, 60*time.Second, *result)
	})

	t.Run("uses RateLimitReset when RetryAfter not set", func(t *testing.T) {
		// Use a generous buffer to avoid timing sensitivity in slow CI
		futureReset := time.Now().Add(5 * time.Minute)
		err := &APIError{
			StatusCode:     429,
			Message:        "rate limited",
			RateLimitReset: &futureReset,
		}
		result := err.GetRetryDelay()
		require.NotNil(t, result)
		// Allow for up to 10 seconds of test execution time
		require.True(t, *result > 4*time.Minute+50*time.Second && *result <= 5*time.Minute,
			"expected delay ~5m, got %v", *result)
	})

	t.Run("returns nil when neither set", func(t *testing.T) {
		err := &APIError{
			StatusCode: 500,
			Message:    "server error",
		}
		require.Nil(t, err.GetRetryDelay())
	})

	t.Run("returns nil when RateLimitReset is in the past", func(t *testing.T) {
		// Use a clearly past time to avoid boundary issues
		pastReset := time.Now().Add(-5 * time.Minute)
		err := &APIError{
			StatusCode:     429,
			Message:        "rate limited",
			RateLimitReset: &pastReset,
		}
		require.Nil(t, err.GetRetryDelay())
	})
}

func TestAPIError_IsRateLimitError(t *testing.T) {
	// Fixed time for tests that only check nil vs non-nil
	anyTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		err      *APIError
		expected bool
	}{
		{
			name:     "429 is rate limit error",
			err:      &APIError{StatusCode: 429},
			expected: true,
		},
		{
			name:     "403 with RateLimitReset is rate limit error",
			err:      &APIError{StatusCode: 403, RateLimitReset: &anyTime},
			expected: true,
		},
		{
			name:     "403 without RateLimitReset is not rate limit error",
			err:      &APIError{StatusCode: 403},
			expected: false,
		},
		{
			name:     "500 is not rate limit error",
			err:      &APIError{StatusCode: 500},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.err.IsRateLimitError())
		})
	}
}

func TestAPIError_IsServerError(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected bool
	}{
		{name: "500 is server error", err: &APIError{StatusCode: 500}, expected: true},
		{name: "502 is server error", err: &APIError{StatusCode: 502}, expected: true},
		{name: "503 is server error", err: &APIError{StatusCode: 503}, expected: true},
		{name: "504 is server error", err: &APIError{StatusCode: 504}, expected: true},
		{name: "599 is server error", err: &APIError{StatusCode: 599}, expected: true},
		{name: "400 is not server error", err: &APIError{StatusCode: 400}, expected: false},
		{name: "429 is not server error", err: &APIError{StatusCode: 429}, expected: false},
		{name: "200 is not server error", err: &APIError{StatusCode: 200}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.err.IsServerError())
		})
	}
}

func TestGetRetryDelayFromError(t *testing.T) {
	retryDuration := 30 * time.Second

	tests := []struct {
		name          string
		err           error
		expectNil     bool
		expectSeconds int
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			expectNil: true,
		},
		{
			name:      "non-APIError returns nil",
			err:       errors.New("some error"),
			expectNil: true,
		},
		{
			name: "APIError with RetryAfter returns delay",
			err: &APIError{
				StatusCode: 429,
				RetryAfter: &retryDuration,
			},
			expectNil:     false,
			expectSeconds: 30,
		},
		{
			name: "wrapped APIError returns delay",
			err: errors.Join(
				errors.New("failed to fetch"),
				&APIError{StatusCode: 429, RetryAfter: &retryDuration},
			),
			expectNil:     false,
			expectSeconds: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRetryDelayFromError(tt.err)
			if tt.expectNil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, time.Duration(tt.expectSeconds)*time.Second, *result)
			}
		})
	}
}

func TestIsRetriableError_WithAPIError(t *testing.T) {
	// Fixed time for tests that only check nil vs non-nil
	anyTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		err      *APIError
		expected bool
	}{
		{
			name:     "APIError 429 is retriable",
			err:      &APIError{StatusCode: 429, Message: "rate limited"},
			expected: true,
		},
		{
			name:     "APIError 500 is retriable",
			err:      &APIError{StatusCode: 500, Message: "server error"},
			expected: true,
		},
		{
			name:     "APIError 502 is retriable",
			err:      &APIError{StatusCode: 502, Message: "bad gateway"},
			expected: true,
		},
		{
			name:     "APIError 403 with RateLimitReset is retriable",
			err:      &APIError{StatusCode: 403, Message: "rate limited", RateLimitReset: &anyTime},
			expected: true,
		},
		{
			name:     "APIError 403 without RateLimitReset is not retriable",
			err:      &APIError{StatusCode: 403, Message: "forbidden"},
			expected: false,
		},
		{
			name:     "APIError 404 is not retriable",
			err:      &APIError{StatusCode: 404, Message: "not found"},
			expected: false,
		},
		{
			name:     "APIError 401 is not retriable",
			err:      &APIError{StatusCode: 401, Message: "unauthorized"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, IsRetriableError(tt.err))
		})
	}
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}
