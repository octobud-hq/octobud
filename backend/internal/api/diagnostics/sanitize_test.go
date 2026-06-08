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

package diagnostics

import (
	"strings"
	"testing"
)

// withFakeHome overrides the package-level homeDir resolver for the duration
// of the test, so we can assert path-scrubbing behavior without depending on
// the test runner's actual $HOME.
func withFakeHome(t *testing.T, fake string) {
	t.Helper()
	orig := homeDir
	homeDir = func() string { return fake }
	t.Cleanup(func() { homeDir = orig })
}

func TestSanitizeString_RedactsClassicPAT(t *testing.T) {
	withFakeHome(t, "")
	in := "request failed with token ghp_abcdefghijklmnopqrstuvwxyz1234"
	out := sanitizeString(in)
	if out == in {
		t.Fatal("expected ghp_ prefix to be redacted")
	}
	if strings.Contains(out, "ghp_a") {
		t.Fatalf("token leaked through redaction: %q", out)
	}
}

func TestSanitizeString_RedactsFineGrainedPAT(t *testing.T) {
	withFakeHome(t, "")
	in := "auth=github_pat_11AAAAAAA0abcdefghijklmnopqrstuvwxyz"
	out := sanitizeString(in)
	if strings.Contains(out, "github_pat_11") {
		t.Fatalf("fine-grained PAT leaked: %q", out)
	}
}

func TestSanitizeString_RedactsBearer(t *testing.T) {
	withFakeHome(t, "")
	in := `Authorization: Bearer eyJhbGci.payload.sig`
	out := sanitizeString(in)
	if strings.Contains(out, "eyJhbGci") {
		t.Fatalf("bearer token leaked: %q", out)
	}
	if !strings.Contains(out, "Bearer "+redacted) {
		t.Fatalf("bearer replacement missing: %q", out)
	}
}

func TestSanitizeString_ReplacesHome(t *testing.T) {
	withFakeHome(t, "/Users/alice")
	in := "open /Users/alice/Library/Application Support/octobud/db.sqlite failed"
	out := sanitizeString(in)
	if strings.Contains(out, "/Users/alice") {
		t.Fatalf("home prefix not replaced: %q", out)
	}
	if !strings.Contains(out, "~/Library") {
		t.Fatalf("expected ~ substitution: %q", out)
	}
	// sanitizeString must NOT escape spaces (only sanitizePath does that —
	// log messages are free text and backslash-spaces would be jarring).
	if strings.Contains(out, `\ `) {
		t.Fatalf("sanitizeString must not escape spaces: %q", out)
	}
}

func TestSanitizeString_EmptyStringPassesThrough(t *testing.T) {
	withFakeHome(t, "/Users/alice")
	if got := sanitizeString(""); got != "" {
		t.Fatalf("expected empty input to return empty, got %q", got)
	}
}

func TestSanitizePath_ReplacesHomeAndEscapesSpaces(t *testing.T) {
	withFakeHome(t, "/Users/alice")
	in := "/Users/alice/Library/Application Support/octobud/logs"
	out := sanitizePath(in)
	want := `~/Library/Application\ Support/octobud/logs`
	if out != want {
		t.Fatalf("sanitizePath mismatch\n got: %q\nwant: %q", out, want)
	}
}

func TestSanitizePath_NoHomeStillEscapesSpaces(t *testing.T) {
	withFakeHome(t, "")
	in := "/var/folders/My Stuff/x"
	out := sanitizePath(in)
	want := `/var/folders/My\ Stuff/x`
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestSanitizeEntry_RedactsSensitiveFieldKeys(t *testing.T) {
	withFakeHome(t, "")
	entry := LogEntry{
		Message: "ok",
		Fields: map[string]interface{}{
			"authorization": "ghp_realtokenvalue123456789",
			"github_token":  "ghp_othertoken9876543210",
			"x-api-key":     "live-api-key-do-not-leak",
			"cookie":        "session=abcdef",
			"set-cookie":    "session=abcdef; HttpOnly",
			"client_secret": "supersecret",
			"username":      "should-survive",
		},
	}
	sanitizeEntry(&entry)
	for _, key := range []string{
		"authorization", "github_token", "x-api-key", "cookie", "set-cookie", "client_secret",
	} {
		if v, ok := entry.Fields[key]; !ok || v != redacted {
			t.Errorf("field %q not redacted: got %v", key, v)
		}
	}
	if entry.Fields["username"] != "should-survive" {
		t.Errorf("non-sensitive field clobbered: %v", entry.Fields["username"])
	}
}

func TestSanitizeEntry_ScrubsTokensInNonSensitiveStringFields(t *testing.T) {
	withFakeHome(t, "/Users/alice")
	entry := LogEntry{
		Message:    "failed opening /Users/alice/Library/octobud.log",
		Stacktrace: "raw: ghp_abcdefghijklmnopqrstuvwxyz0123",
		Fields: map[string]interface{}{
			"path": "/Users/alice/Library/octobud.log",
			"note": "Authorization: Bearer eyJhbGci.payload.sig",
		},
	}
	sanitizeEntry(&entry)
	if strings.Contains(entry.Message, "/Users/alice") {
		t.Errorf("home not replaced in Message: %q", entry.Message)
	}
	if strings.Contains(entry.Stacktrace, "ghp_abc") {
		t.Errorf("token leaked in Stacktrace: %q", entry.Stacktrace)
	}
	if s, _ := entry.Fields["path"].(string); strings.Contains(s, "/Users/alice") {
		t.Errorf("home not replaced in field: %q", s)
	}
	if s, _ := entry.Fields["note"].(string); strings.Contains(s, "eyJhbGci") {
		t.Errorf("bearer not redacted in field: %q", s)
	}
}
