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
