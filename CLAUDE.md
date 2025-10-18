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

1. **Scenarios** (`Scenario` struct in main.go:24-45)
   - YAML files defining policy tests in two formats:
     - **Legacy format**: Uses `actions`, `resources`, and `expect` map
     - **Collection format**: Uses `tests` array with named test cases
   - Support `extends:` for inheritance (recursive merging)
   - Can reference `policy_template` (Go template) or `policy_json` (pre-rendered)
   - Both formats support template variables and SCP/RCP merging

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
│   ├── scenarios/*.yml        # 18 integration test scenarios
│   ├── policies/*.json        # Test IAM policies (identity + resource policies + templates)
│   ├── vars/*.yml             # Variable files for template rendering
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
1. **01-07**: Legacy format - Basic policy evaluation, SCPs, RCPs, explicit denies
2. **08-09**: Collection format - Named tests with SCPs
3. **10-11**: Context conditions - IP, MFA, tags, stringList
4. **12-13**: Resource policies - Cross-account access, test-level overrides
5. **14-16**: Variable templating - vars_file, extends:, variable overrides
6. **17**: RCP in collection format
7. **18**: Comprehensive - All features combined (inheritance, templates, resource policies, SCPs, cross-account)

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

**Legacy Format** (backwards compatible):
```yaml
policy_json: "../policies/my-policy.json"
scp_paths:
  - "../scp/permissive.json"
actions:
  - "s3:GetObject"
  - "s3:PutObject"
resources:
  - "arn:aws:s3:::test-bucket/*"
expect:
  "s3:GetObject": "allowed"
  "s3:PutObject": "allowed"
```

**Collection Format** (recommended for new scenarios):
```yaml
policy_json: "../policies/my-policy.json"
scp_paths:
  - "../scp/permissive.json"

tests:
  - name: "GetObject should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::test-bucket/*"
    expect: "allowed"

  - name: "PutObject should be allowed"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::test-bucket/*"
    expect: "allowed"

  - name: "DeleteBucket should be denied"
    action: "s3:DeleteBucket"
    resource: "arn:aws:s3:::test-bucket"
    expect: "implicitDeny"
```

**Key differences:**
- Collection format provides descriptive test names (optional)
- Test names are optional - omit for auto-generated "action on resource" format
- Each test can have different resources
- Tests can have individual context conditions
- Better output showing which specific test failed
- Easier to understand test intent

**Optional test names:**
```yaml
tests:
  # Named test
  - name: "GetObject should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"

  # Unnamed test - displays as "s3:PutObject on arn:aws:s3:::bucket/*"
  - action: "s3:PutObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
```

Failure output format:
- Named tests: `✗ FAIL: expected allowed, got implicitDeny (matched: ...)`
- Unnamed tests: `✗ FAIL: s3:GetObject on arn:aws:s3:::bucket/* failed: expected allowed, got implicitDeny`

Run tests: `cd test && bash run-tests.sh`

### Using Context Conditions

Context conditions allow testing policies with IAM condition keys:

```yaml
policy_json: "../policies/policy-with-conditions.json"

tests:
  - name: "Access allowed from trusted IP"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::secure-bucket/*"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["10.0.1.50"]
    expect: "allowed"

  - name: "Access denied from untrusted IP"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::secure-bucket/*"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["192.168.1.1"]
    expect: "implicitDeny"

  # Multiple context keys
  - action: "s3:DeleteObject"
    resource: "arn:aws:s3:::secure-bucket/*"
    context:
      - ContextKeyName: "aws:MultiFactorAuthPresent"
        ContextKeyType: "boolean"
        ContextKeyValues: ["true"]
      - ContextKeyName: "aws:PrincipalTag/Department"
        ContextKeyType: "string"
        ContextKeyValues: ["Engineering"]
    expect: "allowed"
```

**Supported context types:**
- `string` - Single string value
- `stringList` - Multiple string values
- `numeric` - Numeric value
- `numericList` - Multiple numeric values
- `boolean` - Boolean value (true/false)
- `booleanList` - Multiple boolean values

**Note:** AWS SimulateCustomPolicy has limitations in condition evaluation. Some complex conditions may not evaluate as expected in simulation.

### Resource Policies and Cross-Account Testing

Test how identity policies and resource policies interact, essential for S3 buckets, KMS keys, SNS/SQS, and other resource-based policies:

```yaml
# Alice's identity policy
policy_json: "../policies/user-alice-identity.json"

# S3 bucket's resource policy
resource_policy_json: "../policies/s3-bucket-policy.json"

# Simulate as Alice from account 111111111111
caller_arn: "arn:aws:iam::111111111111:user/alice"

# Bucket owned by account 222222222222
resource_owner: "arn:aws:iam::222222222222:root"

tests:
  - name: "Cross-account read allowed by both policies"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    expect: "allowed"

  - name: "Write denied by resource policy"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    expect: "explicitDeny"
```

**Test-level overrides:**

Each test can override scenario-level settings:

```yaml
caller_arn: "arn:aws:iam::111111111111:user/alice"  # Default

tests:
  - name: "Test as Alice"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"

  - name: "Test as Bob (override)"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    caller_arn: "arn:aws:iam::111111111111:user/bob"  # Override for this test
    resource_policy_json: "../policies/different-policy.json"  # Different policy
    expect: "implicitDeny"
```

**Supported parameters:**
- `resource_policy_json` / `resource_policy_template` - Resource-based policy
- `caller_arn` - IAM principal to simulate as (required when using resource policies)
- `resource_owner` - Account ARN that owns the resource (for cross-account)
- `resource_handling_option` - EC2 scenario type (EC2-VPC-InstanceStore, etc.)

### Using Template Variables

politest supports Go template rendering across all fields - policies, actions, resources, context values, and ARNs.

**Create a variable file** (`vars/common-vars.yml`):
```yaml
account_id: "123456789012"
bucket_name: "my-app-data"
department: "Engineering"
region: "us-east-1"
```

**Create a policy template** (`policies/policy.json.tpl`):
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["s3:GetObject", "s3:PutObject"],
    "Resource": "arn:aws:s3:::{{.bucket_name}}/*",
    "Condition": {
      "StringEquals": {
        "aws:PrincipalTag/Department": "{{.department}}"
      }
    }
  }]
}
```

**Reference in scenario:**
```yaml
vars_file: "../vars/common-vars.yml"
policy_template: "../policies/policy.json.tpl"
caller_arn: "arn:aws:iam::{{.account_id}}:user/alice"

tests:
  - name: "GetObject with correct tags"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/data.txt"
    context:
      - ContextKeyName: "aws:PrincipalTag/Department"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.department}}"]
    expect: "allowed"
```

### Scenario Inheritance with extends:

Reuse base scenarios and override specific settings:

**Base scenario** (`scenarios/14-base-with-variables.yml`):
```yaml
vars_file: "../vars/common-vars.yml"
policy_template: "../policies/s3-templated.json.tpl"
caller_arn: "arn:aws:iam::{{.account_id}}:user/alice"

tests:
  - name: "GetObject allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/*"
    expect: "allowed"
```

**Child scenario with SCP** (`scenarios/15-extends-add-scp.yml`):
```yaml
extends: "14-base-with-variables.yml"

# Add organizational SCP boundary
scp_paths:
  - "../scp/deny-s3-write.json"

# Override expectations - writes now denied
tests:
  - name: "GetObject still allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/*"
    expect: "allowed"

  - name: "PutObject denied by SCP"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/*"
    expect: "explicitDeny"
```

**Child scenario with variable override** (`scenarios/16-extends-override-vars.yml`):
```yaml
extends: "14-base-with-variables.yml"

# Override specific variables (merges with vars_file)
vars:
  department: "Finance"
  bucket_name: "finance-data"

# Tests from parent are re-run with new variable values
```

**Inheritance behavior:**
- Maps (`vars`, `expect`) are **deep-merged** (child adds/overrides keys)
- Arrays (`actions`, `resources`, `scp_paths`, `tests`) are **replaced entirely**
- `policy_template` and `policy_json` are mutually exclusive
- Child variables override parent variables
- Supports recursive inheritance (child extends child extends base)

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
