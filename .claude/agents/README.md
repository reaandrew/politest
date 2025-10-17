---
name: Agent Directory Documentation
description: Documentation index for all Claude Code agents in this project. This is not an executable agent.
---

# Claude Code Agents

This directory contains specialized Claude Code agents designed to improve the development workflow for this project. Each agent focuses on a specific task and follows the principle of **analyze first, act after approval** to respect project guardrails.

## Quick Reference

| Agent | Trigger | Priority | Purpose |
|-------|---------|----------|---------|
| [CI Failure Analyzer](#ci-failure-analyzer) | `@claude analyze-ci` | 🔴 High | Analyze CI failures without making changes |
| [Pre-commit Validator](#pre-commit-validator) | `@claude validate` | 🔴 High | Validate against all pre-commit hooks locally |
| [Workflow Debugger](#workflow-debugger) | `@claude debug-workflow` | 🔴 High | Debug GitHub Actions workflow issues |
| [Coverage Guardian](#coverage-guardian) | `@claude check-coverage` | 🟡 Medium | Ensure 80% test coverage threshold |
| [Commit Formatter](#commit-formatter) | `@claude format-commit` | 🟡 Medium | Generate conventional commit messages |
| [SonarCloud Pre-flight](#sonarcloud-pre-flight) | `@claude sonar-check` | 🟡 Medium | Predict SonarCloud quality gate results |
| [Secret Pre-scanner](#secret-pre-scanner) | `@claude scan-secrets` | 🟢 Nice | Enhanced secret detection |
| [Auto-fix Post-mortem](#auto-fix-post-mortem) | `@claude autofix-report` | 🟢 Nice | Analyze auto-fix effectiveness |
| [Dependency Validator](#dependency-validator) | `@claude update-deps` | 🟢 Nice | Safely update dependencies |
| [Release Notes](#release-notes) | `@claude release-notes` | 🟢 Nice | Generate release notes from commits |

## Agent Descriptions

### CI Failure Analyzer
**File**: [`ci-failure-analyzer.md`](ci-failure-analyzer.md)
**Trigger**: `@claude analyze-ci`
**Priority**: 🔴 High

Deep analysis of CI failures before attempting fixes. Categorizes errors, assesses blast radius, and provides structured reports without making any code changes.

**Use when**:
- CI fails and you need to understand why
- Investigating intermittent test failures
- Before triggering auto-fix workflow

**Key features**:
- Root cause categorization (syntax, tests, coverage, quality, security)
- Flakiness detection with confidence levels
- Blast radius assessment
- No code changes - analysis only

---

### Pre-commit Validator
**File**: [`precommit-validator.md`](precommit-validator.md)
**Trigger**: `@claude validate`
**Priority**: 🔴 High

Validates all changes against pre-commit hooks before pushing. Enforces the critical project rule: **NEVER PUSH WHEN PRE-COMMIT CHECKS FAIL**.

**Use when**:
- Before every `git push`
- After making code changes
- To understand pre-commit hook failures

**Key features**:
- Runs all pre-commit hooks (flake8, black, pytest, gitguardian)
- Handles auto-fixes from formatting hooks
- Re-runs until stable
- Detailed reporting with remediation steps

---

### Workflow Debugger
**File**: [`workflow-debugger.md`](workflow-debugger.md)
**Trigger**: `@claude debug-workflow`
**Priority**: 🔴 High

Debug GitHub Actions workflow failures, permission issues, and configuration problems. Essential for maintaining CI/CD pipeline health.

**Use when**:
- GitHub Actions workflow fails
- Getting permission errors
- After updating workflow configuration
- Understanding cryptic workflow errors

**Key features**:
- Fetches and parses workflow logs
- Identifies permission, syntax, and tool failures
- Compares with successful runs
- Validates workflow YAML syntax
- Suggests fixes with explanations

---

### Coverage Guardian
**File**: [`coverage-guardian.md`](coverage-guardian.md)
**Trigger**: `@claude check-coverage`
**Priority**: 🟡 Medium

Ensures new code maintains the 80% test coverage threshold. Analyzes coverage gaps and suggests specific tests to write.

**Use when**:
- After adding new functions/methods
- Before committing to verify 80% threshold
- When pre-commit hook fails on coverage

**Key features**:
- Identifies uncovered lines and branches
- Suggests specific test cases with examples
- Prioritizes tests by impact
- Effort estimation to reach threshold
- Never writes tests without approval

---

### Commit Formatter
**File**: [`commit-formatter.md`](commit-formatter.md)
**Trigger**: `@claude format-commit`
**Priority**: 🟡 Medium

Generates properly formatted conventional commit messages that enable semantic-release versioning.

**Use when**:
- Before every commit
- When commit-msg hook rejects message
- To ensure semantic-release works correctly

**Key features**:
- Analyzes staged changes
- Determines commit type (feat/fix/docs/etc)
- Detects breaking changes
- Explains semantic versioning impact
- Includes Co-Authored-By for Claude

**Versioning impact**:
- `feat:` → Minor (0.1.0 → 0.2.0)
- `fix:` → Patch (0.1.0 → 0.1.1)
- `feat!:` → Major (0.1.0 → 1.0.0)

---

### SonarCloud Pre-flight
**File**: [`sonar-preflight.md`](sonar-preflight.md)
**Trigger**: `@claude sonar-check`
**Priority**: 🟡 Medium

Predicts SonarCloud quality gate results before pushing to main branch. Catches code quality issues locally.

**Use when**:
- Before merging PR to main
- After SonarCloud fails in CI
- Proactively during development

**Key features**:
- Checks coverage (≥ 80%)
- Analyzes code smells and complexity
- Calculates duplication (< 3%)
- Basic security scanning
- Maintainability rating estimation

**Quality gates**:
| Metric | Threshold | Impact |
|--------|-----------|--------|
| Coverage | ≥ 80% | BLOCKER |
| Duplicated Lines | < 3% | BLOCKER |
| Code Smells | Varies | WARNING |

---

### Secret Pre-scanner
**File**: [`secret-prescanner.md`](secret-prescanner.md)
**Trigger**: `@claude scan-secrets`
**Priority**: 🟢 Nice to Have

Comprehensive secret detection beyond the pre-commit hook. Extra security layer to prevent credentials from entering the repository.

**Use when**:
- Before committing sensitive changes
- After adding configuration files
- Before pushing to public repositories
- Paranoid mode for critical changes

**Key features**:
- GitGuardian ggshield scanning
- Extended pattern matching
- Environment file checking
- URL credential detection
- Git history scanning (optional)

**Risk levels**:
- 🔴 CRITICAL: Real secrets → BLOCK
- 🟠 HIGH: Likely secrets → BLOCK
- 🟡 MEDIUM: Possible secrets → WARN
- 🟢 LOW: Test credentials → VERIFY

---

### Auto-fix Post-mortem
**File**: [`autofix-postmortem.md`](autofix-postmortem.md)
**Trigger**: `@claude autofix-report`
**Priority**: 🟢 Nice to Have

Analyzes auto-fix workflow effectiveness and identifies improvement opportunities. Provides metrics for continuous improvement.

**Use when**:
- Weekly/monthly effectiveness reviews
- After making changes to auto-fix workflow
- To identify improvement opportunities
- To justify auto-fix system value

**Key features**:
- Success rate calculation (target > 80%)
- Average time to fix tracking (target < 5m)
- CI trigger success monitoring (target > 90%)
- Failure pattern analysis
- Trend comparison (week-over-week)
- Actionable improvement recommendations

---

### Dependency Validator
**File**: [`dep-update-validator.md`](dep-update-validator.md)
**Trigger**: `@claude update-deps`
**Priority**: 🟢 Nice to Have

Safely updates Python dependencies while ensuring all tests pass and CI guardrails remain intact.

**Use when**:
- Monthly dependency maintenance
- After security advisories
- Before major feature work
- When CI fails due to dependency drift

**Key features**:
- Checks for outdated packages
- Reviews changelogs for breaking changes
- Updates one dependency at a time
- Tests after each update
- Automatic rollback on failure
- Prioritizes security updates

**Update strategy**:
1. Security patches (always apply)
2. Patch updates (low risk)
3. Minor updates (medium risk)
4. Major updates (high risk, review first)

---

### Release Notes
**File**: [`release-notes.md`](release-notes.md)
**Trigger**: `@claude release-notes`
**Priority**: 🟢 Nice to Have

Generates comprehensive, user-friendly release notes from conventional commits. Complements semantic-release automation.

**Use when**:
- Before creating GitHub release
- After merging to main
- For changelog generation
- For user communication

**Key features**:
- Parses conventional commits
- Groups by category (Features, Fixes, Breaking Changes)
- Identifies contributors
- Links to commits, PRs, issues
- Generates statistics
- Markdown formatted for GitHub

**Sections included**:
- 🎉 Features
- 🐛 Bug Fixes
- 💥 Breaking Changes
- 📚 Documentation
- 🔧 Internal Changes
- 👥 Contributors
- 📊 Statistics

---

## Usage Patterns

### Daily Development Workflow

```bash
# 1. Make code changes
vim src/acronymcreator/core.py

# 2. Check test coverage
@claude check-coverage

# 3. Write suggested tests (if needed)
# ... write tests ...

# 4. Validate all pre-commit hooks
@claude validate

# 5. Generate commit message
@claude format-commit

# 6. Review and commit
git commit -m "<generated message>"

# 7. Push
git push
```

### Pre-merge to Main Workflow

```bash
# 1. Check SonarCloud quality gates
@claude sonar-check

# 2. Fix any code smells if needed
# ... refactor ...

# 3. Scan for secrets (paranoid mode)
@claude scan-secrets

# 4. Validate everything
@claude validate

# 5. Merge to main
git checkout main
git merge feature-branch
git push
```

### CI Failure Recovery Workflow

```bash
# 1. Analyze the failure
@claude analyze-ci

# 2. If workflow issue
@claude debug-workflow

# 3. Fix manually or wait for auto-fix
# Auto-fix workflow will trigger automatically

# 4. Review auto-fix effectiveness
@claude autofix-report
```

### Monthly Maintenance Workflow

```bash
# 1. Update dependencies
@claude update-deps

# 2. Review auto-fix metrics
@claude autofix-report

# 3. Check if updates broke anything
@claude validate

# 4. Commit dependency updates
@claude format-commit
```

### Release Workflow

```bash
# 1. Generate release notes
@claude release-notes

# 2. Review notes
# ... review ...

# 3. Merge to main (triggers semantic-release)
git checkout main
git merge develop
git push

# 4. semantic-release creates tag and GitHub release
# (Automatic via CI)
```

## Agent Design Principles

All agents follow these principles:

1. **Analyze First, Act After Approval**
   - Never make changes without user approval
   - Report findings, suggest actions, wait for confirmation

2. **Respect Project Guardrails**
   - Never bypass pre-commit hooks
   - Never commit when tests fail
   - Never push secrets
   - Always maintain 80% coverage

3. **Provide Actionable Insights**
   - Specific line numbers and file paths
   - Concrete recommendations with examples
   - Confidence levels for assessments

4. **Fail Safe**
   - Rollback on failures
   - Never leave repository in broken state
   - Always validate before committing

5. **Educate and Explain**
   - Explain why, not just what
   - Link to documentation
   - Help users learn best practices

## Tools Required

All agents use a common set of tools:

- **Bash**: Running git, pytest, pre-commit, gh CLI commands
- **Read**: Reading files and configuration
- **Write**: Creating new files (when approved)
- **Edit**: Modifying existing files (when approved)
- **Grep**: Searching code and logs
- **Glob**: Finding files by pattern
- **WebFetch**: Fetching GitHub API data and documentation

## Integration with Project Guardrails

These agents integrate with existing project guardrails:

### Pre-commit Hooks
- **GitGuardian**: Secret scanning
- **pytest**: Test coverage ≥ 80%
- **flake8**: Linting
- **black**: Code formatting
- **conventional-commit**: Commit message format

### CI Pipeline
- **lint-and-test**: Re-runs all pre-commit checks
- **gitguardian-scan**: Full repository scanning
- **sonarcloud**: Code quality analysis (main only)
- **build**: Package validation
- **release**: Automated versioning (main only)

### Agents Complement Guardrails
- Pre-commit hooks: Enforce (blocking)
- CI pipeline: Validate (blocking)
- Agents: Proactive assistance (advisory)

## Contributing New Agents

To add a new agent:

1. Create agent documentation file in `.claude/agents/`
2. Follow existing agent structure:
   - Purpose
   - Trigger phrase
   - What it does (numbered steps)
   - Important rules
   - Usage example
   - When to use
   - Tools required
3. Add to Quick Reference table in this README
4. Add to appropriate workflow section
5. Test agent thoroughly before committing

## Version History

- **v1.0.0** (2025-10-03): Initial agent collection
  - 10 agents covering CI, testing, security, releases
  - Comprehensive workflow integration
  - Documentation and usage patterns

## Questions?

For questions about specific agents, see their individual documentation files. For general questions about the agent system, see the main project documentation in `/CLAUDE.md`.
