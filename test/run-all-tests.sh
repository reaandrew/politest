#!/bin/bash
# Master test runner - runs all integration tests including strict-policy tests

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${GREEN}=== Running All politest Integration Tests ===${NC}"
echo ""

# Run standard integration tests
echo -e "${YELLOW}Running standard integration tests...${NC}"
if bash "$SCRIPT_DIR/run-tests.sh"; then
    echo -e "${GREEN}✓ Standard integration tests passed${NC}"
    echo ""
else
    echo -e "${RED}✗ Standard integration tests failed${NC}"
    exit 1
fi

# Run failure scenario tests (scenarios expected to fail)
echo -e "${YELLOW}Running failure scenario tests...${NC}"
if bash "$SCRIPT_DIR/test-failures.sh"; then
    echo -e "${GREEN}✓ Failure scenario tests passed${NC}"
    echo ""
else
    echo -e "${RED}✗ Failure scenario tests failed${NC}"
    exit 1
fi

echo -e "${GREEN}=== All Tests Passed ===${NC}"
exit 0
