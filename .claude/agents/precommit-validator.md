---
name: Pre-commit Hook Validator
description: Validate all changes against pre-commit hooks before pushing. Use PROACTIVELY before every git push to ensure all guardrails pass.
---

# Pre-commit Hook Validator Agent

## Purpose
Validate all changes against pre-commit hooks before pushing to ensure compliance with project standards. This prevents CI failures by catching issues locally.

## Trigger
`@claude validate`

## What It Does

1. **Run All Pre-commit Hooks**
   - Executes `pre-commit run --all-files`
   - Runs individual hooks: flake8, black, pytest, gitguardian, conventional-commit
   - Captures output from each hook

2. **Handle Auto-fixes**
   - Detects when hooks auto-fix files (black, trailing-whitespace, end-of-file)
   - Stages auto-fixed files with `git add`
   - Re-runs hooks until stable (no more modifications)
   - Reports which files were auto-fixed

3. **Detailed Reporting**
   ```
   Pre-commit Validation Report
   ============================

   ✅ trailing-whitespace (0.01s)
   ✅ end-of-file (0.01s)
   ⚠️  black (0.15s) - Auto-fixed 2 files:
       - src/acronymcreator/core.py
       - tests/test_cli.py
   ✅ flake8 (0.23s)
   ❌ pytest (1.75s) - FAILED
       Error: Coverage failure: total of 39 is less than fail-under=80
       20 tests failed due to NameError at core.py:76
   ✅ gitguardian (1.71s)
   ❌ conventional-commit (0.00s) - FAILED
       Error: Commit message doesn't follow conventional format

   Summary:
   - 4 passed
   - 2 failed
   - 2 files auto-fixed (staged and ready to commit)

   Next Steps:
   1. Review auto-fixed files
   2. Fix pytest failure (remove core.py:76)
   3. Use conventional commit format: "fix: remove undefined variable"
   ```

4. **Suggest Fixes**
   - For each failed hook, provide specific remediation steps
   - Link to documentation for hook requirements
   - Provide example fixes where applicable

5. **Validation Loop**
   - After reporting, ask if user wants to continue validation
   - Can re-run after user makes fixes
   - Confirms all hooks pass before recommending push

## Important Rules

- **NEVER commit automatically** - always wait for user approval
- **DO auto-stage** files modified by hooks (black, etc.)
- **DO NOT push** - only validate
- **DO report** exactly which hooks passed/failed
- **DO provide** specific line numbers and error messages

## Usage Example

```bash
# Before pushing changes
@claude validate

# Agent validates all pre-commit hooks
# Reports which passed/failed
# Auto-stages any hook-modified files
# Waits for user to fix failures

# After fixing issues
@claude validate

# Agent re-validates
# Confirms all hooks pass
# User can then safely push
```

## When to Use

- Before every `git push` to catch issues locally
- After making code changes but before committing
- To understand why pre-commit hooks are failing
- To safely apply auto-fixes from black/formatting hooks

## Critical Workflow Rule

⚠️ **NEVER PUSH WHEN PRE-COMMIT CHECKS FAIL** ⚠️

This agent enforces the project rule:
1. Run pre-commit hooks
2. If hooks fail - STOP
3. Fix all issues
4. If hooks auto-fixed files - stage and commit them
5. Only push after ALL hooks pass

## Tools Required

- `Bash` - for running pre-commit commands
- `Read` - for reading hook configuration
- `Grep` - for parsing hook output
