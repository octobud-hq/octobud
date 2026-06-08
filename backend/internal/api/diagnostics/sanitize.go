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
	"os"
	"regexp"
	"strings"
	"sync"
)

// homeDir resolves and caches the current user's home directory once. We use
// it to redact `/Users/<name>/...` paths from copy-bundle output — the local
// account name is incidental PII users may not realize they're sharing.
var homeDir = sync.OnceValue(func() string {
	if dir, err := os.UserHomeDir(); err == nil {
		return dir
	}
	return ""
})

// sanitizePath strips the user's home prefix to "~" and shell-escapes spaces,
// so paths in the bundle paste cleanly into a terminal without leaking the
// local account name. Returns the input unchanged if no home is detected.
func sanitizePath(p string) string {
	p = replaceHome(p)
	return strings.ReplaceAll(p, " ", `\ `)
}

// replaceHome substitutes "~" for any occurrence of the user's home directory.
// Applied to all sanitized strings — log messages routinely embed full paths
// when reporting file open / decode errors.
func replaceHome(s string) string {
	home := homeDir()
	if home == "" || s == "" {
		return s
	}
	return strings.ReplaceAll(s, home, "~")
}

// Patterns that look like credentials. Matched against both message text and
// structured field values. Conservative on purpose — we'd rather over-redact
// than leak.
//
// - ghp_, gho_, ghu_, ghs_, ghr_: classic and fine-grained PAT prefixes
// - github_pat_: fine-grained PAT prefix (with underscore segment)
// - "Bearer <token>" inline in messages
var (
	tokenPrefixRegex = regexp.MustCompile(
		`\b(gh[pousr]_[A-Za-z0-9_]{16,}|github_pat_[A-Za-z0-9_]{16,})\b`,
	)
	bearerRegex = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._\-]{8,}`)
)

const redacted = "[REDACTED]"

// sanitizeString redacts token-like substrings and home-dir paths from text.
func sanitizeString(s string) string {
	if s == "" {
		return s
	}
	s = tokenPrefixRegex.ReplaceAllString(s, redacted)
	s = bearerRegex.ReplaceAllString(s, "Bearer "+redacted)
	s = replaceHome(s)
	return s
}

// sensitiveFieldKeys are structured-log field names whose values get blanket
// redacted, regardless of content. Lowercased for comparison.
var sensitiveFieldKeys = map[string]struct{}{
	// Generic auth / credential keys.
	"authorization": {},
	"auth":          {},
	"auth_token":    {},
	"authtoken":     {},
	"bearer_token":  {},
	"bearertoken":   {},
	"token":         {},
	"access_token":  {},
	"accesstoken":   {},
	"refresh_token": {},
	"refreshtoken":  {},
	"api_key":       {},
	"apikey":        {},
	"x-api-key":     {},
	"x-auth-token":  {},
	"password":      {},
	"passwd":        {},
	"passphrase":    {},
	"pwd":           {},
	"secret":        {},
	"client_secret": {},
	"clientsecret":  {},
	"private_key":   {},
	"privatekey":    {},
	"signature":     {},
	// Cookies often carry session credentials.
	"cookie":     {},
	"set-cookie": {},
	// GitHub-specific token field names we use internally.
	"github_token": {},
	"githubtoken":  {},
	"gh_token":     {},
	"ghtoken":      {},
	"gh_pat":       {},
	"ghpat":        {},
}

// sanitizeEntry redacts credentials from a log entry in place. Message,
// stacktrace, and string-typed field values are scrubbed; fields whose key
// matches sensitiveFieldKeys are wholesale replaced.
func sanitizeEntry(entry *LogEntry) {
	entry.Message = sanitizeString(entry.Message)
	entry.Stacktrace = sanitizeString(entry.Stacktrace)
	if entry.Fields == nil {
		return
	}
	for k, v := range entry.Fields {
		if _, sensitive := sensitiveFieldKeys[strings.ToLower(k)]; sensitive {
			entry.Fields[k] = redacted
			continue
		}
		if s, ok := v.(string); ok {
			entry.Fields[k] = sanitizeString(s)
		}
	}
}

// sanitizeEntries applies sanitizeEntry to every entry in the slice.
func sanitizeEntries(entries []LogEntry) {
	for i := range entries {
		sanitizeEntry(&entries[i])
	}
}
