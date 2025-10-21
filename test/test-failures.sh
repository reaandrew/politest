#!/bin/bash
# Test script for scenarios designed to fail
# Verifies that failure output contains expected elements

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

cd "$(dirname "$0")"

POLITEST="../politest"
FAIL_COUNT=0
PASS_COUNT=0

# Build politest if it doesn't exist
if [ ! -f "$POLITEST" ]; then
    echo -e "${YELLOW}Building politest...${NC}"
    cd ..
    go build -o politest .
    cd test
fi

echo -e "${GREEN}Running politest failure output tests...${NC}"
echo ""

# Helper function to check if output contains expected string
assert_contains() {
    local output="$1"
    local expected="$2"
    local description="$3"

    if echo "$output" | grep -q "$expected"; then
        echo -e "  ${GREEN}✓${NC} $description"
        return 0
    else
        echo -e "  ${RED}✗${NC} $description"
        echo -e "    ${RED}Expected to find: $expected${NC}"
        return 1
    fi
}

# Helper function to run a failure test
run_failure_test() {
    local scenario_file="$1"
    local test_name=$(basename "$scenario_file" .yml)

    echo -e "${YELLOW}Testing: $test_name${NC}"

    # Run the test and capture output and exit code
    set +e
    output=$($POLITEST --scenario "scenarios/$scenario_file" 2>&1)
    exit_code=$?
    set -e

    local test_passed=true

    # Assert exit code is 2 (test failure) - tests run but assertions failed
    if [ $exit_code -eq 2 ]; then
        echo -e "  ${GREEN}✓${NC} Exit code 2 (test failed as expected)"
    else
        echo -e "  ${RED}✗${NC} Exit code: expected 2, got $exit_code"
        test_passed=false
    fi

    # Test 1: Basic failure output format
    assert_contains "$output" "Expected: allowed" "Shows 'Expected: allowed'" || test_passed=false
    assert_contains "$output" "Action:   s3:GetObject" "Shows 'Action: s3:GetObject'" || test_passed=false
    assert_contains "$output" "Resource: arn:aws:s3:::test-bucket/data.txt" "Shows single resource" || test_passed=false
    assert_contains "$output" "Got:      explicitDeny" "Shows 'Got: explicitDeny'" || test_passed=false

    # Test 2: Original Sid displayed (not tracking Sid)
    assert_contains "$output" "Sid: DenyS3" "Shows original Sid 'DenyS3'" || test_passed=false
    if echo "$output" | grep -q "scp:deny-s3.json#stmt:"; then
        echo -e "  ${RED}✗${NC} Should NOT show tracking Sid (scp:deny-s3.json#stmt:X)"
        test_passed=false
    else
        echo -e "  ${GREEN}✓${NC} Does not show tracking Sid"
    fi

    # Test 3: Line numbers in source file
    assert_contains "$output" "Source: .*/test/scp/deny-s3.json:4-9" "Shows line numbers (4-9)" || test_passed=false

    # Test 4: Statement lines displayed
    assert_contains "$output" '4:     {' "Shows line 4 content" || test_passed=false
    assert_contains "$output" '5:       "Sid": "DenyS3"' "Shows line 5 content" || test_passed=false
    assert_contains "$output" '9:     }' "Shows line 9 content" || test_passed=false

    # Test 5: Context keys (from test 2)
    assert_contains "$output" "aws:SourceIp = 10.0.1.50" "Shows context key aws:SourceIp" || test_passed=false
    assert_contains "$output" "aws:MultiFactorAuthPresent = true" "Shows context key aws:MultiFactorAuthPresent" || test_passed=false

    # Test 6: Multiple resources (from test 3)
    assert_contains "$output" "Resources:" "Shows 'Resources:' header for multiple resources" || test_passed=false
    assert_contains "$output" "arn:aws:s3:::bucket1" "Shows first resource in list" || test_passed=false
    assert_contains "$output" "arn:aws:s3:::bucket2" "Shows second resource in list" || test_passed=false
    assert_contains "$output" "arn:aws:s3:::bucket3" "Shows third resource in list" || test_passed=false

    if [ "$test_passed" = true ]; then
        echo -e "  ${GREEN}✓ PASS${NC}"
        ((PASS_COUNT++))
    else
        echo -e "  ${RED}✗ FAIL${NC}"
        ((FAIL_COUNT++))
        # Optionally show full output on failure
        # echo -e "\n${YELLOW}Full output:${NC}"
        # echo "$output"
    fi

    echo ""
}

# Run all fail-*.yml scenarios
for scenario in scenarios/fail-*.yml; do
    if [ -f "$scenario" ]; then
        run_failure_test "$(basename "$scenario")"
    fi
done

# Summary
echo "========================================"
if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "Test Results: ${GREEN}$PASS_COUNT passed${NC}, ${RED}$FAIL_COUNT failed${NC}"
    echo "========================================"
    exit 0
else
    echo -e "Test Results: ${GREEN}$PASS_COUNT passed${NC}, ${RED}$FAIL_COUNT failed${NC}"
    echo "========================================"
    exit 1
fi
