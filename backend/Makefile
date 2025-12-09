# Get Go bin directory for tools installed via go install
# GOBIN takes precedence over GOPATH/bin if set
GOBIN := $(shell go env GOBIN)
GOPATH_BIN := $(shell go env GOPATH)/bin
GO_TOOL_BIN := $(if $(GOBIN),$(GOBIN),$(GOPATH_BIN))
SQLC := $(GO_TOOL_BIN)/sqlc
MOCKGEN := $(GO_TOOL_BIN)/mockgen
GOOSE := $(GO_TOOL_BIN)/goose
GOLANGCI_LINT := $(GO_TOOL_BIN)/golangci-lint

.PHONY: install-tools dev test test-integration lint format generate-sqlc generate-mocks generate

# Pinned tool versions - update these when upgrading tools
GOOSE_VERSION := v3.26.0
SQLC_VERSION := v1.30.0
MOCKGEN_VERSION := v0.6.0
GOLANGCI_LINT_VERSION := v2.6.2

install-tools:
	@echo "Installing Go tooling..."
	go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

dev:
	CGO_ENABLED=1 go run ./cmd/octobud --no-open

test:
	@mkdir -p web/dist
	@touch web/dist/.gitkeep
	go test ./...

# Run integration tests
test-integration:
	@mkdir -p web/dist
	@touch web/dist/.gitkeep
	@echo "Running integration tests..."
	CGO_ENABLED=1 go test -tags=test,integration -v ./integration/...

lint:
	@echo "Running golangci-lint..."
	@if ! command -v $(GOLANGCI_LINT) > /dev/null; then \
		echo "Error: golangci-lint is not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run
	@echo "✓ Linting complete"

format:
	@echo "Formatting with golangci-lint --fix..."
	@if ! command -v $(GOLANGCI_LINT) > /dev/null; then \
		echo "Error: golangci-lint is not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run --fix
	@echo "✓ Formatting complete"

generate-sqlc:
	$(SQLC) generate -f internal/db/sqlite/sqlc.yaml

generate-mocks:
	@echo "Generating mocks..."
	PATH="$(GO_TOOL_BIN):$$PATH" go generate

generate: generate-sqlc generate-mocks
	@echo "Code generation complete"
