# Sietch Vault Testing and Build Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=sietch
BINARY_UNIX=$(BINARY_NAME)_unix

# Test parameters
TEST_TIMEOUT=30m
TEST_VERBOSE=-v
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out
COVERAGE_HTML=$(COVERAGE_DIR)/coverage.html

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./main.go

# Build for Unix
build-unix:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./main.go

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -rf $(COVERAGE_DIR)
	rm -rf test_vaults/

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run all tests
test:
	$(GOTEST) $(TEST_VERBOSE) -timeout $(TEST_TIMEOUT) ./...

# Run tests with race detection
test-race:
	$(GOTEST) $(TEST_VERBOSE) -race -timeout $(TEST_TIMEOUT) ./...

# Run tests with coverage
test-coverage:
	mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(TEST_VERBOSE) -timeout $(TEST_TIMEOUT) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Run tests with coverage and open report in browser
test-coverage-view: test-coverage
	@if command -v xdg-open > /dev/null 2>&1; then \
		xdg-open $(COVERAGE_HTML); \
	elif command -v open > /dev/null 2>&1; then \
		open $(COVERAGE_HTML); \
	else \
		echo "Coverage report available at: $(COVERAGE_HTML)"; \
	fi

# Run only unit tests
test-unit:
	$(GOTEST) $(TEST_VERBOSE) -timeout $(TEST_TIMEOUT) -short ./...

# Run only integration tests
test-integration:
	$(GOTEST) $(TEST_VERBOSE) -timeout $(TEST_TIMEOUT) -run Integration ./...

# Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Run tests for specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-pkg PKG=./internal/encryption"; \
		exit 1; \
	fi
	$(GOTEST) $(TEST_VERBOSE) -timeout $(TEST_TIMEOUT) $(PKG)

# Run tests with coverage for specific package
test-pkg-coverage:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-pkg-coverage PKG=./internal/encryption"; \
		exit 1; \
	fi
	mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(TEST_VERBOSE) -timeout $(TEST_TIMEOUT) -coverprofile=$(COVERAGE_DIR)/pkg_coverage.out $(PKG)
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/pkg_coverage.out -o $(COVERAGE_DIR)/pkg_coverage.html

# Lint code
lint:
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.60.3"; \
	fi

# Format code
fmt:
	$(GOCMD) fmt ./...

# Vet code
vet:
	$(GOCMD) vet ./...

# Run all quality checks
check: fmt vet lint test-race test-coverage

# Install the binary
install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# Install to local bin (for development)
install-local: build
	cp $(BINARY_NAME) /home/nilay/.local/bin/$(BINARY_NAME)
	@echo "✅ Sietch installed to ~/.local/bin/"

# Create test vaults for integration testing
create-test-vaults:
	@echo "Creating test vaults..."
	@mkdir -p test_vaults
	@./$(BINARY_NAME) init --name test-vault-aes --path test_vaults --key-type aes --passphrase-value testpass123 --force 2>/dev/null || echo "AES test vault creation failed (binary may not be built)"
	@./$(BINARY_NAME) init --name test-vault-gpg --path test_vaults --key-type gpg --force 2>/dev/null || echo "GPG test vault creation failed (binary may not be built)"

# Clean test vaults
clean-test-vaults:
	rm -rf test_vaults/

# Run security audit
security-audit:
	@if command -v gosec > /dev/null 2>&1; then \
		gosec -exclude=G301,G302,G304,G306 ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

# Show test coverage summary
coverage-summary:
	@if [ -f $(COVERAGE_FILE) ]; then \
		$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total; \
	else \
		echo "No coverage file found. Run 'make test-coverage' first."; \
	fi

# Check version consistency between local and CI
check-versions:
	@./scripts/check-versions.sh

# CI pipeline: run all checks and tests
ci: deps check security-audit

# Development workflow: format, test, and build
dev: fmt test build

# Release workflow: clean, format, test with coverage, and build
release: clean fmt test-coverage build

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build the binary"
	@echo "  build-unix         - Build for Unix/Linux"
	@echo "  clean              - Clean build artifacts and test data"
	@echo "  deps               - Download and tidy dependencies"
	@echo "  test               - Run all tests"
	@echo "  test-race          - Run tests with race detection"
	@echo "  test-coverage      - Run tests with coverage reporting"
	@echo "  test-coverage-view - Run tests with coverage and open report"
	@echo "  test-unit          - Run only unit tests (short)"
	@echo "  test-integration   - Run only integration tests"
	@echo "  test-pkg           - Run tests for specific package (PKG=./path/to/pkg)"
	@echo "  bench              - Run benchmarks"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo "  vet                - Vet code"
	@echo "  check              - Run all quality checks"
	@echo "  install            - Install binary to GOPATH"
	@echo "  create-test-vaults - Create test vaults for integration testing"
	@echo "  clean-test-vaults  - Clean test vault data"
	@echo "  security-audit     - Run security audit"
	@echo "  coverage-summary   - Show test coverage summary"
	@echo "  check-versions     - Check version consistency with CI environment"
	@echo "  ci                 - CI pipeline (deps, check, security-audit)"
	@echo "  dev                - Development workflow (fmt, test, build)"
	@echo "  release            - Release workflow (clean, fmt, test-coverage, build)"
	@echo "  help               - Show this help message"

.PHONY: build build-unix clean deps test test-race test-coverage test-coverage-view test-unit test-integration test-pkg test-pkg-coverage bench lint fmt vet check install create-test-vaults clean-test-vaults security-audit coverage-summary check-versions ci dev release help
