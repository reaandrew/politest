---
name: Auto-fix Post-mortem
description: Analyze auto-fix workflow effectiveness and identify improvement opportunities. Use for weekly/monthly reviews or after changes to auto-fix workflow.
---

# Auto-fix Post-mortem Agent

## Purpose
Analyze auto-fix workflow effectiveness and identify improvement opportunities. Provides metrics and insights for continuous improvement of the automated CI recovery system.

## Trigger
`@claude autofix-report`

## What It Does

1. **Fetch Recent Auto-fix Runs**
   - Uses `gh run list --workflow=auto-fix-ci.yml --limit=20`
   - Retrieves last N auto-fix workflow executions
   - Collects timestamps, durations, outcomes
   ```
   Auto-fix Runs (Last 20)
   ======================

   Total Runs: 20
   Date Range: 2025-09-15 to 2025-10-03
   ```

2. **Calculate Success Metrics**
   - Success rate (fixed vs failed to fix)
   - Average time to fix
   - Number of retry attempts
   - Types of failures fixed vs not fixed
   ```
   Success Metrics
   ==============

   Success Rate: 75% (15/20 runs)
   - Successfully fixed: 15
   - Failed to fix: 3
   - Skipped (already running): 2

   Average Time to Fix: 3m 24s
   - Fastest: 1m 12s (simple syntax error)
   - Slowest: 8m 45s (multiple test failures)

   Attempts Distribution:
   - First attempt success: 12 (80% of successes)
   - Second attempt success: 3 (20% of successes)
   ```

3. **Identify Common Failure Patterns**
   - Categorize errors auto-fix handles well vs poorly
   - Find recurring issues
   - Identify types needing human intervention
   ```
   Failure Pattern Analysis
   =======================

   Successfully Auto-fixed:
   1. NameError (undefined variables): 8 occurrences ✅
   2. Syntax errors: 4 occurrences ✅
   3. Import errors: 2 occurrences ✅
   4. Simple test failures: 1 occurrence ✅

   Failed to Auto-fix:
   1. Complex logic errors: 2 occurrences ❌
      - Require understanding business logic
      - Auto-fix made incorrect assumptions
   2. Coverage threshold failures: 1 occurrence ❌
      - Needs new test cases, not just fixes

   Patterns Requiring Human Review:
   - Breaking API changes
   - Performance regressions
   - Complex refactoring needs
   ```

4. **Analyze CI Trigger Success**
   - Check if auto-fix successfully triggered follow-up CI
   - Measure time from fix to CI completion
   - Identify trigger failures
   ```
   CI Trigger Analysis
   ==================

   Runs that Triggered CI: 12/15 (80%)
   Runs that Failed to Trigger: 3/15 (20%)

   Failed Trigger Causes:
   1. Permission denied (2 occurrences)
      - Fixed: Added actions:write permission
   2. Branch ref mismatch (1 occurrence)
      - Fixed: Use correct branch variable

   Average CI Re-run Duration: 2m 15s
   - CI triggered successfully
   - Tests passed on re-run
   ```

5. **Measure Turn-around Time**
   - Time from CI failure to auto-fix completion
   - Time from auto-fix to passing CI
   - Total recovery time
   ```
   Turn-around Time Metrics
   =======================

   Phase 1: CI Failure Detection
   - Average: 30s (workflow_run trigger delay)

   Phase 2: Auto-fix Execution
   - Average: 3m 24s (analysis + fix + commit)

   Phase 3: CI Re-trigger & Completion
   - Average: 2m 15s (new CI run)

   Total Recovery Time: 6m 9s (average)

   Best Case: 2m 30s (simple error, fast CI)
   Worst Case: 15m 20s (complex error, slow CI)
   ```

6. **Generate Improvement Recommendations**
   ```
   Recommendations for Auto-fix Improvement
   =======================================

   HIGH Priority:
   1. Improve CI trigger reliability (currently 80%)
      - Action: Already fixed with actions:write permission
      - Verify in next 10 runs

   2. Add coverage failure handling (0% success rate)
      - Action: Enhance prompt to generate test cases
      - Example: "If coverage < 80%, write tests for uncovered lines"

   MEDIUM Priority:
   3. Reduce time to fix (currently 3m 24s)
      - Action: Cache dependencies in workflow
      - Expected improvement: 30-40% faster

   4. Better handling of logic errors (33% success rate)
      - Action: Add confidence scoring before fixing
      - If confidence < 70%, create issue instead of auto-fixing

   LOW Priority:
   5. Add auto-fix for flake8 warnings
      - Currently requires manual intervention
      - Could auto-apply black formatting
   ```

7. **Compare Time Periods**
   - Week-over-week improvement
   - Before/after configuration changes
   - Identify trends
   ```
   Trend Analysis
   =============

   Last 7 Days vs Previous 7 Days:

   Success Rate: 85% → 90% (+5%) ✅
   Average Fix Time: 4m 12s → 3m 24s (-19%) ✅
   CI Trigger Success: 65% → 95% (+30%) ✅

   Improvement Drivers:
   - Added actions:write permission (CI trigger fix)
   - Improved prompt clarity (faster analysis)
   - Better error categorization (higher success rate)
   ```

## Important Rules

- **DO collect** quantitative metrics
- **DO identify** patterns, not just individual failures
- **DO provide** actionable recommendations
- **DO compare** time periods to show trends
- **NEVER make changes** without user approval
- **DO explain** what each metric means

## Usage Example

```bash
# Generate monthly auto-fix report
@claude autofix-report

# Agent analyzes recent runs
# Calculates success metrics
# Provides improvement recommendations

# User reviews report
# Decides which improvements to implement

# Agent can then make improvements:
@claude implement recommendation #2
```

## When to Use

- Weekly/monthly to track auto-fix effectiveness
- After making changes to auto-fix workflow
- To justify auto-fix system value
- To identify areas needing improvement
- Before expanding auto-fix capabilities

## Key Metrics Tracked

| Metric | Target | Current |
|--------|--------|---------|
| Success Rate | > 80% | 75% |
| Avg Fix Time | < 5m | 3m 24s |
| CI Trigger Success | > 90% | 80% |
| First Attempt Success | > 75% | 80% |
| Total Recovery Time | < 10m | 6m 9s |

## Report Formats

**Executive Summary** (default):
- High-level metrics
- Key trends
- Top 3 recommendations

**Detailed Analysis**:
- All metrics with breakdowns
- Every auto-fix run analyzed
- Failure case studies

**Comparative Report**:
- Before/after specific change
- Week-over-week trends
- Month-over-month progress

## Tools Required

- `Bash` - for gh CLI commands
- `WebFetch` - for GitHub API access
- `Read` - for reading workflow logs
- `Grep` - for parsing log patterns
