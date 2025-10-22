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

# Run strict-policy flag tests
echo -e "${YELLOW}Running --strict-policy flag tests...${NC}"
if bash "$SCRIPT_DIR/test-strict-policy.sh"; then
    echo -e "${GREEN}✓ Strict-policy tests passed${NC}"
    echo ""
else
    echo -e "${RED}✗ Strict-policy tests failed${NC}"
    exit 1
fi

echo -e "${GREEN}=== All Tests Passed ===${NC}"
exit 0
