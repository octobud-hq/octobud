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

// Package server provides common HTTP server setup utilities shared between entrypoints.
package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
)

// Config holds server configuration options.
type Config struct {
	// RequestLogging enables chi's request logger middleware (typically for development/production server)
	RequestLogging bool

	// CORSOrigins is a list of allowed CORS origins. If empty, defaults to localhost origins.
	CORSOrigins []string
}

// DefaultConfig returns a Config with sensible defaults for local development.
func DefaultConfig() Config {
	return Config{
		RequestLogging: false,
		CORSOrigins:    []string{"http://localhost:*", "http://127.0.0.1:*"},
	}
}

// NewRouter creates a new chi router with standard middleware configured.
func NewRouter(cfg Config) chi.Router {
	router := chi.NewRouter()

	// Core middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	// Optional request logging
	if cfg.RequestLogging {
		router.Use(middleware.Logger)
	}

	// Security middleware
	router.Use(shared.SecurityHeadersMiddleware)
	router.Use(shared.BodyLimitMiddleware(shared.DefaultMaxBodySize))

	// CORS
	origins := cfg.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:*", "http://127.0.0.1:*"}
	}
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler)

	// Health check
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return router
}

// NewHTTPServer creates an http.Server with standard timeout configuration.
func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// ParseCORSOrigins parses a comma-separated list of CORS origins from a string.
// Returns default localhost origins if the input is empty.
func ParseCORSOrigins(origins string) []string {
	if origins == "" {
		return []string{
			"http://localhost:5173", // Frontend dev server
			"http://localhost:8808", // Default production port
			"http://localhost:8080", // Backend dev mode
		}
	}

	var result []string
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			result = append(result, origin)
		}
	}
	return result
}

// CORSOriginsFromEnv returns CORS origins from the CORS_ALLOWED_ORIGINS environment variable.
func CORSOriginsFromEnv() []string {
	return ParseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
}
