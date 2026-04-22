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
	"net/http"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
)

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
