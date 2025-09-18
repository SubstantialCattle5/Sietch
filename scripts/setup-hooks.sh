#!/bin/bash

# Sietch Vault - Git Hooks Setup Script
# This script sets up Husky Git hooks for code quality checks

set -e

echo "🔧 Setting up Sietch Vault development environment..."

# Check if we're in a Git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "❌ This script must be run from within a Git repository"
    exit 1
fi

# Check if Node.js is installed
if ! command -v node >/dev/null 2>&1; then
    echo "❌ Node.js is required but not installed"
    echo "💡 Please install Node.js from https://nodejs.org/"
    exit 1
fi

# Check if npm is installed
if ! command -v npm >/dev/null 2>&1; then
    echo "❌ npm is required but not installed"
    echo "💡 npm usually comes with Node.js"
    exit 1
fi

# Check if Go is installed and version
EXPECTED_GO_VERSION="1.24"
if ! command -v go >/dev/null 2>&1; then
    echo "❌ Go is required but not installed"
    echo "💡 Please install Go $EXPECTED_GO_VERSION from https://golang.org/dl/"
    exit 1
fi

CURRENT_GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
if [ "$CURRENT_GO_VERSION" != "$EXPECTED_GO_VERSION" ]; then
    echo "⚠️  Go version mismatch: found $CURRENT_GO_VERSION, CI uses $EXPECTED_GO_VERSION"
    echo "💡 Consider updating Go to match CI environment for consistency"
    echo "   Current: go$CURRENT_GO_VERSION"
    echo "   CI uses: go$EXPECTED_GO_VERSION"
else
    echo "✅ Go version $CURRENT_GO_VERSION matches CI environment"
fi

echo "✅ Prerequisites check passed"

# Install npm dependencies
echo "📦 Installing npm dependencies..."
npm install

# Install Go dependencies
echo "📦 Installing Go dependencies..."
make deps

# Install development tools
echo "🔧 Installing development tools..."

# Install golangci-lint if not present or wrong version
EXPECTED_GOLANGCI_VERSION="v1.60.3"
if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "📥 Installing golangci-lint $EXPECTED_GOLANGCI_VERSION..."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $EXPECTED_GOLANGCI_VERSION
    echo "✅ golangci-lint installed"
else
    CURRENT_VERSION=$(golangci-lint version | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1)
    if [ "$CURRENT_VERSION" != "$EXPECTED_GOLANGCI_VERSION" ]; then
        echo "⚠️  golangci-lint version mismatch: found $CURRENT_VERSION, expected $EXPECTED_GOLANGCI_VERSION"
        echo "📥 Updating to $EXPECTED_GOLANGCI_VERSION..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $EXPECTED_GOLANGCI_VERSION
        echo "✅ golangci-lint updated"
    else
        echo "✅ golangci-lint $CURRENT_VERSION already installed"
    fi
fi

# Install gosec if not present
if ! command -v gosec >/dev/null 2>&1; then
    echo "📥 Installing gosec..."
    go install github.com/securego/gosec/v2/cmd/gosec@latest
    echo "✅ gosec installed"
else
    echo "✅ gosec already installed"
fi

# Setup Husky hooks
echo "🪝 Setting up Git hooks..."
npx husky install

# Verify hooks are working
echo "🧪 Testing hooks setup..."
if [ -f .husky/pre-commit ] && [ -x .husky/pre-commit ]; then
    echo "✅ pre-commit hook is executable"
else
    echo "⚠️  pre-commit hook setup issue"
fi

if [ -f .husky/pre-push ] && [ -x .husky/pre-push ]; then
    echo "✅ pre-push hook is executable"
else
    echo "⚠️  pre-push hook setup issue"
fi

if [ -f .husky/commit-msg ] && [ -x .husky/commit-msg ]; then
    echo "✅ commit-msg hook is executable"
else
    echo "⚠️  commit-msg hook setup issue"
fi

# Run initial checks
echo "🔍 Running initial code quality checks..."
echo "📊 Verifying tool versions..."
echo "  Go: $(go version)"
echo "  golangci-lint: $(golangci-lint version | head -1)"
echo "  gosec: $(gosec -version 2>/dev/null || echo 'installed')"

echo "📝 Checking formatting..."
make fmt

echo "🧹 Running linter..."
if make lint; then
    echo "✅ Linting passed"
else
    echo "⚠️  Linting issues found - please review and fix"
fi

echo "🧪 Running tests..."
if make test-unit; then
    echo "✅ Unit tests passed"
else
    echo "⚠️  Some tests failed - please review and fix"
fi

echo ""
echo "🎉 Development environment setup complete!"
echo ""
echo "📋 What's been set up:"
echo "  ✅ Husky Git hooks installed"
echo "  ✅ Pre-commit: formatting, linting, unit tests"
echo "  ✅ Pre-push: full tests, build verification, security audit"
echo "  ✅ Commit-msg: conventional commits enforcement"
echo "  ✅ Development tools installed"
echo ""
echo "💡 Tips:"
echo "  • Use conventional commit format: 'feat: add new feature'"
echo "  • Run 'make help' to see available commands"
echo "  • Set HUSKY=0 to skip hooks temporarily"
echo "  • Run 'npm run prepare' if hooks stop working"
echo ""
echo "🚀 You're ready to contribute to Sietch Vault!"
