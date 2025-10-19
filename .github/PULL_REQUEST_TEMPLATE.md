## Description

<!-- Provide a clear and concise description of your changes -->

## Type of Change

<!-- Check all that apply -->

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Refactoring (no functional changes)
- [ ] CI/CD changes
- [ ] Dependency updates

## Related Issues

<!-- Link related issues using keywords: Fixes #123, Closes #456, Relates to #789 -->

Fixes #
Relates to #

## Changes Made

<!-- List the specific changes you made -->

-
-
-

## Testing

### Test Coverage

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Test coverage maintained at ≥ 80%
- [ ] All tests pass locally

**Coverage details:**
```bash
# Paste test coverage output
go test -v -coverprofile=coverage.out ./...
```

### Manual Testing

Describe how you tested these changes:

1.
2.
3.

**Test scenario file (if applicable):**
```yaml
# Paste test scenario used for manual testing
```

**Test output:**
```
# Paste relevant output
```

## Code Quality

- [ ] Code follows the project's style guidelines
- [ ] `gofmt` applied (code is formatted)
- [ ] `go vet` passes with no warnings
- [ ] `staticcheck` passes with no issues
- [ ] Pre-commit hooks pass locally
- [ ] No new linter warnings introduced

## Documentation

- [ ] Updated CLAUDE.md (if architecture changed)
- [ ] Updated README.md (if user-facing changes)
- [ ] Updated CONTRIBUTING.md (if development process changed)
- [ ] Added/updated code comments
- [ ] Added/updated docstrings for public functions

## Security

- [ ] No secrets or credentials committed
- [ ] No new security vulnerabilities introduced (govulncheck passes)
- [ ] Security implications have been considered
- [ ] Dependency updates are from trusted sources

## Breaking Changes

<!-- If this is a breaking change, describe the impact and migration path -->

**Impact:**


**Migration guide:**


## Conventional Commits

<!-- Your commit messages should follow conventional commits format -->

**Commit message format:**
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types used:**
- [ ] `feat:` - New feature
- [ ] `fix:` - Bug fix
- [ ] `refactor:` - Code refactoring
- [ ] `chore:` - Maintenance/build/deps
- [ ] `docs:` - Documentation only
- [ ] `test:` - Test changes
- [ ] `ci:` - CI/CD changes
- [ ] `perf:` - Performance improvement

**Breaking change notation:**
- [ ] Used `BREAKING CHANGE:` in footer or `!` after type (if applicable)

## CI Pipeline Status

<!-- All CI jobs must pass before merge -->

Expected CI results:
- [ ] Lint and Test - PASS
- [ ] Dependency Scan - PASS
- [ ] GitGuardian - PASS
- [ ] SonarCloud - PASS
- [ ] Semgrep - PASS
- [ ] Build - PASS
- [ ] Integration Tests - PASS (if applicable)

## Reviewer Notes

<!-- Any specific areas you want reviewers to focus on? -->

**Focus areas:**


**Questions for reviewers:**


## Screenshots / Outputs

<!-- If applicable, add screenshots or output examples -->

**Before:**
```
```

**After:**
```
```

## Checklist

- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published
- [ ] I have checked my code for security vulnerabilities
- [ ] I have used conventional commit format
- [ ] I have updated CLAUDE.md if architecture changed
- [ ] I have not committed any secrets or credentials

## Additional Context

<!-- Any additional information that would help reviewers -->
