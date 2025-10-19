# Contributing to politest

Thank you for your interest in contributing to politest! This guide will help you get started with development.

## Table of Contents

- [Development Setup](#development-setup)
- [Running Tests](#running-tests)
- [Commit Message Format](#commit-message-format)
- [Pre-commit Hooks](#pre-commit-hooks)
- [CI Pipeline](#ci-pipeline)
- [Adding Test Scenarios](#adding-test-scenarios)
- [Debugging](#debugging)
- [Pull Request Process](#pull-request-process)

## Development Setup

### Prerequisites

- **Go 1.24+** - [Install Go](https://go.dev/doc/install)
- **AWS Credentials** - For integration tests (uses default credential chain)
- **lefthook** - Git hooks manager (installed automatically via Go)

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/reaandrew/politest.git
cd politest

# Install dependencies
go mod download

# Install pre-commit hooks
go install github.com/evilmartians/lefthook@latest
lefthook install

# Build the binary
go build -o politest .

# Verify setup
./politest --help
```

## Running Tests

### Unit Tests

```bash
# Run all tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage in browser
go tool cover -html=coverage.out

# Coverage threshold: 80% minimum
```

### Integration Tests

Integration tests run against the real AWS IAM API:

```bash
cd test
bash run-tests.sh
```

**Requirements:**
- AWS credentials configured (IAM permission: `iam:SimulateCustomPolicy`)
- The API is read-only and doesn't modify resources

### Quick Checks (Pre-commit)

Run the same checks that pre-commit hooks run:

```bash
# Format code
gofmt -w *.go internal/*.go

# Vet code
go vet ./...

# Static analysis
staticcheck ./...

# Run tests
go test -race -coverprofile=coverage.out -covermode=atomic ./...
```

## Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/) for automated versioning and changelog generation.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature (triggers minor version bump: 1.0.0 → 1.1.0)
- `fix`: Bug fix (triggers patch version bump: 1.0.0 → 1.0.1)
- `refactor`: Code refactoring without feature/fix changes
- `chore`: Maintenance tasks (dependencies, configs, build)
- `docs`: Documentation changes only
- `test`: Test additions or changes
- `ci`: CI/CD configuration changes
- `perf`: Performance improvements
- `style`: Code style/formatting changes (not CSS)

### Examples

```bash
feat: add support for resource-based policies
fix: correct context key type mapping for boolean values
refactor: extract helper functions to reduce cognitive complexity
chore: upgrade to Go 1.24
docs: update README with new scenario examples
test: add integration tests for SCP merging
ci: add SonarCloud quality gate checks
```

### Breaking Changes

For breaking changes, add `BREAKING CHANGE:` in the footer or `!` after type:

```bash
feat!: change scenario file format to YAML v2

BREAKING CHANGE: Old YAML format no longer supported.
See migration guide in docs/MIGRATION.md
```

**Version bumps:**
- `feat:` → Minor (1.0.0 → 1.1.0)
- `fix:` → Patch (1.0.0 → 1.0.1)
- `BREAKING CHANGE:` → Major (1.0.0 → 2.0.0)

### Commit Validation

- Pre-commit hook validates commit message format automatically
- Failed commits will show: "commit-msg hook rejected message"
- Commit messages must follow conventional commits spec

### Getting Help with Commits

If you need help generating a proper commit message, you can use:
- The Conventional Commit Formatter agent (if using Claude Code)
- Online tool: [commitizen](https://commitizen-tools.github.io/commitizen/)

## Pre-commit Hooks

We use **lefthook** (Go-based, not Python pre-commit) to run checks before each commit.

### Installed Hooks

```yaml
pre-commit:
  parallel: true
  commands:
    fmt:               # gofmt -w {staged_files}
    vet:               # go vet ./...
    staticcheck:       # staticcheck ./...
    test:              # go test -race -coverprofile=coverage.out ./...
    mod-tidy:          # go mod tidy (only if go.mod/go.sum changed)
    trailing-whitespace: # fails if found
```

### Hook Management

```bash
# Install hooks
lefthook install

# Run hooks manually
lefthook run pre-commit

# Skip hooks (not recommended)
git commit --no-verify
```

**Note:** Never skip hooks unless absolutely necessary. They ensure code quality and prevent CI failures.

## CI Pipeline

Our CI pipeline runs on every push and PR. Understanding it helps you debug failures.

### Pipeline Jobs

#### 1. Lint and Test
**Duration:** ~1 minute
**Runs:** gofmt, go vet, staticcheck, tests with coverage

**Common Failures:**
- Code not formatted: Run `gofmt -w *.go internal/*.go`
- Vet errors: Run `go vet ./...` locally
- Test failures: Run `go test -v ./...` to see details
- Coverage below 80%: Add tests to increase coverage

#### 2. Dependency Vulnerability Scan
**Duration:** ~30 seconds
**Runs:** govulncheck to detect known vulnerabilities

**Common Failures:**
- Outdated Go version: Update `go.mod` and `.github/workflows/ci.yml`
- Vulnerable dependencies: Run `go get -u` to update, or pin versions

**Important:** This job WILL fail the build if vulnerabilities are found (no `continue-on-error`).

#### 3. GitGuardian Scan
**Duration:** ~15 seconds
**Runs:** Secret detection in git history

**Common Failures:**
- Accidentally committed secrets (.env, API keys)
- Use GitGuardian's pre-commit hook to catch before push

#### 4. SonarCloud Analysis
**Duration:** ~1 minute
**Runs:** Code quality, security, and coverage analysis

**Quality Gate Requirements:**
- New code coverage: ≥ 80%
- Reliability rating: A (no bugs)
- Security rating: A (no vulnerabilities)
- Maintainability rating: A (no code smells)

**Common Failures:**
- Coverage too low: Add tests
- Code smells: Refactor to reduce complexity
- Cognitive complexity: Extract helper functions
- Too many parameters: Use config structs

#### 5. Semgrep Security Analysis
**Duration:** ~4 minutes
**Runs:** Static analysis security testing (SAST)

**Common Failures:**
- Security vulnerabilities detected
- Check Semgrep output for specific issues

#### 6. Build
**Duration:** ~20 seconds
**Runs:** Cross-platform builds (Linux, macOS, Windows)

**Common Failures:**
- Build errors: Usually caught by lint-and-test job first
- Platform-specific code issues

#### 7. Integration Tests
**Duration:** ~1 minute
**Runs:** Real AWS IAM API tests

**Requirements:**
- AWS credentials via OIDC (no long-lived keys)
- IAM role: GitHubActionsPolitest

**Common Failures:**
- AWS credential issues: Check OIDC setup
- API errors: Check test scenarios

#### 8. Release
**Duration:** ~30 seconds
**Runs:** Only on main branch after successful tests
**Creates:** GitHub releases with semantic versioning

**Triggered by:** Conventional commits
- `feat:` → New minor version
- `fix:` → New patch version
- `BREAKING CHANGE:` → New major version

### Checking CI Status

```bash
# List recent runs
gh run list --limit 5

# View specific run
gh run view <run-id>

# View failed job logs
gh run view <run-id> --log-failed

# Re-run failed jobs
gh run rerun <run-id> --failed
```

## Adding Test Scenarios

### Creating a New Scenario

1. Create YAML in `test/scenarios/`:

```yaml
# test/scenarios/my-test.yml
policy_json: "../policies/my-policy.json"
scp_paths:
  - "../scp/permissive.json"

# Legacy format (simple)
actions:
  - "s3:GetObject"
  - "s3:PutObject"
resources:
  - "arn:aws:s3:::my-bucket/*"
expect:
  "s3:GetObject": "allowed"
  "s3:PutObject": "explicitDeny"

# OR Collection format (recommended for complex tests)
tests:
  - name: "Allow read access"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::my-bucket/*"
    expect: "allowed"

  - name: "Deny write access"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::my-bucket/*"
    expect: "explicitDeny"
```

2. Run the test:

```bash
go run . --scenario test/scenarios/my-test.yml
```

3. Add to integration test suite:

Edit `test/run-tests.sh` and add your scenario to the test list.

### Using Template Variables

```yaml
vars_file: "vars.yml"
vars:
  bucket_name: "my-bucket"
  region: "us-east-1"

policy_template: "policy.json.tpl"
resources:
  - "arn:aws:s3:::{{.bucket_name}}/*"
```

Create `vars.yml`:
```yaml
bucket_name: my-bucket
region: us-east-1
account_id: "123456789012"
```

Create `policy.json.tpl`:
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::{{.bucket_name}}/*"
  }]
}
```

### Scenario Inheritance

```yaml
# child-scenario.yml
extends: "base-scenario.yml"  # Inherits all fields

# Override specific fields
vars:
  environment: "production"  # Adds to parent vars

actions:
  - "s3:GetObject"  # Replaces parent actions entirely
```

## Debugging

### Saving Raw AWS Response

```bash
go run . --scenario test.yml --save /tmp/response.json
```

Then inspect `/tmp/response.json` to see:
- Evaluation results
- Matched statements (which policy allowed/denied)
- Decision details

### Verbose Output

Enable verbose mode to see template rendering and variable substitution:

```bash
# Run with -v flag (if implemented)
go run . --scenario test.yml -v
```

### Common Issues

**Problem:** Test expects "allowed" but gets "implicitDeny"

**Solution:**
- Check policy has explicit Allow statement
- Verify action and resource match exactly
- Check SCPs aren't blocking the action

**Problem:** Template rendering errors

**Solution:**
- Validate YAML syntax
- Check variable names match between vars and template
- Ensure template uses Go template syntax: `{{.var_name}}`

**Problem:** Context conditions not working

**Solution:**
- Verify context key names are correct (case-sensitive)
- Check context type matches (string, stringList, numeric, etc.)
- Note: IpAddress types are NOT supported by AWS SDK

## Pull Request Process

1. **Fork and Branch**
   ```bash
   git checkout -b feature/my-feature
   # or
   git checkout -b fix/my-bugfix
   ```

2. **Make Changes**
   - Write code following existing patterns
   - Add tests (maintain 80%+ coverage)
   - Update CLAUDE.md if architecture changes

3. **Test Locally**
   ```bash
   # Run all checks
   go test -v -race ./...
   go vet ./...
   staticcheck ./...
   gofmt -w .

   # Run integration tests
   cd test && bash run-tests.sh
   ```

4. **Commit with Conventional Commits**
   ```bash
   git add .
   git commit -m "feat: add new feature description"
   ```

5. **Push and Create PR**
   ```bash
   git push origin feature/my-feature
   gh pr create --title "feat: add new feature" --body "Description..."
   ```

6. **PR Checklist**
   - [ ] Tests pass locally
   - [ ] Code coverage ≥ 80%
   - [ ] Conventional commit format used
   - [ ] CLAUDE.md updated if needed
   - [ ] Integration tests pass (if applicable)
   - [ ] No security vulnerabilities introduced

7. **CI Must Pass**
   All CI jobs must pass before merge:
   - Lint and Test ✓
   - Dependency Scan ✓
   - GitGuardian ✓
   - SonarCloud ✓
   - Semgrep ✓
   - Build ✓
   - Integration Tests ✓

8. **Review and Merge**
   - Address review feedback
   - Squash commits if requested
   - Merge when approved and CI passes

## Code Style Guidelines

### Go Conventions

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting (enforced by pre-commit)
- Keep functions under 50 lines when possible
- Cognitive complexity limit: 15 (enforced by SonarCloud)
- Maximum function parameters: 7 (use config structs for more)

### Testing

- Test files: `*_test.go`
- Table-driven tests preferred
- Use descriptive test names: `TestFunctionName_Scenario_ExpectedResult`
- Mock external dependencies (AWS API, file system)

### Documentation

- Public functions must have doc comments
- Doc comments start with function name
- Example: `// LoadScenarioWithExtends loads a scenario and recursively merges parent scenarios`

### Error Handling

- Check all errors
- Use `internal.Check(err)` for fatal errors
- Use `internal.Die(format, args...)` for user-facing errors
- Return errors for library functions

## Questions?

- Open an [issue](https://github.com/reaandrew/politest/issues)
- Check existing [discussions](https://github.com/reaandrew/politest/discussions)
- Review CLAUDE.md for technical details

Thank you for contributing to politest!
