ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: help setup install-tools backend-install-tools frontend-install backend-dev frontend-dev dev format format-backend format-frontend lint lint-backend lint-frontend test test-backend test-frontend test-integration clean generate build run tray-icon build-macos build-macos-unsigned

.DEFAULT_GOAL := help

# Default target - show help
help:
	@echo "Octobud - Available Make Targets"
	@echo ""
	@echo "Setup & Installation:"
	@echo "  make setup              - Complete first-time setup"
	@echo "  make install-tools      - Install Go tools (goose, sqlc, mockgen)"
	@echo "  make generate           - Generate code (sqlc, mocks)"
	@echo "  make frontend-install   - Install frontend npm dependencies"
	@echo ""
	@echo "Development (hot reload):"
	@echo "  make backend-dev        - Run backend API on :8080"
	@echo "  make frontend-dev       - Run Vite dev server on :5173 (hot reload)"
	@echo "  make dev                - Alias for backend-dev"
	@echo ""
	@echo "  For full hot reload, run in two terminals:"
	@echo "    Terminal 1: make backend-dev"
	@echo "    Terminal 2: make frontend-dev"
	@echo "    Then open: http://localhost:5173"
	@echo ""
	@echo "Build:"
	@echo "  make build              - Build self-contained Octobud binary (macOS)"
	@echo "  make build-macos        - Build signed macOS .app + .pkg installer"
	@echo "  make build-macos-unsigned - Build unsigned macOS .app + .pkg (for testing)"
	@echo "  make run                - Build and run Octobud locally"
	@echo ""
	@echo "Utilities:"
	@echo "  make format             - Format Go and frontend code"
	@echo "  make lint               - Lint backend and frontend"
	@echo "  make lint-backend       - Lint backend only"
	@echo "  make lint-frontend      - Lint frontend only"
	@echo "  make test               - Run backend and frontend tests"
	@echo "  make test-backend       - Run backend tests only"
	@echo "  make test-frontend      - Run frontend tests only"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make tray-icon          - Regenerate menu bar icon from baby_octo.png"
	@echo "  make clean              - Remove built binaries"

# Complete first-time setup
setup: install-tools frontend-install
	@mkdir -p backend/web/dist
	@touch backend/web/dist/.gitkeep
	@echo ""
	@echo "✓ Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Build and run Octobud:"
	@echo "     make run"
	@echo ""
	@echo "  2. Or run in development mode:"
	@echo "     Terminal 1: make backend-dev"
	@echo "     Terminal 2: make frontend-dev"
	@echo ""
	@echo "  3. Open http://localhost:8808 (build) or http://localhost:5173 (dev)"
	@echo ""

install-tools: backend-install-tools

backend-install-tools:
	@echo "Installing Go tools..."
	$(MAKE) -C backend install-tools
	@echo "✓ Go tools installed"

frontend-install:
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "✓ Frontend dependencies installed"

backend-dev:
	cd backend && CGO_ENABLED=1 go run ./cmd/octobud --port 8080 --no-open --frontend-url http://localhost:5173

frontend-dev:
	cd frontend && npm run dev

# Full dev mode with hot reload (run in two terminals)
# Terminal 1: make backend-dev  (API on :8080, tray opens :5173)
# Terminal 2: make frontend-dev (Vite on :5173 with hot reload, proxies /api to :8080)
dev: backend-dev


format: format-backend format-frontend
	@echo ""
	@echo "✓ All formatting complete!"

format-backend:
	@echo "Formatting backend..."
	@cd backend && $(MAKE) format

format-frontend:
	@echo "Formatting frontend..."
	@cd frontend && npm run format
	@echo "✓ Frontend formatting complete"

lint: lint-backend lint-frontend
	@echo ""
	@echo "✓ All linting complete!"

lint-backend:
	@echo "Linting backend..."
	cd backend && $(MAKE) lint

lint-frontend:
	@echo "Linting frontend..."
	@cd frontend && npm run check
	@cd frontend && npm run lint || (echo "⚠ ESLint found issues" && exit 1)
	@echo "✓ Frontend linting complete"

test: test-backend test-frontend
	@echo ""
	@echo "✓ All tests passed!"

test-backend:
	@echo "Running backend tests..."
	cd backend && $(MAKE) test

test-frontend:
	@echo "Running frontend tests..."
	cd frontend && npm test

test-integration:
	@echo "Running integration tests..."
	cd backend && $(MAKE) test-integration

clean:
	rm -rf backend/bin bin
	find backend/web/dist -mindepth 1 ! -name '.gitkeep' -delete 2>/dev/null || true
	@echo "✓ Cleaned build artifacts"

# Build self-contained Octobud binary with embedded frontend
build: frontend-install
	@echo "Building Octobud..."
	@VERSION=$$(cat VERSION 2>/dev/null || echo ""); \
	if [ -z "$$VERSION" ]; then \
		VERSION=$$(git describe --tags --dirty --always 2>/dev/null | sed 's/^v//' || echo "dev"); \
	fi; \
	echo "  Version: $$VERSION"
	@echo "  Step 1: Building frontend..."
	@cd frontend && npm run build
	@echo "  Step 2: Copying frontend assets..."
	@mkdir -p backend/web/dist
	@cp -r frontend/build/* backend/web/dist/
	@echo "  Step 3: Building Go binary..."
	@mkdir -p bin
	@VERSION=$$(cat VERSION 2>/dev/null || echo ""); \
	if [ -z "$$VERSION" ]; then \
		VERSION=$$(git describe --tags --dirty --always 2>/dev/null | sed 's/^v//' || echo "dev"); \
	fi; \
	cd backend && CGO_ENABLED=1 go build -ldflags="-s -w -X github.com/octobud-hq/octobud/backend/internal/version.Version=$$VERSION" -o ../bin/octobud ./cmd/octobud
	@echo ""
	@echo "✓ Build complete!"
	@echo "  Binary: bin/octobud"
	@echo ""
	@echo "Run with: ./bin/octobud"

# Build and run Octobud locally
run: build
	@echo ""
	./bin/octobud

# Note: macOS has full support (menu bar, auto-start)
# Linux and Windows: core functionality works, but menu bar and auto-start are not available

generate:
	@echo "Generating code..."
	$(MAKE) -C backend generate
	@echo "✓ Code generation complete"

# Regenerate tray icon from baby_octo.png (22x22 for macOS menu bar)
tray-icon:
	@echo "Generating tray icon..."
	@sips -z 22 22 frontend/static/baby_octo.png --out backend/assets/tray_icon.png
	@echo "✓ Tray icon updated: backend/assets/tray_icon.png"

# Build signed macOS .app bundle and .pkg installer
# Requires: Developer ID Application and Developer ID Installer certificates
build-macos: frontend-install
	@echo "Building signed macOS app + installer..."
	./macos/build-local.sh

# Build unsigned macOS .app bundle and .pkg installer (for testing)
build-macos-unsigned: frontend-install
	@echo "Building unsigned macOS app + installer..."
	./macos/build-local.sh --skip-sign
