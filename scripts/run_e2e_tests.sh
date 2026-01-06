#!/usr/bin/env bash
# Copyright 2025 Takhin Data, Inc.
#
# E2E Test Runner Script
# Runs the complete end-to-end test suite

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/../backend"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "  Takhin E2E Test Suite Runner"
echo "========================================="
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

# Change to backend directory
cd "$BACKEND_DIR"

# Parse command line arguments
TEST_SUITE="${1:-all}"
VERBOSE="${2:-false}"

# Build test flags
TEST_FLAGS="-v -race -tags=e2e -timeout=30m"
if [ "$VERBOSE" = "true" ]; then
    TEST_FLAGS="$TEST_FLAGS -v"
fi

# Function to run a specific test suite
run_suite() {
    local suite_name=$1
    local suite_path=$2
    
    echo -e "${YELLOW}Running $suite_name tests...${NC}"
    
    if go test $TEST_FLAGS "$suite_path"; then
        echo -e "${GREEN}✓ $suite_name tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $suite_name tests failed${NC}"
        return 1
    fi
}

# Track failures
FAILED_SUITES=()

# Run test suites based on selection
case "$TEST_SUITE" in
    "all")
        echo "Running all E2E test suites..."
        echo
        
        run_suite "Producer/Consumer" "./tests/e2e/producer_consumer" || FAILED_SUITES+=("producer_consumer")
        echo
        
        run_suite "Consumer Group" "./tests/e2e/consumer_group" || FAILED_SUITES+=("consumer_group")
        echo
        
        run_suite "Admin API" "./tests/e2e/admin_api" || FAILED_SUITES+=("admin_api")
        echo
        
        run_suite "Fault Injection" "./tests/e2e/fault_injection" || FAILED_SUITES+=("fault_injection")
        echo
        
        run_suite "Performance" "./tests/e2e/performance" || FAILED_SUITES+=("performance")
        echo
        ;;
        
    "producer_consumer")
        run_suite "Producer/Consumer" "./tests/e2e/producer_consumer" || FAILED_SUITES+=("producer_consumer")
        ;;
        
    "consumer_group")
        run_suite "Consumer Group" "./tests/e2e/consumer_group" || FAILED_SUITES+=("consumer_group")
        ;;
        
    "admin_api")
        run_suite "Admin API" "./tests/e2e/admin_api" || FAILED_SUITES+=("admin_api")
        ;;
        
    "fault_injection")
        run_suite "Fault Injection" "./tests/e2e/fault_injection" || FAILED_SUITES+=("fault_injection")
        ;;
        
    "performance")
        run_suite "Performance" "./tests/e2e/performance" || FAILED_SUITES+=("performance")
        ;;
        
    *)
        echo -e "${RED}Error: Unknown test suite: $TEST_SUITE${NC}"
        echo "Available suites: all, producer_consumer, consumer_group, admin_api, fault_injection, performance"
        exit 1
        ;;
esac

# Summary
echo "========================================="
echo "  Test Summary"
echo "========================================="

if [ ${#FAILED_SUITES[@]} -eq 0 ]; then
    echo -e "${GREEN}All test suites passed!${NC}"
    exit 0
else
    echo -e "${RED}Failed test suites:${NC}"
    for suite in "${FAILED_SUITES[@]}"; do
        echo -e "  ${RED}✗ $suite${NC}"
    done
    exit 1
fi
