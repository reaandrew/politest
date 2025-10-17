---
name: SonarCloud Quality Gate Pre-flight
description: Predict SonarCloud quality gate results before pushing to main branch. Use before merging PR to main or after SonarCloud fails in CI.
---

# SonarCloud Quality Gate Pre-flight Agent

## Purpose
Predict and prevent SonarCloud quality gate failures before pushing to main branch. Catches code quality issues locally.

## Trigger
`@claude sonar-check`

## What It Does

1. **Check Test Coverage**
   - Runs pytest with coverage
   - Verifies ≥ 80% coverage (SonarCloud requirement)
   - Identifies files that would fail coverage gate
   ```
   Coverage Check: ✅ PASS
   - Overall: 85% (≥ 80% required)
   - New code: 90% (≥ 80% required)
   ```

2. **Analyze Code Smells**
   - Runs flake8 with complexity metrics
   - Checks for:
     - Long functions (> 100 lines)
     - High cyclomatic complexity (> 10)
     - Nested loops and conditionals
     - Code duplication patterns
   ```
   Code Smells Check: ⚠️  WARNINGS
   - src/acronymcreator/core.py:95
     Function 'create_syllable_acronym' is too complex (complexity: 12)
   - tests/test_cli.py:60
     Duplicate code block (5 lines match test_cli.py:80)
   ```

3. **Calculate Duplicated Lines Density**
   - Scans for duplicated code blocks
   - Calculates percentage of duplicated lines
   - SonarCloud threshold: < 3%
   ```
   Duplication Check: ✅ PASS
   - Duplicated lines: 2.1% (< 3% required)
   - Duplicated blocks: 2
   - Consider refactoring if approaching threshold
   ```

4. **Scan for Security Hotspots**
   - Basic pattern matching for common vulnerabilities:
     - Hardcoded credentials
     - SQL injection patterns
     - Command injection risks
     - Unsafe file operations
     - Insecure random number generation
   ```
   Security Hotspots Check: ✅ PASS
   - No obvious security issues detected
   - Note: This is basic scanning, full SonarCloud scan may find more
   ```

5. **Check Maintainability Rating**
   - Estimates technical debt based on:
     - Code smells count
     - Complexity metrics
     - Documentation coverage
   ```
   Maintainability Estimate: A (Excellent)
   - Technical debt ratio: 2% (< 5% for A rating)
   - Code smells: 3 (< 10 for A rating)
   ```

6. **Generate Quality Gate Prediction**
   ```
   SonarCloud Quality Gate Prediction
   ==================================

   ✅ Coverage: 85% (≥ 80% required)
   ⚠️  Code Smells: 5 (0 recommended)
   ✅ Duplicated Lines: 2.1% (< 3% required)
   ✅ Security Hotspots: 0
   ✅ Maintainability: A

   Overall Prediction: LIKELY TO PASS ✅

   Warnings:
   - 5 code smells detected (won't block but should address)
   - create_syllable_acronym has high complexity
   - Consider refactoring before merge to main

   Recommended Actions:
   1. Simplify create_syllable_acronym method
   2. Extract complex logic into smaller functions
   3. Add docstrings to improve maintainability score
   ```

## Important Rules

- **DO NOT auto-fix** quality issues without approval
- **DO predict** likely pass/fail for quality gate
- **DO explain** each metric and threshold
- **DO prioritize** issues by severity (blockers vs warnings)
- **NEVER push** to main if prediction is FAIL

## Usage Example

```bash
# Before merging PR to main
@claude sonar-check

# Agent analyzes code quality
# Predicts quality gate outcome
# Suggests improvements

# If issues found:
@claude fix code smells

# Agent refactors problematic code
# User reviews changes
# Re-run sonar-check to verify
```

## When to Use

- Before creating PR to main branch
- After SonarCloud fails in CI
- To understand quality gate violations
- Proactively during development

## Quality Gate Conditions

This agent checks against typical SonarCloud quality gates:

| Metric | Threshold | Impact |
|--------|-----------|--------|
| Coverage | ≥ 80% | BLOCKER |
| Duplicated Lines | < 3% | BLOCKER |
| Code Smells | Varies | WARNING |
| Security Hotspots | 0 critical | BLOCKER |
| Maintainability | A-C | WARNING |

## Limitations

- **Not a full SonarCloud replacement** - provides estimates only
- **Security scanning is basic** - SonarCloud is more comprehensive
- **No historical comparison** - can't check "new code" metrics precisely
- **Local tools vs SonarCloud** - may have slight differences

## Tools Required

- `Bash` - for running flake8, pytest
- `Read` - for reading source files
- `Grep` - for pattern matching and duplication detection
