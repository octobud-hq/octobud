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

package db

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsSQLiteBusy(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"SQLITE_BUSY", errors.New("database is locked (5) (SQLITE_BUSY)"), true},
		{"database is locked", errors.New("failed: database is locked"), true},
		{"other error", errors.New("some other error"), false},
		{
			"wrapped SQLITE_BUSY",
			errors.New("failed to upsert: database is locked (5) (SQLITE_BUSY)"),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSQLiteBusy(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRetryOnBusy_Success(t *testing.T) {
	ctx := context.Background()

	result, err := RetryOnBusy(ctx, func() (string, error) {
		return "success", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success", result)
}

func TestRetryOnBusy_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	var attempts atomic.Int32

	_, err := RetryOnBusy(ctx, func() (string, error) {
		attempts.Add(1)
		return "", errors.New("permanent error")
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "permanent error")
	require.Equal(t, int32(1), attempts.Load(), "should only attempt once for non-retryable errors")
}

func TestRetryOnBusy_RetriesOnSQLiteBusy(t *testing.T) {
	ctx := context.Background()
	var attempts atomic.Int32

	result, err := RetryOnBusy(ctx, func() (string, error) {
		attempt := attempts.Add(1)
		if attempt < 3 {
			return "", errors.New("database is locked (5) (SQLITE_BUSY)")
		}
		return "success after retries", nil
	})

	require.NoError(t, err)
	require.Equal(t, "success after retries", result)
	require.Equal(t, int32(3), attempts.Load(), "should have retried until success")
}

func TestRetryOnBusy_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var attempts atomic.Int32

	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := RetryOnBusy(ctx, func() (string, error) {
		attempts.Add(1)
		return "", errors.New("database is locked (5) (SQLITE_BUSY)")
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), "should return context error")
}

func TestRetryOnBusy_MaxElapsedTime(t *testing.T) {
	ctx := context.Background()
	var attempts atomic.Int32

	cfg := RetryConfig{
		MaxElapsedTime:      200 * time.Millisecond,
		InitialInterval:     50 * time.Millisecond,
		MaxInterval:         100 * time.Millisecond,
		Multiplier:          1.5,
		RandomizationFactor: 0,
	}

	start := time.Now()
	_, err := RetryOnBusyWithConfig(ctx, cfg, func() (string, error) {
		attempts.Add(1)
		return "", errors.New("database is locked (5) (SQLITE_BUSY)")
	})
	elapsed := time.Since(start)

	require.Error(t, err)
	require.True(t, elapsed < 500*time.Millisecond, "should stop retrying within max elapsed time")
	require.Greater(t, attempts.Load(), int32(1), "should have attempted multiple times")
}

func TestRetryVoidOnBusy_Success(t *testing.T) {
	ctx := context.Background()

	err := RetryVoidOnBusy(ctx, func() error {
		return nil
	})

	require.NoError(t, err)
}

func TestRetryVoidOnBusy_RetriesOnSQLiteBusy(t *testing.T) {
	ctx := context.Background()
	var attempts atomic.Int32

	err := RetryVoidOnBusy(ctx, func() error {
		attempt := attempts.Add(1)
		if attempt < 3 {
			return errors.New("SQLITE_BUSY")
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, int32(3), attempts.Load())
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	require.Equal(t, 5*time.Second, cfg.MaxElapsedTime)
	require.Equal(t, 50*time.Millisecond, cfg.InitialInterval)
	require.Equal(t, 1*time.Second, cfg.MaxInterval)
	require.Equal(t, 2.0, cfg.Multiplier)
	require.Equal(t, 0.5, cfg.RandomizationFactor)
}
