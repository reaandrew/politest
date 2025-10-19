<p align="center">
  <img src="images/logo.png" alt="politest logo" width="200"/>
</p>

# politest

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=reaandrew_politest&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=reaandrew_politest)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=reaandrew_politest&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=reaandrew_politest)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=reaandrew_politest&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=reaandrew_politest)
[![Semgrep](https://img.shields.io/badge/Semgrep-Enabled-blueviolet?logo=semgrep)](https://semgrep.dev/)

A single-binary Go tool for testing AWS IAM policies using scenario-based YAML configurations.

## Features

- **YAML-based scenarios**
  - Inheritance via `extends:`
- **Go template support**
  - Dynamic values (`{{ .variable }}`)
- **Policy templates**
  - Or pre-rendered JSON policies
- **SCP merging**
  - From multiple files/globs into permissions boundaries
- **AWS IAM SimulateCustomPolicy integration**
- **Expectation assertions**
  - For CI/CD integration
- **Clean table output**
  - With optional raw JSON export

## ⚠️ Understanding What politest Tests

**politest is a pre-deployment validation tool that helps you catch IAM policy issues early, but it is NOT a replacement for integration testing in real AWS environments.**

### What politest Does

politest uses AWS's `SimulateCustomPolicy` API to evaluate policies **before deployment**. This provides:

✅ **Fast feedback loop**
  - Test policy changes in seconds without deploying

✅ **Blended testing**
  - See how identity policies interact with SCPs/RCPs

✅ **Fail fast**
  - Catch obvious misconfigurations early in development

✅ **CI/CD integration**
  - Automated policy validation on every commit

### Important Limitations

⚠️ **politest "bends the rules" for testing convenience:**

- **SCPs/RCPs in SimulateCustomPolicy**
  - The API wasn't designed for testing organizational policies alongside identity policies
  - politest uses the `PermissionsBoundaryPolicyInputList` parameter to simulate SCP/RCP behavior
  - This **approximates** real-world behavior but may not be 100% accurate

- **Simulation vs Reality**
  - `SimulateCustomPolicy` provides a **best-effort simulation**
  - Some complex conditions, resource policy interactions, and edge cases may behave differently in production

- **Missing Context**
  - Real AWS environments have additional factors not fully captured in simulation
  - Resource ownership, trust policies, session policies, permission boundaries

### What You Still Need

✅ **Integration testing in actual AWS accounts**
  - Deploy policies to dev/staging and test real resource access

✅ **Production validation**
  - Verify permissions work as expected with real workloads

✅ **Security reviews**
  - Have security teams review policies before production deployment

**Remember:** politest helps you **fail faster during development** by catching obvious mistakes before deployment. Use it as **unit tests for IAM policies** - essential for development velocity, but always validate with real integration tests in actual AWS environments.

## Installation

```bash
# Build the binary
go build -o politest

# Or run directly
go run . --scenario path/to/scenario.yml
```

## Quick Start

### 1. Create a base scenario (`scenarios/_common.yml`)

```yaml
vars:
  account_id: "123456789012"
  region: "us-east-1"

scp_paths:
  - "../scp/010-base.json"
  - "../scp/020-guardrails.json"

context:
  - ContextKeyName: "aws:RequestedRegion"
    ContextKeyValues: ["{{ .region }}"]
    ContextKeyType: "string"
```

### 2. Create a specific test scenario (`scenarios/athena_test.yml`)

```yaml
extends: "_common.yml"

vars:
  workgroup: "primary"

policy_template: "../policies/athena_policy.json.tmpl"

actions:
  - "athena:BatchGetNamedQuery"
  - "athena:GetQueryExecution"

resources:
  - "arn:aws:athena:{{ .region }}:{{ .account_id }}:workgroup/{{ .workgroup }}"

context:
  - ContextKeyName: "aws:CalledVia"
    ContextKeyValues: ["athena.amazonaws.com"]
    ContextKeyType: "stringList"

expect:
  "athena:BatchGetNamedQuery": "allowed"
  "athena:GetQueryExecution": "allowed"
```

### 3. Create a policy template (`policies/athena_policy.json.tmpl`)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AthenaAccess",
      "Effect": "Allow",
      "Action": [
        "athena:BatchGetNamedQuery",
        "athena:GetQueryExecution"
      ],
      "Resource": "arn:aws:athena:{{ .region }}:{{ .account_id }}:workgroup/{{ .workgroup }}",
      "Condition": {
        "StringEquals": {
          "aws:RequestedRegion": "{{ .region }}"
        }
      }
    }
  ]
}
```

### 4. Create SCP files (optional, `scp/010-base.json`)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "*",
      "Resource": "*"
    }
  ]
}
```

### 5. Run the test

```bash
# Run with expectations (fails on mismatch)
./politest --scenario scenarios/athena_test.yml

# Run without assertions
./politest --scenario scenarios/athena_test.yml --no-assert

# Save raw AWS response
./politest --scenario scenarios/athena_test.yml --save /tmp/response.json
```

## Usage

```bash
politest [flags]

Flags:
  --scenario string    Path to scenario YAML (required)
  --save string        Path to save raw JSON response (optional)
  --no-assert          Do not fail on expectation mismatches (optional)
  --no-warn            Suppress SCP/RCP simulation approximation warning (optional)
```

## Scenario Configuration

### Required Fields

- **One of:**
  - `policy_template`
    - Path to a Go template file that renders to JSON
  - `policy_json`
    - Path to a pre-rendered JSON policy file
- `actions`
  - List of IAM actions to test (can use templates)

### Optional Fields

- `extends`
  - Path to parent scenario (supports inheritance)
- `vars_file`
  - Path to YAML file with variables
- `vars`
  - Inline variables (overrides vars_file)
- `scp_paths`
  - List of SCP file paths or globs to merge
- `resources`
  - List of resource ARNs (can use templates)
- `context`
  - List of context entries for conditions
- `expect`
  - Map of action → expected decision ("allowed" or "denied")

### Inheritance with `extends:`

Child scenarios inherit all fields from parent and can override:

- **Variables**
  - Deep-merged (child overrides parent)
- **Other fields**
  - Completely replaced (not merged)
- **Relative paths**
  - Resolved from the scenario file's directory

### Variables

Variables can be defined in three places (priority order):

1. **Inline `vars:` in the scenario**
2. **External `vars_file:` YAML**
3. **Inherited from parent via `extends:`**

Use Go template syntax: `{{ .variable_name }}`

### Context Entries

```yaml
context:
  - ContextKeyName: "aws:RequestedRegion"
    ContextKeyValues: ["us-east-1", "eu-west-1"]
    ContextKeyType: "stringList"  # string, stringList, numeric, numericList, boolean, booleanList
```

**Supported Context Types:**

- `string`
  - Single string value
- `stringList`
  - List of strings
- `numeric`
  - Single numeric value
- `numericList`
  - List of numeric values
- `boolean`
  - Single boolean value
- `booleanList`
  - List of boolean values

**Note:** IpAddress and IpAddressList types are not supported by the AWS SDK.

### SCP Merging

Multiple SCP files are merged into a single permissions boundary:

```yaml
scp_paths:
  - "../scp/010-base.json"
  - "../scp/*.json"  # globs supported
  - "../scp/specific-restriction.json"
```

All statements from all files are combined into one policy document.

## Output

### Table Output

```
Action                         Decision  Matched (details)
----------------------------  --------  ----------------------------------------
athena:BatchGetNamedQuery     allowed   PolicyInputList.1
athena:GetQueryExecution      allowed   PolicyInputList.1
```

### Exit Codes

- `0`
  - Success (all expectations met or no expectations)
- `1`
  - Error (invalid scenario, AWS error, etc.)
- `2`
  - Expectation failures (unless `--no-assert` used)

## Examples

### Example 1: Simple Policy Test

```yaml
# scenarios/s3_read.yml
policy_json: "../policies/s3_read.json"

actions:
  - "s3:GetObject"
  - "s3:ListBucket"

resources:
  - "arn:aws:s3:::my-bucket/*"

expect:
  "s3:GetObject": "allowed"
  "s3:ListBucket": "denied"
```

### Example 2: Template with Variables

```yaml
# scenarios/dynamodb_test.yml
vars:
  table_name: "users-table"
  region: "us-west-2"

policy_template: "../policies/dynamodb.json.tmpl"

actions:
  - "dynamodb:GetItem"
  - "dynamodb:PutItem"

resources:
  - "arn:aws:dynamodb:{{ .region }}:{{ .account_id }}:table/{{ .table_name }}"
```

### Example 3: With SCPs and Context

```yaml
# scenarios/ec2_restricted.yml
extends: "_common.yml"

policy_template: "../policies/ec2.json.tmpl"

scp_paths:
  - "../scp/region-restriction.json"
  - "../scp/instance-type-restriction.json"

actions:
  - "ec2:RunInstances"

resources:
  - "*"

context:
  - ContextKeyName: "aws:RequestedRegion"
    ContextKeyValues: ["us-east-1"]
    ContextKeyType: "string"
  - ContextKeyName: "ec2:InstanceType"
    ContextKeyValues: ["t3.micro"]
    ContextKeyType: "string"

expect:
  "ec2:RunInstances": "denied"  # Should be blocked by SCP
```

## AWS Credentials

The tool uses the AWS SDK v2 default credential chain:

- **Environment variables**
  - `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- **Shared credentials file**
  - `~/.aws/credentials`
- **IAM role**
  - When running on EC2/ECS/Lambda

Required IAM permission: `iam:SimulateCustomPolicy`

## Development

### Running Tests

```bash
# Run unit tests (if any exist)
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run integration tests (requires AWS credentials)
cd test && bash run-tests.sh
```

Integration tests are located in the `test/` directory and cover:
- Policy-only allow scenarios
- Policy allows, SCP denies
- Policy allows, RCP denies
- Multiple SCPs merging
- Explicit deny in policy
- Template variables
- Context conditions

### Pre-commit Hooks

This project uses [lefthook](https://github.com/evilmartians/lefthook) for Git hooks:

```bash
# Install hooks
lefthook install

# Hooks run automatically on commit:
# - gofmt -w (auto-format)
# - go vet (static analysis)
# - staticcheck (linting)
# - go test (unit tests)
# - go mod tidy (dependency cleanup)
# - trailing whitespace check
```

### CI/CD

The GitHub Actions workflow (`.github/workflows/ci.yml`) runs:
- Linting and testing
- Dependency scanning (Trivy)
- Secret detection (GitGuardian)
- Code quality analysis (SonarCloud)
- Security scanning (Semgrep)
- Cross-platform builds
- Integration tests against real AWS API
- Semantic versioning releases

## Tips

1. **Organize scenarios**
   - Use `_common.yml` for shared config, extend in specific tests

2. **Use templates**
   - Policy templates with variables make tests reusable across accounts/regions

3. **CI Integration**
   - Use `expect:` assertions and check exit codes

4. **Debug**
   - Use `--save` to inspect raw AWS responses and examine `MatchedStatements`

5. **Glob SCPs**
   - Use wildcards to merge multiple SCP files automatically

6. **Case-insensitive decisions**
   - Expected decisions are compared case-insensitively (e.g., "allowed" matches "Allowed")

## Project Structure Example

```
.
├── politest              # binary
├── scenarios/
│   ├── _common.yml       # base configuration
│   ├── athena_test.yml
│   ├── s3_test.yml
│   └── ec2_test.yml
├── policies/
│   ├── athena_policy.json.tmpl
│   ├── s3_policy.json
│   └── ec2_policy.json.tmpl
└── scp/
    ├── 010-base.json
    ├── 020-region-restriction.json
    └── 030-service-restriction.json
```

## License

MIT
