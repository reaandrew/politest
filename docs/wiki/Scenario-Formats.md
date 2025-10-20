# Scenario Formats

politest uses a collection format for defining test scenarios. This guide explains the structure and features available.

## Table of Contents

- [Basic Structure](#basic-structure)

- [Test Cases](#test-cases)

- [Actions Array Expansion](#actions-array-expansion)

- [Optional Test Names](#optional-test-names)

- [Advanced Features](#advanced-features)

## Basic Structure

Every scenario file contains:

1. **Policy definition** (identity policy)

2. **Tests array** (required - one or more test cases)

3. **Optional configuration** (SCPs, variables, resource policies, etc.)

```yaml
# Identity policy (required - choose one)
policy_json: "path/to/policy.json"  # OR
policy_template: "path/to/policy.json.tpl"

# Optional global configuration
vars_file: "vars/common.yml"
vars:
  account_id: "123456789012"

scp_paths:
  - "../scp/*.json"

context:
  - ContextKeyName: "aws:RequestedRegion"
    ContextKeyType: "string"
    ContextKeyValues: ["us-east-1"]

# Tests (required)
tests:
  - name: "Test description"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
```

## Test Cases

Each test in the `tests` array specifies:

| Field | Required | Description |
|-------|----------|-------------|
| `name` | No | Descriptive test name (auto-generated if omitted) |
| `action` | Yes* | Single IAM action to test |
| `actions` | Yes* | Multiple IAM actions (expands to separate tests) |
| `resource` | No | Single resource ARN (defaults to `*`) |
| `resources` | No | Multiple resource ARNs |
| `context` | No | IAM condition context for this test |
| `expect` | Yes | Expected decision: `allowed`, `implicitDeny`, or `explicitDeny` |

\* Either `action` or `actions` is required (mutually exclusive)

### Basic Example

From [`test/scenarios/08-collection-format-s3.yml`](https://github.com/reaandrew/politest/blob/main/test/scenarios/08-collection-format-s3.yml):

```yaml
policy_json: "../policies/allow-s3.json"

tests:
  - name: "GetObject from test-bucket should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::test-bucket/*"
    expect: "allowed"

  - name: "PutObject to test-bucket should be allowed"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::test-bucket/*"
    expect: "allowed"

  - name: "DeleteBucket should be denied (not in policy)"
    action: "s3:DeleteBucket"
    resource: "arn:aws:s3:::test-bucket"
    expect: "implicitDeny"
```

## Actions Array Expansion

The `actions` field allows testing multiple actions with the same configuration. Each action expands into a separate test execution.

### Example

```yaml
tests:
  # This single test definition...
  - name: "Test S3 read operations"
    actions:
      - "s3:GetObject"
      - "s3:GetObjectVersion"
      - "s3:ListBucket"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"

  # ...expands to 3 separate tests:
  # [1/3] Test S3 read operations
  # [2/3] Test S3 read operations
  # [3/3] Test S3 read operations
```

### Real-World Example

From [`examples/athena-policy/scenario.yml`](https://github.com/reaandrew/politest/tree/main/examples/athena-policy):

```yaml
tests:
  - name: "AthenaWorkgroupActionsAllow"
    actions:
      - "athena:BatchGetNamedQuery"
      - "athena:BatchGetPreparedStatement"
      - "athena:BatchGetQueryExecution"
      - "athena:CreateNamedQuery"
      - "athena:StartQueryExecution"
      # ... 15 more actions
    resources:
      - "arn:aws:athena:{{.region}}:{{.account_id}}:workgroup/*"
    context:
      - ContextKeyName: "aws:CalledVia"
        ContextKeyValues: ["athena.amazonaws.com"]
        ContextKeyType: "stringList"
    expect: "allowed"
```

This expands to 20 individual tests, one per action, all using the same resources, context, and expectation.

### When to Use Actions Array

✅ **Use when:**

- Testing multiple similar actions with same resource/context/expect

- Validating a group of related permissions

- Reducing repetition in test definitions

❌ **Don't use when:**

- Actions have different expected outcomes

- Actions need different resources or context

- You want different test names for each action

## Optional Test Names

Test names are **optional**. If omitted, politest generates a name from the action and resource:

```yaml
tests:
  # Named test - uses provided name
  - name: "Custom descriptive name"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"

  # Unnamed test - auto-generates "s3:PutObject on arn:aws:s3:::bucket/*"
  - action: "s3:PutObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
```

**Output:**

```
[1/2] Custom descriptive name
  ✓ PASS: allowed

[2/2] s3:PutObject on arn:aws:s3:::bucket/*
  ✓ PASS: allowed
```

## Advanced Features

### Per-Test Context Conditions

Each test can have its own IAM condition context:

```yaml
policy_json: "allow-s3-with-conditions.json"

tests:
  - name: "GetObject allowed from trusted IP"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::secure-bucket/*"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["10.0.1.50"]
    expect: "allowed"

  - name: "GetObject denied from untrusted IP"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::secure-bucket/*"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["192.168.1.1"]
    expect: "implicitDeny"
```

### Per-Test Policy Overrides

Override resource policies, caller ARNs, and other settings per test:

```yaml
policy_json: "user-alice-identity.json"
caller_arn: "arn:aws:iam::111111111111:user/alice"  # Default

tests:
  - name: "Alice accesses shared bucket"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    resource_policy_json: "bucket-policy.json"
    expect: "allowed"

  - name: "Bob tries same action (override caller)"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    caller_arn: "arn:aws:iam::111111111111:user/bob"  # Override for this test
    resource_policy_json: "bucket-policy.json"
    expect: "implicitDeny"
```

**Available overrides per test:**

- `caller_arn` - Simulate as different IAM principal

- `resource_policy_json` / `resource_policy_template` - Different resource policy

- `resource_owner` - Different account owns the resource

- `resource_handling_option` - EC2-specific scenarios

### Multiple Resources

Test an action against multiple resources simultaneously:

```yaml
tests:
  - name: "GetObject on multiple buckets"
    action: "s3:GetObject"
    resources:
      - "arn:aws:s3:::bucket1/*"
      - "arn:aws:s3:::bucket2/*"
      - "arn:aws:s3:::bucket3/*"
    expect: "allowed"
```

## Complete Example

Comprehensive scenario using all features:

```yaml
# Template variables
vars_file: "vars/production.yml"
vars:
  environment: "prod"

# Identity policy (templated)
policy_template: "policies/s3-policy.json.tpl"

# Resource policy (templated)
resource_policy_template: "policies/bucket-policy.json.tpl"

# Cross-account simulation
caller_arn: "arn:aws:iam::{{.source_account}}:user/alice"
resource_owner: "arn:aws:iam::{{.target_account}}:root"

# Organizational policies
scp_paths:
  - "scp/organization-*.json"

# Global context (applied to all tests unless overridden)
context:
  - ContextKeyName: "aws:PrincipalTag/Department"
    ContextKeyType: "string"
    ContextKeyValues: ["{{.department}}"]

# Test cases
tests:
  - name: "Alice reads from {{.environment}} bucket with MFA"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/data.txt"
    context:
      - ContextKeyName: "aws:MultiFactorAuthPresent"
        ContextKeyType: "boolean"
        ContextKeyValues: ["true"]
    expect: "allowed"

  - name: "Test multiple read actions"
    actions:
      - "s3:GetObject"
      - "s3:GetObjectVersion"
      - "s3:GetObjectMetadata"
    resource: "arn:aws:s3:::{{.bucket_name}}/*"
    expect: "allowed"

  - name: "Bob denied (override caller)"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/data.txt"
    caller_arn: "arn:aws:iam::{{.source_account}}:user/bob"
    expect: "implicitDeny"
```

## Next Steps

- **[Learn Template Variables](Template-Variables)** - Make scenarios reusable

- **[API Reference](API-Reference)** - Complete field reference

- **[See Examples](https://github.com/reaandrew/politest/tree/main/test/scenarios)** - 18 working scenarios

---

**Explore real scenarios:** [test/scenarios/](https://github.com/reaandrew/politest/tree/main/test/scenarios) directory →
