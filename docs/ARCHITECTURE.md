# Architecture Overview

This document provides a high-level overview of Octobud's architecture for developers.

## Overview

Octobud is a desktop application for managing GitHub notifications. It consists of:

- **Backend**: Go-based HTTP server with SQLite database
- **Frontend**: SvelteKit web application
- **Desktop Integration**: macOS menu bar integration (via systray)

The application runs as a single binary with an embedded frontend, accessible via a local web interface.

## Project Structure

```
octobud/
├── backend/              # Go backend
│   ├── cmd/octobud/     # Main entry point
│   ├── internal/
│   │   ├── api/         # HTTP handlers
│   │   ├── core/        # Business logic services
│   │   ├── db/          # Database layer (SQLite)
│   │   ├── github/      # GitHub API client
│   │   ├── jobs/        # Background job scheduler
│   │   ├── query/       # Query language parser/evaluator
│   │   ├── sync/        # Sync service
│   │   └── tray/        # System tray integration
│   └── web/             # Embedded frontend assets
├── frontend/            # SvelteKit frontend
│   └── src/
│       ├── lib/         # Components and utilities
│       └── routes/      # SvelteKit routes
└── docs/                # Documentation
```

## Backend Architecture

### Database Layer

- **SQLite**: Single-file database for local storage
- **sqlc**: Generates type-safe Go code from SQL queries
- **goose**: Database migrations

Key tables:
- `notifications`: GitHub notification data
- `repositories`: Repository information
- `pull_requests`: PR metadata
- `tags`: User-defined tags
- `views`: Saved filter views
- `rules`: Automation rules
- `users`: User configuration and sync settings

### API Layer

- **Router**: Chi router handling `/api/*` endpoints
- **Handlers**: HTTP handlers in `internal/api/*`
  - `notifications/`: Notification CRUD operations
  - `oauth/`: GitHub OAuth device flow
  - `user/`: User settings and token management
  - `tags/`, `views/`, `rules/`: Resource management
  - `sync/`: Sync operations

### Core Services

- **TokenManager**: Manages GitHub token storage (Keychain on macOS, encrypted SQLite elsewhere)
- **SyncService**: Handles syncing notifications from GitHub API
- **JobScheduler**: Background job queue for async operations
- **QueryEngine**: Parses and evaluates notification queries

### GitHub Integration

- **OAuth Device Flow**: For user authentication
- **Personal Access Tokens**: Alternative auth method
- **API Client**: Wraps GitHub REST API calls
- **Rate Limiting**: Respects GitHub rate limits

### Query Language

Custom query language for filtering notifications:
- Parsed into AST (`internal/query/parse/`)
- Evaluated against notifications (`internal/query/eval/`)
- Converted to SQL for efficient database queries (`internal/query/sql/`)

Examples:
- `is:unread type:PullRequest`
- `repo:owner/name reason:mention`
- `author:dependabot`

## Frontend Architecture

### SvelteKit

- **Framework**: Svelte 5 with SvelteKit
- **Routing**: File-based routing in `routes/`
- **State Management**: Svelte stores for global state
- **Styling**: Tailwind CSS

### Key Components

- **NotificationView**: Main notification list view
- **InlineDetailView**: Side panel for notification details
- **QueryInput**: Search/filter input with autocomplete
- **Settings**: User settings management

### State Management

- **Stores**: Svelte stores in `src/lib/stores/`
  - `notificationStore`: Notification data
  - `viewStore`: Current view and navigation
  - `settingsStore`: User preferences

### API Client

- Type-safe API client in `src/lib/api/`
- Handles authentication, error handling, and request/response serialization

## Data Flow

### Initial Sync

1. User authenticates via OAuth or PAT
2. Token stored securely (Keychain or encrypted SQLite)
3. Initial sync fetches notifications from GitHub API
4. Data stored in local SQLite database
5. Frontend queries local database for fast access

### Ongoing Sync

1. Background job periodically syncs new/updated notifications
2. Changes pushed to frontend via Server-Sent Events (SSE)
3. Frontend updates stores and UI reactively

### User Actions

1. User performs action (archive, star, etc.)
2. Frontend sends API request
3. Backend updates database and GitHub API
4. Response sent back to frontend
5. UI updates reactively

## Security Considerations

### Token Storage

- **macOS**: Stored in macOS Keychain (`keybase/go-keychain`)
- **Linux/Windows**: AES-256-GCM encrypted in SQLite
  - Encryption key stored in `.token_key` file (0600 permissions)
  - Note: Key stored alongside database (improvement needed for future)

### Input Validation

- **SQL Injection**: All queries use parameterized statements (via sqlc)
- **XSS**: Frontend uses DOMPurify for sanitizing markdown content
- **Query Parsing**: Custom parser prevents injection in query language

### Network Security

- **HTTPS**: Local-only HTTP server (localhost)
- **Certificate Validation**: All GitHub API calls verify certificates
- **Rate Limiting**: Respects GitHub API rate limits

## Platform Support

### macOS

- Full support with menu bar integration
- Auto-start capability
- Secure Keychain storage
- Native app bundle (.app)

### Linux & Windows

- Core functionality works
- Menu bar integration not available
- Auto-start requires manual configuration
- Encrypted token storage (local key)

## Development

### Building

```bash
# Build all
make build

# Development mode (hot reload)
make backend-dev    # Terminal 1
make frontend-dev   # Terminal 2
```

### Database Migrations

```bash
cd backend
goose -dir internal/db/sqlite/migrations sqlite3 path/to/db.db up
```

### Code Generation

```bash
make generate  # Runs sqlc and mockgen
```

## Dependencies

### Backend

- Go 1.24+
- SQLite (via modernc.org/sqlite)
- Chi (HTTP router)
- Zap (logging)
- systray (menu bar)

### Frontend

- Node.js 24+
- Svelte 5
- SvelteKit
- Tailwind CSS
- DOMPurify (XSS prevention)

## Performance Considerations

- **Local Database**: Fast queries without network latency
- **Caching**: Repository and PR data cached locally
- **Incremental Sync**: Only fetches new/updated notifications
- **Lazy Loading**: Frontend loads data as needed
- **Background Jobs**: Heavy operations run asynchronously

## Future Improvements

- Better token security on Linux/Windows (use system keychains)
- Multi-user support
- GitHub Enterprise support
- Cross-platform menu bar integration

