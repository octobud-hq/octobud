# Contributing to Octobud

Thank you for your interest in contributing to Octobud! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Code Style](#code-style)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

This project follows our [Code of Conduct](../CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Types of Contributions

We welcome many types of contributions:

- **Bug fixes** - Fix issues in the codebase
- **Features** - Add new functionality
- **Documentation** - Improve or add documentation
- **Tests** - Improve test coverage
- **Bug reports** - Report issues you encounter
- **Feature requests** - Suggest new features

### Before You Start

1. Check if there's an existing issue for what you want to work on
2. For significant changes, open an issue first to discuss the approach
3. For bugs, check if it's already been reported

## Project Structure

```
octobud/
├── backend/           # Go backend (runs locally)
│   ├── cmd/
│   │   └── octobud/  # Main entry point (single binary)
│   ├── internal/     # Internal packages
│   │   ├── api/      # HTTP handlers
│   │   ├── core/     # Business logic services
│   │   ├── db/       # Database layer
│   │   │   └── sqlite/ # SQLite queries (sqlc generated)
│   │   ├── github/   # GitHub API client
│   │   ├── jobs/     # Background job scheduler
│   │   ├── sync/     # Sync service
│   │   └── query/    # Query language parser
│   └── web/          # Embedded frontend assets
├── frontend/         # Svelte 5 + SvelteKit frontend
│   └── src/
│       ├── lib/      # Components and utilities
│       └── routes/   # SvelteKit routes
└── docs/             # Documentation
```

For a detailed architecture overview, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Development Setup

### Prerequisites

- **Go** 1.21+
- **Node.js** 18+ and npm
- **GitHub account** - OAuth is the preferred authentication method (recommended). Alternatively, you can use a Personal Access Token with `notifications` and `repo` scopes ([create one here](https://github.com/settings/tokens))

No external dependencies required - Octobud uses SQLite and runs as a single desktop application binary.

### Setup

```bash
# Clone and setup
git clone https://github.com/octobud-hq/octobud.git
cd octobud
make setup
```

This installs Go tools and npm dependencies.

### Start Development Servers

Run in separate terminals:

```bash
make backend-dev    # Terminal 1 - Backend runs locally (port 8080 for dev)
make frontend-dev   # Terminal 2 - Frontend with hot reload (port 5173)
```

Open http://localhost:5173 and connect your GitHub account. OAuth is the preferred method (recommended), or you can configure a Personal Access Token in the app (Settings → Account).

### All Available Commands

| Command | Description |
|---------|-------------|
| `make setup` | First-time setup (install tools, deps) |
| `make install-tools` | Install Go tools (goose, sqlc, mockgen) |
| `make frontend-install` | Install frontend npm dependencies |
| `make generate` | Generate code (sqlc queries, mocks) |
| `make backend-dev` | Run backend locally (for development) |
| `make frontend-dev` | Run frontend dev server locally |
| `make dev` | Run backend locally (alias) |
| `make build` | Build desktop app binary (macOS) |
| `make build-macos` | Build signed macOS .app + .pkg installer |
| `make run` | Build and run locally |
| `make format` | Format all code (backend and frontend) |
| `make lint` | Lint all code (backend and frontend) |
| `make test` | Run all tests (backend and frontend) |
| `make test-integration` | Run integration tests |
| `make clean` | Remove built binaries |

## Making Changes

### Branch Naming

Use descriptive branch names:

- `feature/add-snooze-reminder`
- `fix/notification-sync-error`
- `docs/update-readme`
- `refactor/query-parser`

### Commit Messages

Write clear, concise commit messages:

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain what and why, not how.

Fixes #123
```

### Code Changes

1. Create a branch from `main`
2. Make your changes
3. Add or update tests as needed
4. Ensure all tests pass
5. Update documentation if needed

## Submitting a Pull Request

### Before Submitting

- [ ] Code follows the style guidelines
- [ ] Tests pass locally (`make test`)
- [ ] New code has appropriate test coverage
- [ ] Documentation is updated if needed
- [ ] Commit messages are clear and descriptive

### PR Description

Include in your PR description:

1. **What** - Brief description of the change
2. **Why** - Motivation for the change
3. **How** - High-level approach (if not obvious)
4. **Testing** - How you tested the changes
5. **Screenshots** - For UI changes

### Review Process

1. A maintainer will review your PR
2. Address any feedback
3. Once approved, a maintainer will merge

## Code Style

Before committing, run from the project root:

```bash
make format    # Format all code
make lint      # Check for issues
```

### Go (Backend)

- Follow standard Go conventions
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors explicitly

### TypeScript/Svelte (Frontend)

- Follow the existing code style
- Use TypeScript for type safety
- Prefer composition over inheritance
- Keep components focused and small

### General

- Keep functions small and focused
- Prefer clarity over cleverness
- Add comments for complex logic
- Remove commented-out code

## Testing

Run all tests from the project root:

```bash
make test             # Run all tests (backend + frontend)
make test-backend     # Backend only
make test-frontend    # Frontend only
make test-integration # Integration tests
```

### Test Guidelines

- Write tests for new features
- Add regression tests for bug fixes
- Aim for meaningful coverage, not 100%
- Use table-driven tests in Go where appropriate
- Please do manual testing of new features and check for regressions if touching an existing feature ;)

## Documentation

### When to Update Docs

- Adding new features
- Changing existing behavior
- Adding configuration options
- Changing API endpoints or routes

### Documentation Structure

```
docs/
├── README.md              # Documentation index
├── start-here.md          # Initial setup and core workflows
├── installation.md          # Installation and configuration
├── guides/                # User guides and reference
│   ├── query-syntax.md
│   ├── keyboard-shortcuts.md
│   └── views-and-rules.md
├── concepts/              # How things work under the hood
│   ├── action-hints.md
│   ├── query-engine.md
│   └── sync.md
└── security/              # Security documentation
    └── ...
```

## Questions?

- Open a [Discussion](https://github.com/octobud-hq/octobud/discussions) for questions
- Join our community chat (if available)
- Check existing issues and discussions

Thank you for contributing! 🎉
