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
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
)

// RetryConfig configures the retry behavior for database operations.
type RetryConfig struct {
	// MaxElapsedTime is the maximum total time to spend retrying.
	// Default: 5 seconds
	MaxElapsedTime time.Duration

	// InitialInterval is the initial backoff interval.
	// Default: 50ms
	InitialInterval time.Duration

	// MaxInterval is the maximum backoff interval.
	// Default: 1 second
	MaxInterval time.Duration

	// Multiplier is the factor by which the interval increases.
	// Default: 2.0
	Multiplier float64

	// RandomizationFactor adds jitter to prevent thundering herd.
	// Default: 0.5 (±50% of the interval)
	RandomizationFactor float64
}

// DefaultRetryConfig returns sensible defaults for SQLite retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxElapsedTime:      5 * time.Second,
		InitialInterval:     50 * time.Millisecond,
		MaxInterval:         1 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.5,
	}
}

// IsSQLiteBusy checks if the error is a SQLite busy/locked error that should be retried.
func IsSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "SQLITE_BUSY") ||
		strings.Contains(errStr, "database is locked")
}

// RetryOnBusy executes the given operation with automatic retry on SQLite busy errors.
// It uses exponential backoff with jitter to avoid thundering herd problems.
//
// Usage:
//
//	result, err := db.RetryOnBusy(ctx, func() (MyType, error) {
//	    return store.SomeOperation(ctx, args)
//	})
func RetryOnBusy[T any](ctx context.Context, operation func() (T, error)) (T, error) {
	return RetryOnBusyWithConfig(ctx, DefaultRetryConfig(), operation)
}

// RetryOnBusyWithConfig is like RetryOnBusy but allows custom retry configuration.
func RetryOnBusyWithConfig[T any](
	ctx context.Context,
	cfg RetryConfig,
	operation func() (T, error),
) (T, error) {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = cfg.InitialInterval
	b.MaxInterval = cfg.MaxInterval
	b.Multiplier = cfg.Multiplier
	b.RandomizationFactor = cfg.RandomizationFactor

	retryOp := func() (T, error) {
		res, err := operation()
		if err != nil && IsSQLiteBusy(err) {
			// Return the error to trigger a retry
			return res, err
		}
		if err != nil {
			// Non-retryable error - wrap in Permanent to stop retries
			return res, backoff.Permanent(err)
		}
		return res, nil
	}

	return backoff.Retry(ctx, retryOp,
		backoff.WithBackOff(b),
		backoff.WithMaxElapsedTime(cfg.MaxElapsedTime),
	)
}

// RetryVoidOnBusy executes operations that don't return a value (only error).
//
// Usage:
//
//	err := db.RetryVoidOnBusy(ctx, func() error {
//	    return store.DeleteSomething(ctx, id)
//	})
func RetryVoidOnBusy(ctx context.Context, operation func() error) error {
	_, err := RetryOnBusy(ctx, func() (struct{}, error) {
		return struct{}{}, operation()
	})
	return err
}

// RetryVoidOnBusyWithConfig is like RetryVoidOnBusy but allows custom retry configuration.
func RetryVoidOnBusyWithConfig(ctx context.Context, cfg RetryConfig, operation func() error) error {
	_, err := RetryOnBusyWithConfig(ctx, cfg, func() (struct{}, error) {
		return struct{}{}, operation()
	})
	return err
}
