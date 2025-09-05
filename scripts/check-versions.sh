#!/bin/bash

# Sietch Vault - Version Consistency Checker
# This script verifies that local development tools match CI environment

set -e

echo "🔍 Checking version consistency between local and CI environments..."

# Expected versions (must match CI configuration)
EXPECTED_GO_VERSION="1.23"
EXPECTED_GOLANGCI_VERSION="v1.60.3"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    local status=$1
    local message=$2
    case $status in
        "OK")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "WARNING")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "ERROR")
            echo -e "${RED}❌ $message${NC}"
            ;;
    esac
}

# Check Go version
echo "📊 Checking Go version..."
if command -v go >/dev/null 2>&1; then
    CURRENT_GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    if [ "$CURRENT_GO_VERSION" = "$EXPECTED_GO_VERSION" ]; then
        print_status "OK" "Go version matches CI: $CURRENT_GO_VERSION"
    else
        print_status "WARNING" "Go version mismatch - Local: $CURRENT_GO_VERSION, CI: $EXPECTED_GO_VERSION"
        echo "   Consider updating to Go $EXPECTED_GO_VERSION for consistency"
    fi
else
    print_status "ERROR" "Go is not installed"
    exit 1
fi

# Check golangci-lint version
echo "🧹 Checking golangci-lint version..."
if command -v golangci-lint >/dev/null 2>&1; then
    CURRENT_GOLANGCI_VERSION=$(golangci-lint version | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1)
    if [ "$CURRENT_GOLANGCI_VERSION" = "$EXPECTED_GOLANGCI_VERSION" ]; then
        print_status "OK" "golangci-lint version matches CI: $CURRENT_GOLANGCI_VERSION"
    else
        print_status "WARNING" "golangci-lint version mismatch - Local: $CURRENT_GOLANGCI_VERSION, CI: $EXPECTED_GOLANGCI_VERSION"
        echo "   Run './scripts/setup-hooks.sh' to update to $EXPECTED_GOLANGCI_VERSION"
    fi
else
    print_status "ERROR" "golangci-lint is not installed"
    echo "   Run './scripts/setup-hooks.sh' to install it"
    exit 1
fi

# Check gosec
echo "🔒 Checking gosec..."
if command -v gosec >/dev/null 2>&1; then
    print_status "OK" "gosec is installed"
else
    print_status "WARNING" "gosec is not installed"
    echo "   Run './scripts/setup-hooks.sh' to install it"
fi

# Check configuration files
echo "⚙️  Checking configuration files..."
if [ -f ".golangci.yml" ]; then
    # Check if gci is configured
    if grep -q "gci:" .golangci.yml; then
        print_status "OK" ".golangci.yml has gci configuration"
    else
        print_status "ERROR" ".golangci.yml missing gci configuration"
        exit 1
    fi
else
    print_status "ERROR" ".golangci.yml not found"
    exit 1
fi

if [ -f ".gosec.json" ]; then
    print_status "OK" ".gosec.json configuration found"
else
    print_status "WARNING" ".gosec.json configuration not found"
fi

echo ""
echo "📋 Summary:"
echo "   This check helps prevent the 'passes locally, fails in CI' issue"
echo "   by ensuring your local tools match the CI environment."
echo ""
echo "💡 To fix version mismatches:"
echo "   • Run './scripts/setup-hooks.sh' to sync tool versions"
echo "   • Consider updating Go to match CI for best consistency"
echo ""
echo "🎉 Version check complete!"
