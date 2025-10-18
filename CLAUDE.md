# politest - AWS IAM Policy Testing Tool

## Overview

`politest` is a single-binary Go CLI tool for testing AWS IAM policies using the `SimulateCustomPolicy` API. It supports YAML-based test scenarios with template variables, scenario inheritance, and Service Control Policy (SCP) / Resource Control Policy (RCP) merging.

## Quick Commands

```bash
# Build
go build -o politest .

# Run all checks (what pre-commit does)
gofmt -w *.go
go vet ./...
staticcheck ./...
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run a scenario
go run . --scenario scenarios/example.yml

# Run integration tests (requires AWS credentials)
cd test && bash run-tests.sh
```

## Architecture

### Core Concepts

1. **Scenarios** (`Scenario` struct in main.go:24-35)
   - YAML files defining policy tests
   - Support `extends:` for inheritance (recursive merging)
   - Can reference `policy_template` (Go template) or `policy_json` (pre-rendered)
   - Include `actions`, `resources`, `context` for simulation
   - Define `expect` map for assertions (action → decision)

2. **Variable Templating**
   - Go `text/template` for all fields: policies, actions, resources, context values
   - Variables sourced from `vars_file` (YAML) or inline `vars` map
   - Child scenarios override parent variables

3. **SCP/RCP Merging** (main.go:336-363)
   - `scp_paths` accepts globs (e.g., `scp/*.json`)
   - All matching files merged into single permissions boundary
   - Extracts `Statement` arrays and combines them
   - Used via `PermissionsBoundaryPolicyInputList` parameter

4. **AWS Integration** (main.go:115-126)
   - Uses `iam.SimulateCustomPolicy` API
   - Supports context keys with types: string, stringList, numeric, numericList, boolean, booleanList
   - Returns evaluation results: allowed, explicitDeny, implicitDeny

### File Structure

```
.
├── main.go                    # Single-file application (~460 lines)
├── go.mod, go.sum             # Dependencies: aws-sdk-go-v2, yaml.v3
├── lefthook.yml               # Pre-commit hooks (fmt, vet, staticcheck, test)
├── .github/workflows/ci.yml   # Full CI/CD with semantic-release
├── test/
│   ├── scenarios/*.yml        # 7 integration test scenarios
│   ├── policies/*.json        # Test IAM policies
│   ├── scp/*.json             # Service Control Policies
│   ├── rcp/*.json             # Resource Control Policies
│   └── run-tests.sh           # Integration test runner
└── README.md                  # User-facing documentation
```

## Key Implementation Details

### Scenario Inheritance (main.go:177-237)

The `loadScenarioWithExtends()` function recursively loads parent scenarios. Child fields override parent fields, with special handling:
- Maps (`vars`, `expect`) are deep-merged
- Slices (`actions`, `resources`, `scp_paths`) are replaced entirely
- `policy_template` and `policy_json` are mutually exclusive

### Context Type Parsing (main.go:290-307)

Maps YAML string types to AWS SDK enums:
```go
"string" → ContextKeyTypeEnumString
"stringlist" → ContextKeyTypeEnumStringList
"numeric" → ContextKeyTypeEnumNumeric
"numericlist" → ContextKeyTypeEnumNumericList
"boolean" → ContextKeyTypeEnumBoolean
"booleanlist" → ContextKeyTypeEnumBooleanList
```

**NOTE**: IpAddress and IpAddressList types are NOT supported by the AWS SDK.

### Expectations and Assertions (main.go:159-172)

The `expect` map defines expected outcomes:
```yaml
expect:
  "s3:GetObject": "allowed"
  "s3:DeleteObject": "implicitDeny"
```

Comparisons are case-insensitive using `strings.EqualFold`. Use `--no-assert` flag to skip failing on mismatches.

## Testing

### Integration Tests

Located in `test/` directory. Run with:
```bash
cd test && bash run-tests.sh
```

Requirements:
- AWS credentials configured (uses default credential chain)
- IAM permission: `iam:SimulateCustomPolicy`

Test scenarios cover:
1. Policy-only allow
2. Policy allows, SCP denies
3. Policy allows, RCP denies
4. Multiple SCPs merging
5. Explicit deny in policy
6. Template variables
7. Context conditions

### Unit Tests

Currently none. The `go test` command in pre-commit will pass if no `*_test.go` files exist.

## CI/CD Pipeline

Defined in `.github/workflows/ci.yml`:

1. **lint-and-test**: gofmt, go vet, staticcheck, go test
2. **dependency-scan**: Trivy vulnerability scanning
3. **gitguardian-scan**: Secret detection
4. **sonarcloud**: Code quality analysis
5. **semgrep**: SAST security scanning
6. **build**: Cross-platform binary builds (linux, darwin, windows)
7. **integration-tests**: Runs `test/run-tests.sh` against real AWS API
8. **release**: Semantic versioning with conventional commits

### AWS Authentication

Uses OIDC (no long-lived credentials):
- GitHub Actions assumes IAM role `GitHubActionsPolitest`
- Role ARN stored in GitHub secret `AWS_ROLE_ARN`
- Trust policy restricts to this repo's main branch

## Pre-commit Hooks

Managed by **lefthook** (Go-based, not Python pre-commit):

```yaml
pre-commit:
  parallel: true
  commands:
    fmt:        # gofmt -w {staged_files}
    vet:        # go vet ./...
    staticcheck: # staticcheck ./...
    test:       # go test -race -coverprofile=coverage.out ./...
    mod-tidy:   # go mod tidy (only if go.mod/go.sum changed)
    trailing-whitespace: # fails if found
```

Install hooks: `lefthook install`

## Common Tasks

### Adding a New Test Scenario

1. Create YAML in `test/scenarios/`:
```yaml
policy_json: "../policies/my-policy.json"
scp_paths:
  - "../scp/permissive.json"
actions:
  - "s3:GetObject"
expect:
  "s3:GetObject": "allowed"
```

2. Run: `cd test && bash run-tests.sh`

### Using Template Variables

Create `vars.yml`:
```yaml
bucket_name: my-bucket
region: us-east-1
```

Reference in scenario:
```yaml
vars_file: "vars.yml"
policy_template: "policy.json.tpl"
resources:
  - "arn:aws:s3:::{{.bucket_name}}/*"
```

### Debugging Policy Evaluation

Use `--save` to capture raw AWS response:
```bash
go run . --scenario test.yml --save /tmp/response.json
```

Inspect `MatchedStatements` to see which policy statements applied.

## Important Notes

- **Case-Insensitive Decisions**: Use `strings.EqualFold` for decision comparisons (staticcheck SA6005).
- **No IpAddress Context Types**: The AWS SDK v2 for Go does not include `ContextKeyTypeEnumIpAddress` or `ContextKeyTypeEnumIpAddressList`.
- **Glob Expansion**: SCP paths support globs (`scp/*.json`), expanded relative to scenario file location.

## Dependencies

- `github.com/aws/aws-sdk-go-v2/config` - AWS configuration
- `github.com/aws/aws-sdk-go-v2/service/iam` - IAM client
- `gopkg.in/yaml.v3` - YAML parsing

No external CLI tools required beyond Go toolchain.

## Troubleshooting

### CI Fails on gofmt
Run `gofmt -w *.go` locally and commit.

### Integration Tests Fail with "AccessDenied"
Ensure AWS credentials have `iam:SimulateCustomPolicy` permission. The API is read-only and doesn't modify resources.

### Pre-commit Hooks Not Running
Check `.git/hooks/pre-commit` exists. Run `lefthook install` to reinstall.

### staticcheck Warning SA6005
Use `strings.EqualFold(a, b)` instead of `strings.ToLower(a) == strings.ToLower(b)`.
