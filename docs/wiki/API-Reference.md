# API Reference

Complete YAML schema reference for politest scenario files.

## Table of Contents

- [Scenario Schema](#scenario-schema)
- [Legacy Format Fields](#legacy-format-fields)
- [Collection Format Fields](#collection-format-fields)
- [Test Case Schema](#test-case-schema)
- [Context Entry Schema](#context-entry-schema)
- [Command Line Options](#command-line-options)

## Scenario Schema

Top-level fields available in both legacy and collection formats.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `extends` | string | No | Path to parent scenario file to inherit from |
| `vars_file` | string | No | Path to YAML file containing template variables |
| `vars` | map | No | Inline template variables (overrides vars_file) |
| `policy_template` | string | No* | Path to identity policy template (.json.tpl) |
| `policy_json` | string | No* | Path to rendered identity policy (.json) |
| `resource_policy_template` | string | No | Path to resource policy template (.json.tpl) |
| `resource_policy_json` | string | No | Path to rendered resource policy (.json) |
| `caller_arn` | string | No | IAM ARN to simulate as (e.g., user/role) |
| `resource_owner` | string | No | Account ARN that owns the resource |
| `resource_handling_option` | string | No | EC2 scenario type |
| `scp_paths` | []string | No | Paths to SCP/RCP JSON files (supports globs) |

\* Either `policy_template` or `policy_json` is required (mutually exclusive)

### Scenario Inheritance

When using `extends`, child scenarios merge with parent:

- **Maps** (`vars`, `expect`): Deep-merged (child overrides parent keys)
- **Arrays** (`actions`, `resources`, `scp_paths`, `tests`): Replaced entirely
- **Scalars**: Child overrides parent

**Example:**

```yaml
# parent.yml
vars:
  bucket: "parent-bucket"
  region: "us-east-1"
actions:
  - "s3:GetObject"
```

```yaml
# child.yml
extends: "parent.yml"
vars:
  bucket: "child-bucket"  # Overrides
  # region: us-east-1 inherited
actions:
  - "s3:PutObject"  # Replaces parent actions entirely
```

## Legacy Format Fields

Fields specific to legacy format (mutually exclusive with `tests`).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `actions` | []string | Yes* | List of IAM actions to test |
| `resources` | []string | No | List of resource ARNs (defaults to `["*"]`) |
| `context` | []ContextEntry | No | IAM condition context (global for all actions) |
| `expect` | map[string]string | Yes* | Map of action → expected decision |

\* Required in legacy format

**Example:**

```yaml
policy_json: "policy.json"
actions:
  - "s3:GetObject"
  - "s3:PutObject"
resources:
  - "arn:aws:s3:::bucket/*"
expect:
  "s3:GetObject": "allowed"
  "s3:PutObject": "implicitDeny"
```

## Collection Format Fields

Fields specific to collection format (mutually exclusive with legacy fields).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tests` | []TestCase | Yes* | Array of test cases |

\* Required in collection format

**Example:**

```yaml
policy_json: "policy.json"
tests:
  - name: "GetObject allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
```

## Test Case Schema

Fields available in each test within the `tests` array.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | Test description (auto-generated if omitted) |
| `action` | string | Yes | Single IAM action to test |
| `resource` | string | No** | Single resource ARN |
| `resources` | []string | No** | Multiple resource ARNs |
| `context` | []ContextEntry | No | IAM condition context for this test |
| `expect` | string | Yes | Expected decision: `allowed`, `implicitDeny`, or `explicitDeny` |
| `resource_policy_template` | string | No | Override resource policy template |
| `resource_policy_json` | string | No | Override resource policy |
| `caller_arn` | string | No | Override caller ARN for this test |
| `resource_owner` | string | No | Override resource owner for this test |
| `resource_handling_option` | string | No | Override handling option for this test |

\*\* Either `resource` or `resources` should be provided (defaults to `["*"]`)

### Expected Decisions

| Value | Meaning |
|-------|---------|
| `allowed` | Action is permitted by policies |
| `implicitDeny` | Action not explicitly allowed (default deny) |
| `explicitDeny` | Action explicitly denied in a Deny statement |

**Example:**

```yaml
tests:
  - name: "Cross-account read as Bob"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    caller_arn: "arn:aws:iam::111111111111:user/bob"
    resource_policy_json: "../policies/bucket-policy.json"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["10.0.1.100"]
    expect: "allowed"
```

## Context Entry Schema

IAM condition context for testing policies with Condition elements.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ContextKeyName` | string | Yes | IAM condition key (e.g., `aws:SourceIp`) |
| `ContextKeyType` | string | Yes | Value type (see below) |
| `ContextKeyValues` | []string | Yes | List of values for the context key |

### Context Key Types

| Type | AWS Type | Example Values |
|------|----------|----------------|
| `string` | String | `["10.0.1.100"]` |
| `stringList` | String (multi-value) | `["value1", "value2"]` |
| `numeric` | Numeric | `["100"]` |
| `numericList` | Numeric (multi-value) | `["100", "200"]` |
| `boolean` | Boolean | `["true"]` or `["false"]` |
| `booleanList` | Boolean (multi-value) | `["true", "false"]` |

**Note:** `ipAddress` and `ipAddressList` types are **not supported** by AWS SDK v2 for Go.

**Example:**

```yaml
context:
  - ContextKeyName: "aws:SourceIp"
    ContextKeyType: "string"
    ContextKeyValues: ["10.0.1.50"]

  - ContextKeyName: "aws:MultiFactorAuthPresent"
    ContextKeyType: "boolean"
    ContextKeyValues: ["true"]

  - ContextKeyName: "aws:PrincipalTag/Department"
    ContextKeyType: "string"
    ContextKeyValues: ["Engineering"]

  - ContextKeyName: "aws:RequestedRegion"
    ContextKeyType: "stringList"
    ContextKeyValues: ["us-east-1", "us-west-2"]
```

## Template Variables

All string fields support Go template syntax with `{{.variable}}`.

### Variable Precedence

1. Inline `vars` (highest priority)
2. `vars_file` values
3. Parent scenario variables (if using `extends`)

**Example:**

```yaml
vars_file: "vars/common.yml"  # account_id: "111111111111"
vars:
  account_id: "999999999999"  # Overrides

caller_arn: "arn:aws:iam::{{.account_id}}:user/alice"
# Result: "arn:aws:iam::999999999999:user/alice"
```

## Resource Handling Option Values

For EC2-specific scenarios (rarely used):

- `EC2-Classic-InstanceStore`
- `EC2-Classic-EBS`
- `EC2-VPC-InstanceStore`
- `EC2-VPC-InstanceStore-Subnet`
- `EC2-VPC-EBS`
- `EC2-VPC-EBS-Subnet`

**Example:**

```yaml
resource_handling_option: "EC2-VPC-EBS-Subnet"
```

## Command Line Options

### politest CLI

```
politest --scenario SCENARIO_FILE [OPTIONS]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--scenario` | string | *required* | Path to YAML scenario file |
| `--save` | string | "" | Path to save raw AWS API response JSON |
| `--no-assert` | bool | false | Don't fail on expectation mismatches |
| `--version` | bool | false | Show version and exit |

**Examples:**

```bash
# Basic usage
politest --scenario test.yml

# Save API response for debugging
politest --scenario test.yml --save /tmp/response.json

# Show all results without failing
politest --scenario test.yml --no-assert

# Check version
politest --version
```

## File Path Resolution

All file paths in scenarios are resolved relative to the scenario file's directory.

**Example directory structure:**

```
project/
├── scenarios/
│   ├── test.yml
│   └── base/
│       └── parent.yml
├── policies/
│   └── policy.json
└── vars/
    └── common.yml
```

**In `scenarios/test.yml`:**

```yaml
extends: "base/parent.yml"      # Relative to scenarios/
policy_json: "../policies/policy.json"  # Up one level, then policies/
vars_file: "../vars/common.yml"         # Up one level, then vars/
```

## Glob Patterns

`scp_paths` supports glob patterns for loading multiple files:

```yaml
scp_paths:
  - "../scp/*.json"           # All JSON files in scp/
  - "../scp/production-*.json" # Only files matching pattern
  - "../scp/deny-s3.json"     # Single file (no glob)
```

**Glob expansion:**
- `*` - Match any characters
- `**` - Recursive directory match
- `?` - Match single character
- `[abc]` - Match character set

## Complete Example

Comprehensive scenario using all features:

```yaml
# Scenario inheritance
extends: "../base/s3-base.yml"

# Template variables
vars_file: "../vars/production.yml"
vars:
  environment: "prod"  # Override

# Identity policy (templated)
policy_template: "../policies/s3-policy.json.tpl"

# Resource policy (templated)
resource_policy_template: "../policies/bucket-policy.json.tpl"

# Cross-account simulation
caller_arn: "arn:aws:iam::{{.source_account}}:user/alice"
resource_owner: "arn:aws:iam::{{.target_account}}:root"

# Organizational policies
scp_paths:
  - "../scp/organization-*.json"

# Test cases
tests:
  - name: "Alice reads from {{.environment}} bucket with MFA"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/data.txt"
    context:
      - ContextKeyName: "aws:MultiFactorAuthPresent"
        ContextKeyType: "boolean"
        ContextKeyValues: ["true"]
      - ContextKeyName: "aws:PrincipalTag/Department"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.department}}"]
    expect: "allowed"

  - name: "Bob denied (override caller)"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/data.txt"
    caller_arn: "arn:aws:iam::{{.source_account}}:user/bob"
    expect: "implicitDeny"
```

## Validation Rules

### Mutual Exclusivity

- `policy_template` XOR `policy_json` (one required)
- `resource_policy_template` XOR `resource_policy_json` (both optional)
- Legacy format (`actions`, `expect`) XOR Collection format (`tests`)

### Required Combinations

- If using `resource_policy_*`, must provide `caller_arn`
- If using `extends`, parent file must exist and be valid YAML
- Each test must have `action` and `expect`

## Next Steps

- **[See Examples](https://github.com/reaandrew/politest/tree/main/test/scenarios)** - 18 working scenarios
- **[Troubleshooting](Troubleshooting)** - Common errors and solutions
- **[Advanced Patterns](Advanced-Patterns)** - Complex use cases

---

**Full schema validation:** See [`main.go` Scenario struct](https://github.com/reaandrew/politest/blob/main/main.go) →
