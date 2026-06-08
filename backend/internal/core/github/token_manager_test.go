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
	"context"
	"database/sql"
	"net/http"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/db"
	dbmocks "github.com/octobud-hq/octobud/backend/internal/db/mocks"
	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
)

// skipIfNotDarwin skips tests that exercise StartKeychainRecovery, which is
// macOS-only — it short-circuits on other platforms because the underlying
// keychain-locked-at-boot scenario doesn't apply.
func skipIfNotDarwin(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("keychain recovery is macOS-only")
	}
}

// newTestManager returns a TokenManager wired to a mock github client. The
// mock accepts the SetTokenObserver call that NewTokenManager performs and
// captures the observer so tests can invoke it directly.
func newTestManager(t *testing.T) (*TokenManager, githubinterfaces.TokenObserverFunc) {
	t.Helper()
	ctrl := gomock.NewController(t)

	mockClient := githubmocks.NewMockClient(ctrl)
	var observer githubinterfaces.TokenObserverFunc
	mockClient.EXPECT().
		SetTokenObserver(gomock.Any()).
		Do(func(fn githubinterfaces.TokenObserverFunc) { observer = fn }).
		Times(1)

	// store/encryptor/keychain/authService are unused by the paths under test.
	m := NewTokenManager(nil, nil, nil, mockClient, nil, zap.NewNop())
	if observer == nil {
		t.Fatal("NewTokenManager did not register a token observer")
	}
	return m, observer
}

func TestObserver_CapturesExpirationHeader(t *testing.T) {
	m, observe := newTestManager(t)

	expected := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	observe(http.StatusOK, &expected)

	got := m.GetTokenExpiresAt()
	if got == nil {
		t.Fatal("expected expiration to be captured, got nil")
	}
	if !got.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, *got)
	}
}

func TestObserver_UnauthorizedMarksInvalid(t *testing.T) {
	m, observe := newTestManager(t)
	m.currentSource = TokenSourceStored // required for GetTokenHealth to evaluate

	observe(http.StatusUnauthorized, nil)

	if got := m.GetTokenHealth(); got != TokenHealthInvalid {
		t.Fatalf("expected %q after 401, got %q", TokenHealthInvalid, got)
	}
}

func TestObserver_TwoHundredClearsInvalid(t *testing.T) {
	m, observe := newTestManager(t)
	m.currentSource = TokenSourceStored

	observe(http.StatusUnauthorized, nil)
	observe(http.StatusOK, nil)

	if got := m.GetTokenHealth(); got != TokenHealthOK {
		t.Fatalf("expected %q after recovery, got %q", TokenHealthOK, got)
	}
}

func TestObserver_NonOKNonUnauthorizedDoesNotToggle(t *testing.T) {
	m, observe := newTestManager(t)
	m.currentSource = TokenSourceStored

	// 5xx or rate-limit-style responses shouldn't flip the invalid flag —
	// they're transient, not auth problems.
	observe(http.StatusServiceUnavailable, nil)
	if got := m.GetTokenHealth(); got != TokenHealthOK {
		t.Fatalf("503 should not mark token invalid, got %q", got)
	}

	observe(http.StatusUnauthorized, nil)
	observe(http.StatusInternalServerError, nil)
	if got := m.GetTokenHealth(); got != TokenHealthInvalid {
		t.Fatalf("500 should not clear prior invalid state, got %q", got)
	}
}

func TestGetTokenHealth_NoTokenReturnsOK(t *testing.T) {
	m, _ := newTestManager(t)
	// currentSource defaults to TokenSourceNone via NewTokenManager.

	if got := m.GetTokenHealth(); got != TokenHealthOK {
		t.Fatalf(
			"disconnected should return OK (banner is hidden via connected=false), got %q",
			got,
		)
	}
}

func TestGetTokenHealth_OAuthShortCircuits(t *testing.T) {
	m, observe := newTestManager(t)
	m.currentSource = TokenSourceOAuth

	// Even with an expired token + 401 observed, OAuth sessions should never
	// flip the banner — they refresh automatically and aren't user-actionable.
	past := time.Now().Add(-24 * time.Hour)
	observe(http.StatusUnauthorized, &past)

	if got := m.GetTokenHealth(); got != TokenHealthOK {
		t.Fatalf("OAuth source should short-circuit to OK, got %q", got)
	}
}

func TestGetTokenHealth_ExpiredReturnsInvalid(t *testing.T) {
	m, observe := newTestManager(t)
	m.currentSource = TokenSourceStored

	past := time.Now().Add(-1 * time.Hour)
	observe(http.StatusOK, &past)

	if got := m.GetTokenHealth(); got != TokenHealthInvalid {
		t.Fatalf("past expiration should report invalid, got %q", got)
	}
}

func TestGetTokenHealth_ExpiringWithinWindow(t *testing.T) {
	m, observe := newTestManager(t)
	m.currentSource = TokenSourceStored

	soon := time.Now().Add(3 * 24 * time.Hour) // 3 days out, inside 7-day window
	observe(http.StatusOK, &soon)

	if got := m.GetTokenHealth(); got != TokenHealthExpiring {
		t.Fatalf("expiration within warning window should report expiring, got %q", got)
	}
}

func TestGetTokenHealth_FarFutureReturnsOK(t *testing.T) {
	m, observe := newTestManager(t)
	m.currentSource = TokenSourceStored

	far := time.Now().Add(60 * 24 * time.Hour) // 60 days out, outside window
	observe(http.StatusOK, &far)

	if got := m.GetTokenHealth(); got != TokenHealthOK {
		t.Fatalf("far-future expiration should report OK, got %q", got)
	}
}

func TestGetTokenHealth_NoExpirationObservedReturnsOK(t *testing.T) {
	m, _ := newTestManager(t)
	m.currentSource = TokenSourceStored
	// Classic PATs have no expiration header — nothing observed, token is fine.

	if got := m.GetTokenHealth(); got != TokenHealthOK {
		t.Fatalf("classic PAT (no expiration) should report OK, got %q", got)
	}
}

func TestResetHealth_ClearsObservedState(t *testing.T) {
	m, observe := newTestManager(t)
	m.currentSource = TokenSourceStored

	soon := time.Now().Add(2 * 24 * time.Hour)
	observe(http.StatusUnauthorized, &soon)

	m.resetHealth()

	if m.GetTokenExpiresAt() != nil {
		t.Fatal("resetHealth should clear expiration")
	}
	if got := m.GetTokenHealth(); got != TokenHealthOK {
		t.Fatalf("resetHealth should clear invalid flag, got %q", got)
	}
}

func TestStartKeychainRecovery_NoopWhenTokenLoaded(t *testing.T) {
	skipIfNotDarwin(t)
	// When Initialize already loaded a token, recovery shouldn't start —
	// passing a nil store would panic if we got past the early return.
	m, _ := newTestManager(t)
	m.currentSource = TokenSourceStored

	m.StartKeychainRecovery(context.Background())

	m.recoveryMu.Lock()
	defer m.recoveryMu.Unlock()
	if m.recoveryCancel != nil {
		t.Fatal("recovery loop should not start when token is already loaded")
	}
}

func TestStartKeychainRecovery_NoopWhenNoPriorUser(t *testing.T) {
	skipIfNotDarwin(t)
	// Fresh install case: user has never connected GitHub. We shouldn't loop
	// forever trying to load a token that doesn't exist.
	ctrl := gomock.NewController(t)
	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().SetTokenObserver(gomock.Any()).Times(1)

	mockStore := dbmocks.NewMockStore(ctrl)
	mockStore.EXPECT().GetUser(gomock.Any()).Return(db.User{}, nil).Times(1)

	m := NewTokenManager(mockStore, nil, nil, mockClient, nil, zap.NewNop())

	m.StartKeychainRecovery(context.Background())

	m.recoveryMu.Lock()
	defer m.recoveryMu.Unlock()
	if m.recoveryCancel != nil {
		t.Fatal("recovery loop should not start when no prior connection exists")
	}
}

func TestStartKeychainRecovery_RetriesUntilStopped(t *testing.T) {
	skipIfNotDarwin(t)
	// The recovery loop should keep retrying Initialize while currentSource
	// stays None, and exit promptly once stopRecovery is called.
	ctrl := gomock.NewController(t)
	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().SetTokenObserver(gomock.Any()).Times(1)

	priorUser := db.User{
		GithubUsername: sql.NullString{String: "alice", Valid: true},
		// No keychain (nil) and no encrypted token — Initialize will fall
		// through to TokenSourceNone, keeping the loop alive.
	}

	var getUserCalls atomic.Int32
	mockStore := dbmocks.NewMockStore(ctrl)
	mockStore.EXPECT().GetUser(gomock.Any()).
		DoAndReturn(func(_ context.Context) (db.User, error) {
			getUserCalls.Add(1)
			return priorUser, nil
		}).
		MinTimes(2)

	m := NewTokenManager(mockStore, nil, nil, mockClient, nil, zap.NewNop())
	// Tight backoff so the test runs in milliseconds, but not so tight that
	// CI scheduler jitter starves the loop within the polling window.
	m.recoveryBackoff = []time.Duration{5 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.StartKeychainRecovery(ctx)

	// Wait for at least one retry to happen. StartKeychainRecovery itself
	// calls GetUser once; each Initialize retry calls it again.
	waitFor(t, 500*time.Millisecond, func() bool {
		return getUserCalls.Load() >= 3
	}, "expected loop to retry at least twice")

	m.stopRecovery()

	// Wait for the goroutine to exit. defer in runRecoveryLoop sets
	// recoveryCancel back to nil after exiting the loop.
	waitFor(t, 500*time.Millisecond, func() bool {
		m.recoveryMu.Lock()
		defer m.recoveryMu.Unlock()
		return m.recoveryCancel == nil
	}, "recovery goroutine did not exit after stopRecovery")

	// After exiting, no further GetUser calls should occur.
	settled := getUserCalls.Load()
	time.Sleep(20 * time.Millisecond)
	if got := getUserCalls.Load(); got > settled {
		t.Fatalf("loop kept calling GetUser after stop: %d → %d", settled, got)
	}
}

// waitFor polls cond until it returns true or the timeout elapses, failing the
// test with msg if the timeout is hit.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal(msg)
}
