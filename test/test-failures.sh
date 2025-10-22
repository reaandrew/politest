#!/bin/bash
# Test script for scenarios designed to fail
# Verifies that failure output contains expected elements

set -uo pipefail   # <- no `-e`
IFS=$'\n\t'

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

cd "$(dirname "$0")" || { echo "cd failed"; exit 97; }

POLITEST="../politest"
FAIL_COUNT=0
PASS_COUNT=0

# Build politest if it doesn't exist
if [ ! -f "$POLITEST" ]; then
  echo -e "${YELLOW}Building politest...${NC}"
  if ! ( cd .. && go build -o politest . ); then
    echo -e "${RED}Build failed${NC}"
    exit 98
  fi
fi

echo -e "${GREEN}Running politest failure output tests...${NC}"
echo ""

# Assert helper (regex)
assert_contains() {
  local output="$1" expected_regex="$2" description="$3"
  if echo "$output" | grep -E -q "$expected_regex"; then
    echo -e "  ${GREEN}✓${NC} $description"; return 0
  else
    echo -e "  ${RED}✗${NC} $description"
    echo -e "    ${RED}Expected to match regex:${NC} $expected_regex"; return 1
  fi
}

# Assert helper (literal)
assert_contains_lit() {
  local output="$1" expected_literal="$2" description="$3"
  if echo "$output" | grep -F -q "$expected_literal"; then
    echo -e "  ${GREEN}✓${NC} $description"; return 0
  else
    echo -e "  ${RED}✗${NC} $description"
    echo -e "    ${RED}Expected to find (literal):${NC} $expected_literal"; return 1
  fi
}

run_failure_test() {
  local scenario_file="$1"
  local test_name; test_name=$(basename "$scenario_file" .yml)

  echo -e "${YELLOW}Testing: $test_name${NC}"

  # Determine which flags to use based on scenario name
  local flags="--scenario scenarios/$scenario_file"
  if [[ "$test_name" == fail-strict-policy-* ]]; then
    flags="$flags --strict-policy"
  fi

  # Run politest and capture output + exit code (never abort)
  local output exit_code
  output=$("$POLITEST" $flags 2>&1)
  exit_code=$?

  local test_passed=true

  # Scenario-specific assertions
  case "$test_name" in
    fail-strict-policy-*)
      # For --strict-policy failures, expect exit code 1 (validation error)
      if [ "$exit_code" -eq 1 ]; then
        echo -e "  ${GREEN}✓${NC} Exit code 1 (validation failed as expected)"
      else
        echo -e "  ${RED}✗${NC} Exit code: expected 1, got $exit_code"
        test_passed=false
      fi

      # Check that error message mentions non-IAM fields
      if echo "$output" | grep -q "non-IAM"; then
        echo -e "  ${GREEN}✓${NC} Error message mentions 'non-IAM'"
      else
        echo -e "  ${RED}✗${NC} Error message should mention 'non-IAM'"
        test_passed=false
      fi
      ;;

    fail-01-scp-deny | fail-02-identity-deny-ip | fail-03-identity-deny-mfa)
      # For matched statement failures, expect exit code 2 (assertion failure)
      if [ "$exit_code" -eq 2 ]; then
        echo -e "  ${GREEN}✓${NC} Exit code 2 (test failed as expected)"
      else
        echo -e "  ${RED}✗${NC} Exit code: expected 2, got $exit_code"
        test_passed=false
      fi

      # Common assertions (all matched statement scenarios)
      assert_contains_lit "$output" "Expected: allowed" "Shows 'Expected: allowed'" || test_passed=false
      assert_contains_lit "$output" "Got:      explicitDeny" "Shows 'Got: explicitDeny'" || test_passed=false
      assert_contains_lit "$output" "Matched statements:" "Shows 'Matched statements:' section" || test_passed=false
      ;;
  esac

  # Scenario-specific detailed assertions
  case "$test_name" in
    fail-01-scp-deny)
      # Test 1: s3:GetObject on test-bucket/data.txt
      assert_contains_lit "$output" "Action:   s3:GetObject" "Shows 'Action: s3:GetObject'" || test_passed=false
      assert_contains_lit "$output" "Resource: arn:aws:s3:::test-bucket/data.txt" "Shows resource test-bucket/data.txt" || test_passed=false

      # Test 2: Context keys from second test
      assert_contains_lit "$output" "aws:SourceIp = 10.0.1.50" "Shows context key aws:SourceIp" || test_passed=false
      assert_contains_lit "$output" "aws:MultiFactorAuthPresent = true" "Shows context key aws:MultiFactorAuthPresent" || test_passed=false

      # Test 3: Multiple resources
      assert_contains_lit "$output" "Resources:" "Shows 'Resources:' header for multiple resources" || test_passed=false
      assert_contains_lit "$output" "arn:aws:s3:::bucket1" "Shows first resource in list" || test_passed=false
      assert_contains_lit "$output" "arn:aws:s3:::bucket2" "Shows second resource in list" || test_passed=false
      assert_contains_lit "$output" "arn:aws:s3:::bucket3" "Shows third resource in list" || test_passed=false

      # SCP statement
      assert_contains_lit "$output" "Sid: DenyS3" "Shows original Sid 'DenyS3'" || test_passed=false
      assert_contains "$output" "Source: .*/test/scp/deny-s3\.json:4-9" "Shows SCP source file with line numbers" || test_passed=false
      assert_contains_lit "$output" '4:     {' "Shows line 4 content" || test_passed=false
      assert_contains_lit "$output" '5:       "Sid": "DenyS3"' "Shows line 5 content" || test_passed=false
      assert_contains_lit "$output" '9:     }' "Shows line 9 content" || test_passed=false

      # Should NOT show tracking Sid
      if echo "$output" | grep -F -q "scp:deny-s3.json#stmt:"; then
        echo -e "  ${RED}✗${NC} Should NOT show tracking Sid (scp:deny-s3.json#stmt:X)"
        test_passed=false
      else
        echo -e "  ${GREEN}✓${NC} Does not show tracking Sid"
      fi
      ;;

    fail-02-identity-deny-ip)
      # Action and resource
      assert_contains_lit "$output" "Action:   s3:DeleteObject" "Shows 'Action: s3:DeleteObject'" || test_passed=false
      assert_contains_lit "$output" "Resource: arn:aws:s3:::secure-bucket/sensitive.txt" "Shows resource secure-bucket/sensitive.txt" || test_passed=false

      # Context keys
      assert_contains_lit "$output" "aws:SourceIp = 192.168.1.100" "Shows context key aws:SourceIp = 192.168.1.100" || test_passed=false
      assert_contains_lit "$output" "aws:MultiFactorAuthPresent = true" "Shows context key aws:MultiFactorAuthPresent = true" || test_passed=false

      # Identity policy statement
      assert_contains_lit "$output" "Sid: DenyS3DeleteFromUntrustedNetwork" "Shows original Sid 'DenyS3DeleteFromUntrustedNetwork'" || test_passed=false
      assert_contains "$output" "Source: .*/test/policies/identity-conditional-denies\.json:33-43" "Shows identity policy source with line numbers" || test_passed=false
      assert_contains_lit "$output" '33:     {' "Shows line 33 content" || test_passed=false
      assert_contains_lit "$output" '34:       "Sid": "DenyS3DeleteFromUntrustedNetwork"' "Shows line 34 content" || test_passed=false
      assert_contains_lit "$output" '43:     },' "Shows line 43 content" || test_passed=false

      # Should NOT show tracking Sid
      if echo "$output" | grep -F -q "identity#stmt:"; then
        echo -e "  ${RED}✗${NC} Should NOT show tracking Sid (identity#stmt:X)"
        test_passed=false
      else
        echo -e "  ${GREEN}✓${NC} Does not show tracking Sid"
      fi
      ;;

    fail-03-identity-deny-mfa)
      # Action and resource
      assert_contains_lit "$output" "Action:   ec2:TerminateInstances" "Shows 'Action: ec2:TerminateInstances'" || test_passed=false
      assert_contains_lit "$output" "Resource: *" "Shows resource *" || test_passed=false

      # Context keys
      assert_contains_lit "$output" "aws:MultiFactorAuthPresent = false" "Shows context key aws:MultiFactorAuthPresent = false" || test_passed=false
      assert_contains_lit "$output" "aws:PrincipalTag/Department = Engineering" "Shows context key aws:PrincipalTag/Department = Engineering" || test_passed=false

      # Identity policy statement
      assert_contains_lit "$output" "Sid: DenyEC2WithoutMFA" "Shows original Sid 'DenyEC2WithoutMFA'" || test_passed=false
      assert_contains "$output" "Source: .*/test/policies/identity-conditional-denies\.json:44-57" "Shows identity policy source with line numbers" || test_passed=false
      assert_contains_lit "$output" '44:     {' "Shows line 44 content" || test_passed=false
      assert_contains_lit "$output" '45:       "Sid": "DenyEC2WithoutMFA"' "Shows line 45 content" || test_passed=false
      assert_contains_lit "$output" '57:     }' "Shows line 57 content" || test_passed=false

      # Should NOT show tracking Sid
      if echo "$output" | grep -F -q "identity#stmt:"; then
        echo -e "  ${RED}✗${NC} Should NOT show tracking Sid (identity#stmt:X)"
        test_passed=false
      else
        echo -e "  ${GREEN}✓${NC} Does not show tracking Sid"
      fi
      ;;

    *)
      echo -e "  ${YELLOW}⚠${NC} Unknown scenario: $test_name (skipping specific assertions)"
      ;;
  esac

  if [ "$test_passed" = true ]; then
    echo -e "  ${GREEN}✓ PASS${NC}"
    ((PASS_COUNT++))
  else
    echo -e "  ${RED}✗ FAIL${NC}"
    ((FAIL_COUNT++))
  fi

  echo ""
}

# Expand scenarios and show what matched (helps diagnose early exits)
mapfile -t SCENARIOS < <(ls -1 scenarios/fail-*.yml 2>/dev/null || true)
if [ "${#SCENARIOS[@]}" -eq 0 ]; then
  echo -e "${YELLOW}No scenarios matched: scenarios/fail-*.yml${NC}"
  exit 99
fi

echo -e "${YELLOW}Found ${#SCENARIOS[@]} scenario(s):${NC}"
for s in "${SCENARIOS[@]}"; do echo "  - $s"; done
echo ""

# Run them
for scenario in "${SCENARIOS[@]}"; do
  run_failure_test "$(basename "$scenario")"
done

# Summary
echo "========================================"
echo -e "Test Results: ${GREEN}$PASS_COUNT passed${NC}, ${RED}$FAIL_COUNT failed${NC}"
echo "========================================"
exit $(( FAIL_COUNT > 0 ))
