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

package shared

import (
	"net/http"
	"strings"
)

// SecurityHeadersMiddleware adds security headers to all HTTP responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking - allow same-origin for embedding (good for local/home lab deployments)
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Enable XSS filtering in older browsers (legacy but harmless)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content Security Policy
		// Allow same-origin, inline styles/scripts (required for SvelteKit hydration),
		// data URIs for images/icons, and GitHub domains for profile pictures
		// Images load via <img> tags (controlled by img-src), not via fetch() (connect-src)
		// GitHub avatars are served from:
		// - github.com/username.png (redirects to avatar)
		// - avatars.githubusercontent.com/u/<user_id> (direct avatar URLs from API)
		// - dependabot-badges.githubapp.com (Dependabot status badges in markdown content)
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https://github.com https://avatars.githubusercontent.com https://*.githubusercontent.com https://dependabot-badges.githubapp.com; " + //nolint:lll // URLs are long.
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"frame-ancestors 'self'; " +
			"base-uri 'self'; " +
			"form-action 'self'"
		w.Header().Set("Content-Security-Policy", csp)

		// Strict Transport Security - only set if request is HTTPS or forwarded as HTTPS
		// This prevents accidental use in HTTP-only deployments
		if IsHTTPS(r) {
			// 1 year HSTS max-age with includeSubDomains and preload
			// For local/home lab, we use a shorter duration to allow flexibility
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Referrer Policy - control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (formerly Feature Policy)
		// Restrict access to browser features
		permissionsPolicy := "geolocation=(), microphone=(), camera=(), payment=(), " +
			"usb=(), magnetometer=(), gyroscope=()"
		w.Header().Set("Permissions-Policy", permissionsPolicy)

		next.ServeHTTP(w, r)
	})
}

// IsHTTPS determines if the request is over HTTPS by checking:
// 1. The X-Forwarded-Proto header (set by reverse proxies)
// 2. The TLS connection state
// 3. The X-Forwarded-Ssl header (some proxies use this)
// 4. The Forwarded header (RFC 7239)
//
// This is useful for setting Secure cookies and HSTS headers.
func IsHTTPS(r *http.Request) bool {
	// Check X-Forwarded-Proto header (set by reverse proxies like nginx, Caddy, etc.)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		return true
	}

	// Check if request was made over TLS
	if r.TLS != nil {
		return true
	}

	// Check X-Forwarded-Ssl header (some proxies use this)
	if r.Header.Get("X-Forwarded-Ssl") == "on" {
		return true
	}

	// Check Forwarded header (RFC 7239)
	forwarded := r.Header.Get("Forwarded")
	return strings.Contains(strings.ToLower(forwarded), "proto=https")
}
