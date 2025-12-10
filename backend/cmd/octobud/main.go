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

// Package main provides the unified entry point for Octobud - a self-contained
// binary that runs the HTTP server with embedded frontend and background sync.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pressly/goose/v3"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/octobud-hq/octobud/backend/internal/api"
	"github.com/octobud-hq/octobud/backend/internal/api/navigation"
	config "github.com/octobud-hq/octobud/backend/internal/config"
	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
	coregithub "github.com/octobud-hq/octobud/backend/internal/core/github"
	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	"github.com/octobud-hq/octobud/backend/internal/core/pullrequest"
	"github.com/octobud-hq/octobud/backend/internal/core/repository"
	"github.com/octobud-hq/octobud/backend/internal/core/syncstate"
	"github.com/octobud-hq/octobud/backend/internal/core/update"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/sqlite"
	"github.com/octobud-hq/octobud/backend/internal/github"
	"github.com/octobud-hq/octobud/backend/internal/jobs"
	"github.com/octobud-hq/octobud/backend/internal/osactions"
	"github.com/octobud-hq/octobud/backend/internal/query"
	"github.com/octobud-hq/octobud/backend/internal/server"
	"github.com/octobud-hq/octobud/backend/internal/sync"
	"github.com/octobud-hq/octobud/backend/internal/tray"
	"github.com/octobud-hq/octobud/backend/internal/xcrypto"
	"github.com/octobud-hq/octobud/backend/web"

	// SQLite driver
	_ "modernc.org/sqlite"

	"github.com/octobud-hq/octobud/backend/internal/version"
)

const indexHTML = "index.html"

// appConfig holds the parsed command-line configuration.
type appConfig struct {
	port        int
	dataDir     string
	noOpen      bool
	frontendURL string // URL to open in browser (defaults to http://localhost:<port>)
}

func main() {
	// Parse command-line flags
	port := flag.Int("port", 8808, "Port to listen on")
	dataDir := flag.String("data-dir", "", "Data directory (default: platform-specific)")
	noOpen := flag.Bool("no-open", false, "Don't open browser on startup")
	frontendURL := flag.String(
		"frontend-url",
		"",
		"Frontend URL for tray/browser (default: http://localhost:<port>)",
	)
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Octobud %s\n", version.Get())
		os.Exit(0)
	}

	// Determine data directory
	if *dataDir == "" {
		var err error
		*dataDir, err = db.GetDefaultDataDir()
		if err != nil {
			log.Fatalf("Failed to get default data directory: %v", err)
		}
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(*dataDir, 0700); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Set up file logging with rotation
	logDir := filepath.Join(*dataDir, "logs")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}
	logPath := filepath.Join(logDir, "octobud.log")
	logWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,   // MB before rotation
		MaxBackups: 3,    // old files to keep
		MaxAge:     7,    // days to keep old files
		Compress:   true, // gzip rotated files
	}
	// Write to both file and stdout
	log.SetOutput(io.MultiWriter(os.Stdout, logWriter))

	// Default frontend URL to localhost:<port>
	fURL := *frontendURL
	if fURL == "" {
		fURL = fmt.Sprintf("http://localhost:%d", *port)
	}

	cfg := appConfig{
		port:        *port,
		dataDir:     *dataDir,
		noOpen:      *noOpen,
		frontendURL: fURL,
	}

	// Create a logger for tray operations that writes to both console and logfile
	// This ensures tray operations and early initialization are logged for debugging
	trayLogger := config.NewConsoleLoggerWithFile(logWriter)

	// On macOS, RunWithApp blocks and runs the tray event loop on the main thread.
	// The onReady callback runs our server logic.
	// On other platforms, RunWithApp just calls onReady directly.
	tray.RunWithApp(cfg.frontendURL, trayLogger, func(t *tray.Tray) {
		// Start the server in a goroutine so tray can run its event loop
		go runServer(cfg, t, logWriter)
	}, nil)
}

// runServer contains all the server initialization and lifecycle logic.
//
//nolint:gocyclo // cyclomatic complexity is high but function is cohesive
func runServer(cfg appConfig, trayApp *tray.Tray, logWriter *lumberjack.Logger) {
	// Print startup banner
	fmt.Println()
	fmt.Println("  🐙 Octobud")
	fmt.Printf("     Version: %s\n", version.Get())
	fmt.Printf("     Data: %s\n", cfg.dataDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wire up tray quit to cancel context
	if trayApp != nil {
		trayApp.SetOnQuit(func() {
			fmt.Println("\nShutting down via menu...")
			cancel()
		})
	}

	// Set up signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Set up database path
	dsn := filepath.Join(cfg.dataDir, "octobud.db")
	fmt.Printf("     Database: %s\n", dsn)

	// Open database
	dbConn, err := db.OpenDatabase(dsn)
	if err != nil {
		cancel()
		//nolint:gocritic // exitAfterDefer: cancel() is called explicitly before log.Fatalf
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if closeErr := dbConn.Close(); closeErr != nil {
			log.Printf("Error closing database: %v", closeErr)
		}
	}()

	// Run migrations
	if err = runMigrations(dbConn); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize logger (console format for desktop app)
	// Reuse the same logWriter that was created in main() for consistency
	logger := config.NewConsoleLoggerWithFile(logWriter)

	// Create store
	store := db.NewStore(dbConn)

	// Initialize auth service and ensure user record exists
	authService := authsvc.NewService(store)
	if err = authService.EnsureUser(ctx); err != nil {
		log.Fatalf("Failed to ensure user exists: %v", err)
	}

	// Initialize token storage
	// NewKeychain() returns platform-specific implementation via build tags:
	//   - macOS: real keychain implementation
	//   - Non-macOS: fallback that returns errors (tokens stored in SQLite instead)
	// Both platforms also get an encryptor for SQLite fallback/legacy support
	var encryptor *xcrypto.Encryptor
	var keychain = xcrypto.NewKeychain()
	encryptionKey, err := xcrypto.LoadOrGenerateKey(cfg.dataDir)
	if err != nil {
		log.Fatalf("Failed to load/generate encryption key: %v", err)
	}
	encryptor, err = xcrypto.NewEncryptor(encryptionKey)
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	// Initialize GitHub client
	githubClient := github.NewClient()

	// Create token manager
	tokenManager := coregithub.NewTokenManager(
		store,
		encryptor,
		keychain,
		githubClient,
		authService,
		logger,
	)

	// Initialize token manager (loads stored token)
	if initErr := tokenManager.Initialize(ctx); initErr != nil {
		// Don't fatal on token init failure - user can configure via UI
		log.Printf("Warning: Failed to initialize GitHub token: %v", initErr)
	}

	// Check if token is configured
	tokenConfigured := tokenManager.IsConnected()

	// Initialize business logic services
	syncStateSvc := syncstate.NewSyncStateService(store)
	repositorySvc := repository.NewService(store)
	pullRequestSvc := pullrequest.NewService(store)
	notificationSvc := notification.NewService(store)

	// Initialize sync service
	syncService := sync.NewService(
		logger,
		time.Now,
		githubClient,
		syncStateSvc,
		repositorySvc,
		pullRequestSvc,
		notificationSvc,
		store,
	)

	// Create update service
	updateService := update.NewService(logger)

	// Create scheduler with persistent job queue
	scheduler := jobs.NewSQLiteScheduler(jobs.SQLiteSchedulerConfig{
		Logger:        logger,
		DBConn:        dbConn, // For persistent job queue
		Store:         store,
		SyncService:   syncService,
		SyncInterval:  20 * time.Second,
		AuthService:   authService,
		UpdateService: updateService,
	})

	// Start scheduler
	if startErr := scheduler.Start(ctx); startErr != nil {
		log.Fatalf("Failed to start scheduler: %v", startErr)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if stopErr := scheduler.Stop(shutdownCtx); stopErr != nil {
			log.Printf("Warning: scheduler shutdown error: %v", stopErr)
		}
	}()

	// Create navigation broadcaster for tray menu navigation
	navBroadcaster := navigation.NewBroadcaster(logger)

	// Configure API handler
	// Always set up sync service - OAuth can configure the token later
	opts := []api.HandlerOption{
		api.WithScheduler(scheduler),
		api.WithTokenManager(tokenManager),
		api.WithSyncService(store, githubClient, logger),
		api.WithNavigationBroadcaster(navBroadcaster),
	}
	if tokenConfigured {
		status := tokenManager.GetStatus()
		fmt.Printf("     GitHub: Connected as %s ✓\n", status.GitHubUsername)
	} else {
		fmt.Println("     GitHub: Not configured (configure in app)")
	}
	apiHandler := api.NewHandler(store, opts...)

	// Pass navigation broadcaster to tray if available
	if trayApp != nil {
		navBroadcasterFromHandler := apiHandler.GetNavigationBroadcaster()
		if navBroadcasterFromHandler != nil {
			trayApp.SetNavigationBroadcaster(navBroadcasterFromHandler)
		}
	}

	// Set up router with standard middleware
	router := server.NewRouter(server.DefaultConfig())

	// API routes
	router.Route("/api", apiHandler.RegisterAllRoutes)

	// Serve embedded frontend
	frontendFS, err := web.DistFS()
	if err != nil {
		log.Fatalf("Failed to load frontend assets: %v", err)
	}
	serveStaticFiles(router, frontendFS)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.port)
	httpServer := server.NewHTTPServer(addr, router)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	fmt.Println()
	fmt.Printf("     API: http://localhost:%d\n", cfg.port)
	fmt.Printf("     Frontend: %s\n", cfg.frontendURL)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop (or use menu bar icon to quit).")
	fmt.Println()

	// Auto-open browser to frontend URL
	if !cfg.noOpen {
		// Give the server a moment to start
		time.Sleep(500 * time.Millisecond)
		openBrowser(cfg.frontendURL)
	}

	// Set up a ticker to periodically update tray with sync status
	if trayApp != nil {
		// Helper to update tray status
		updateTrayStatus := func() {
			// Get userID for data scoping
			user, userErr := authService.GetUser(ctx)
			if userErr != nil || user == nil || user.GithubUserID == "" {
				// User not configured yet, skip tray update
				return
			}
			userID := user.GithubUserID

			// Update last sync time
			state, err := syncStateSvc.GetSyncState(ctx, userID)
			if err == nil && state.LastSuccessfulPoll.Valid {
				trayApp.SetLastSync(state.LastSuccessfulPoll.Time)
			}
			// Update unread count
			unreadCount, err := getUnreadCount(ctx, store, userID)
			if err == nil {
				trayApp.SetUnreadCount(int(unreadCount))
			}
		}

		// Update immediately on startup
		updateTrayStatus()

		// Then update periodically
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					updateTrayStatus()
				}
			}
		}()
	}

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		// Signal received, shutdown gracefully
	case err := <-serverErr:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}

	// Stop the tray
	if trayApp != nil {
		trayApp.Quit()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Warning: server shutdown error: %v", err)
	}

	fmt.Println("Goodbye!")
}

// runMigrations runs database migrations using goose.
func runMigrations(dbConn *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}
	goose.SetBaseFS(sqlite.MigrationsFS)
	if err := goose.Up(dbConn, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// serveStaticFiles serves the embedded frontend files with proper SPA routing.
func serveStaticFiles(router chi.Router, frontendFS fs.FS) {
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = indexHTML
		}

		// Check if file exists
		f, err := frontendFS.Open(path)
		if err != nil {
			// File doesn't exist, serve index.html for SPA routing
			path = indexHTML
		} else {
			if closeErr := f.Close(); closeErr != nil {
				log.Printf("Error closing file: %v", closeErr)
			}
		}

		// Open the file
		file, err := frontendFS.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				log.Printf("Error closing file: %v", closeErr)
			}
		}()

		// Get file info
		stat, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Set cache headers
		if strings.HasPrefix(path, "_app/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else if path == indexHTML || path == "sw.js" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}

		// Serve the file
		if seeker, ok := file.(io.ReadSeeker); ok {
			http.ServeContent(w, r, path, stat.ModTime(), seeker)
		} else {
			content, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			setContentType(w, path)
			if _, err := w.Write(content); err != nil {
				log.Printf("Error writing response: %v", err)
			}
		}
	})
}

// setContentType sets the Content-Type header based on file extension.
func setContentType(w http.ResponseWriter, path string) {
	switch {
	case strings.HasSuffix(path, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(path, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(path, ".json"):
		w.Header().Set("Content-Type", "application/json")
	case strings.HasSuffix(path, ".woff2"):
		w.Header().Set("Content-Type", "font/woff2")
	case strings.HasSuffix(path, ".woff"):
		w.Header().Set("Content-Type", "font/woff")
	}
}

// openBrowser opens the URL in the default browser.
// On macOS, it first checks for existing tabs and activates/refreshes them if found.
func openBrowser(targetURL string) {
	ctx := context.Background()
	var err error

	switch runtime.GOOS {
	case "darwin":
		// Try to find and activate existing tab first
		// Navigate to baseURL (without path) to force a refresh
		osActionsSvc := osactions.NewService()
		baseURL := targetURL
		if parsedURL, parseErr := url.Parse(targetURL); parseErr == nil {
			baseURL = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
		}

		// Try Safari first - navigate to baseURL to force refresh
		found, tabErr := osActionsSvc.ActivateBrowserTab(
			ctx,
			"safari",
			baseURL,
			baseURL,
		)
		if tabErr != nil {
			log.Printf("Failed to activate Safari tab: %v", tabErr)
		} else if found {
			log.Printf("Activated and refreshed existing Safari tab")
			return
		}

		// Try Chrome - navigate to baseURL to force refresh
		found, tabErr = osActionsSvc.ActivateBrowserTab(
			ctx,
			"chrome",
			baseURL,
			baseURL,
		)
		if tabErr != nil {
			log.Printf("Failed to activate Chrome tab: %v", tabErr)
		} else if found {
			log.Printf("Activated and refreshed existing Chrome tab")
			return
		}

		// No existing tab found, open new one
		err = exec.CommandContext(ctx, "open", targetURL).Start()
	case "linux":
		err = exec.CommandContext(ctx, "xdg-open", targetURL).Start()
	case "windows":
		err = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", targetURL).Start()
	}

	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

// getUnreadCount returns the count of unread notifications in the inbox.
// Uses the query engine to match the "in:inbox is:unread" query criteria,
// ensuring consistency with how the rest of the application calculates unread counts.
func getUnreadCount(ctx context.Context, store db.Store, userID string) (int64, error) {
	// Use the same query string as calculateInboxUnreadCount in the view service
	queryStr := "in:inbox is:unread"

	dbQuery, err := query.BuildQuery(queryStr, 1, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	result, err := store.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to list notifications: %w", err)
	}

	return result.Total, nil
}
