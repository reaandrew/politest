#!/bin/bash
# Integration test runner for politest
# Runs all scenario tests and reports results

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

# Check if AWS credentials are available
if ! aws sts get-caller-identity &>/dev/null; then
    echo -e "${RED}ERROR: AWS credentials not configured or insufficient permissions${NC}"
    echo "Please ensure:"
    echo "  1. AWS credentials are configured (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)"
    echo "  2. The IAM principal has 'iam:SimulateCustomPolicy' permission"
    exit 1
fi

echo -e "${GREEN}Running politest integration tests...${NC}"
echo ""

PASSED=0
FAILED=0
SCENARIOS=()

# Find all scenario files
while IFS= read -r -d '' scenario; do
    SCENARIOS+=("$scenario")
done < <(find "$SCRIPT_DIR/scenarios" -name "*.yml" -print0 | sort -z)

# Run each scenario
for scenario in "${SCENARIOS[@]}"; do
    scenario_name=$(basename "$scenario")
    echo -e "${YELLOW}Testing: $scenario_name${NC}"

    if "$BINARY" --scenario "$scenario" > /tmp/politest-output.log 2>&1; then
        echo -e "  ${GREEN}✓ PASS${NC}"
        ((PASSED++))
    else
        echo -e "  ${RED}✗ FAIL${NC}"
        echo "  Output:"
        sed 's/^/    /' /tmp/politest-output.log
        ((FAILED++))
    fi
    echo ""
done

# Summary
echo "======================================"
echo -e "Test Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"
echo "======================================"

if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0
