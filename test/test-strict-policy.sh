#!/bin/bash
# Test --strict-policy flag behavior
# This test expects the command to FAIL when non-IAM fields are present

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$PROJECT_ROOT/politest"

# Build the binary if it doesn't exist
if [ ! -f "$BINARY" ]; then
    echo -e "${YELLOW}Building politest binary...${NC}"
    cd "$PROJECT_ROOT"
    go build -o politest
    cd "$SCRIPT_DIR"
fi

echo -e "${GREEN}Testing --strict-policy flag behavior...${NC}"
echo ""

PASSED=0
FAILED=0

# Test 1: --strict-policy should FAIL when policy has non-IAM fields
echo -e "${YELLOW}Test 1: --strict-policy with non-IAM fields (should fail)${NC}"
if "$BINARY" --scenario "$SCRIPT_DIR/scenarios/19-strip-non-iam-fields-default.yml" --strict-policy > /tmp/strict-test.log 2>&1; then
    echo -e "  ${RED}✗ FAIL: Command should have failed but succeeded${NC}"
    FAILED=$((FAILED + 1))
else
    # Check that error message mentions non-IAM fields
    if grep -q "non-IAM" /tmp/strict-test.log; then
        echo -e "  ${GREEN}✓ PASS: Command failed as expected with correct error message${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "  ${RED}✗ FAIL: Command failed but with unexpected error message${NC}"
        echo "  Output:"
        sed 's/^/    /' /tmp/strict-test.log
        FAILED=$((FAILED + 1))
    fi
fi
echo ""

# Test 2: --strict-policy should SUCCEED when policy has only IAM fields
echo -e "${YELLOW}Test 2: --strict-policy with valid IAM fields only (should succeed)${NC}"
if "$BINARY" --scenario "$SCRIPT_DIR/scenarios/01-policy-allows-no-boundaries.yml" --strict-policy > /tmp/strict-test2.log 2>&1; then
    echo -e "  ${GREEN}✓ PASS: Command succeeded as expected${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "  ${RED}✗ FAIL: Command should have succeeded but failed${NC}"
    echo "  Output:"
    sed 's/^/    /' /tmp/strict-test2.log
    FAILED=$((FAILED + 1))
fi
echo ""

# Summary
echo "======================================"
echo -e "Test Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"
echo "======================================"

if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0
