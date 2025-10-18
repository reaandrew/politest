# Scenario Formats

politest supports two scenario formats: **Legacy Format** and **Collection Format**. This guide explains when to use each and how they work.

## Table of Contents

- [Quick Comparison](#quick-comparison)
- [Legacy Format](#legacy-format)
- [Collection Format](#collection-format)
- [Choosing the Right Format](#choosing-the-right-format)
- [Mixing Formats](#mixing-formats)

## Quick Comparison

| Feature | Legacy Format | Collection Format |
|---------|--------------|-------------------|
| **Use Case** | Quick tests, same resource | Complex scenarios, multiple resources |
| **Test Names** | Auto-generated | Custom (optional) |
| **Resource per Test** | Shared across all actions | Individual per test |
| **Context per Test** | ❌ No | ✓ Yes |
| **Override Policies per Test** | ❌ No | ✓ Yes |
| **Readability** | Good for simple cases | Excellent for complex cases |

## Legacy Format

The legacy format is ideal for **quick testing** of multiple actions against the same resources.

### Structure

```yaml
policy_json: "path/to/policy.json"  # Or policy_template
scp_paths:                          # Optional
  - "path/to/scp.json"
actions:                             # List of actions to test
  - "s3:GetObject"
  - "s3:PutObject"
resources:                           # List of resources (tested with each action)
  - "arn:aws:s3:::bucket/*"
expect:                              # Map of action → expected decision
  "s3:GetObject": "allowed"
  "s3:PutObject": "implicitDeny"
```

### Real Example

From [`test/scenarios/01-policy-allows-no-boundaries.yml`](https://github.com/reaandrew/politest/blob/main/test/scenarios/01-policy-allows-no-boundaries.yml):

```yaml
# Test: Policy allows S3, no SCPs/RCPs to block it
# Expected: All S3 actions allowed

policy_json: "../policies/allow-s3.json"

actions:
  - "s3:GetObject"
  - "s3:PutObject"
  - "s3:ListBucket"

resources:
  - "arn:aws:s3:::test-bucket/*"

expect:
  "s3:GetObject": "allowed"
  "s3:PutObject": "allowed"
  "s3:ListBucket": "allowed"
```

**How it works:**
- Tests each action (`s3:GetObject`, `s3:PutObject`, `s3:ListBucket`)
- Against each resource (`arn:aws:s3:::test-bucket/*`)
- Compares result with expectation from `expect` map

### When to Use Legacy Format

✅ **Use when:**
- Testing multiple actions against the same resource(s)
- Quick validation during policy development
- Simple scenarios without conditions

❌ **Don't use when:**
- Each test needs different resources
- Tests need context conditions
- You want descriptive test names
- Tests need per-test policy overrides

## Collection Format

The collection format is ideal for **comprehensive test suites** with complex scenarios.

### Structure

```yaml
policy_json: "path/to/policy.json"  # Or policy_template
resource_policy_json: "..."         # Optional
caller_arn: "..."                   # Optional
resource_owner: "..."               # Optional
scp_paths:                          # Optional
  - "path/to/scp.json"

tests:                               # Array of test cases
  - name: "Descriptive test name"    # Optional
    action: "s3:GetObject"           # Required
    resource: "arn:aws:s3:::..."     # Required (or resources array)
    expect: "allowed"                # Required
    context:                         # Optional
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["10.0.0.1"]
```

### Real Example

From [`test/scenarios/08-collection-format-s3.yml`](https://github.com/reaandrew/politest/blob/main/test/scenarios/08-collection-format-s3.yml):

```yaml
# Test: S3 policy using collection format
# Demonstrates named test cases with individual expectations

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

  - name: "ListBucket on test-bucket should be allowed"
    action: "s3:ListBucket"
    resource: "arn:aws:s3:::test-bucket"
    expect: "allowed"

  - name: "DeleteBucket should be denied (not in policy)"
    action: "s3:DeleteBucket"
    resource: "arn:aws:s3:::test-bucket"
    expect: "implicitDeny"

  - name: "GetObject from different-bucket is also allowed (policy uses wildcard)"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::different-bucket/*"
    expect: "allowed"
```

### Optional Test Names

Test names are **optional**. If omitted, politest generates a name automatically:

```yaml
tests:
  # Named test
  - name: "Custom descriptive name"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"

  # Unnamed test - displays as "s3:PutObject on arn:aws:s3:::bucket/*"
  - action: "s3:PutObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
```

**Output for unnamed tests:**

```
[2/2] s3:PutObject on arn:aws:s3:::bucket/*
  ✓ PASS: allowed
```

### When to Use Collection Format

✅ **Use when:**
- Each test needs different resources
- You want descriptive, readable test names
- Tests require context conditions (IP, MFA, tags)
- Tests need per-test policy overrides (different caller ARNs, resource policies)
- Building comprehensive test suites

❌ **Don't use when:**
- Quick one-off testing
- All tests use the same resource
- Simplicity is more important than features

## Advanced Collection Format Features

### Per-Test Resource Policy Override

From [`test/scenarios/13-test-level-overrides.yml`](https://github.com/reaandrew/politest/blob/main/test/scenarios/13-test-level-overrides.yml):

```yaml
# Default identity policy
policy_json: "../policies/user-alice-identity.json"

# Default caller
caller_arn: "arn:aws:iam::111111111111:user/alice"

tests:
  - name: "Alice with default bucket policy"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    resource_policy_json: "../policies/s3-bucket-policy-cross-account.json"
    expect: "allowed"

  - name: "Bob trying same action (override caller)"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    caller_arn: "arn:aws:iam::111111111111:user/bob"  # Override
    resource_policy_json: "../policies/s3-bucket-policy-cross-account.json"
    expect: "allowed"
```

**Test-level overrides:**
- `caller_arn` - Simulate as different IAM principal
- `resource_policy_json` / `resource_policy_template` - Different resource policy
- `resource_owner` - Different account owns the resource
- `resource_handling_option` - EC2-specific scenarios

### Per-Test Context Conditions

From [`test/scenarios/10-context-conditions.yml`](https://github.com/reaandrew/politest/blob/main/test/scenarios/10-context-conditions.yml):

```yaml
policy_json: "../policies/allow-s3-with-conditions.json"

tests:
  - name: "GetObject allowed from trusted IP range"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::secure-bucket/data.txt"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["10.0.1.50"]
    expect: "allowed"

  - name: "GetObject denied from untrusted IP"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::secure-bucket/data.txt"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["192.168.1.1"]
    expect: "implicitDeny"
```

## Choosing the Right Format

### Decision Tree

```
Do you need to test multiple actions on the same resource?
  ├─ No → Use Collection Format
  └─ Yes
      │
      Do you need context conditions or per-test overrides?
        ├─ Yes → Use Collection Format
        └─ No → Use Legacy Format
```

### Examples by Use Case

#### Use Case: Quick Policy Validation

**Best choice: Legacy Format**

```yaml
policy_json: "new-policy.json"
actions:
  - "s3:GetObject"
  - "s3:PutObject"
  - "s3:DeleteObject"
resources:
  - "arn:aws:s3:::my-bucket/*"
expect:
  "s3:GetObject": "allowed"
  "s3:PutObject": "allowed"
  "s3:DeleteObject": "implicitDeny"
```

#### Use Case: Cross-Account S3 Testing

**Best choice: Collection Format**

```yaml
policy_json: "alice-identity.json"
resource_policy_json: "bucket-policy.json"
caller_arn: "arn:aws:iam::111111111111:user/alice"
resource_owner: "arn:aws:iam::222222222222:root"

tests:
  - name: "Cross-account read"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/*"
    expect: "allowed"

  - name: "Cross-account write denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::shared-bucket/*"
    expect: "explicitDeny"
```

#### Use Case: Conditional Access Testing

**Best choice: Collection Format**

```yaml
policy_json: "mfa-required-policy.json"

tests:
  - name: "Delete with MFA"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::bucket/*"
    context:
      - ContextKeyName: "aws:MultiFactorAuthPresent"
        ContextKeyType: "boolean"
        ContextKeyValues: ["true"]
    expect: "allowed"

  - name: "Delete without MFA"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::bucket/*"
    context:
      - ContextKeyName: "aws:MultiFactorAuthPresent"
        ContextKeyType: "boolean"
        ContextKeyValues: ["false"]
    expect: "implicitDeny"
```

## Mixing Formats

**You cannot mix formats in a single file.** Choose one format per scenario file.

❌ **This will not work:**

```yaml
# DON'T DO THIS
policy_json: "policy.json"

actions:  # Legacy format
  - "s3:GetObject"

tests:    # Collection format
  - name: "test"
    action: "s3:PutObject"
```

✓ **Instead, use separate files:**

```
scenarios/
  ├── quick-test.yml          # Legacy format
  └── comprehensive-test.yml  # Collection format
```

## Converting Between Formats

### Legacy → Collection

**Before (Legacy):**

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

**After (Collection):**

```yaml
policy_json: "policy.json"
tests:
  - name: "GetObject should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"

  - name: "PutObject should be denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "implicitDeny"
```

## Next Steps

- **[Learn Template Variables](Template-Variables)** - Make scenarios reusable
- **[Understand Scenario Inheritance](Scenario-Inheritance)** - Extend base scenarios
- **[Test Resource Policies](Resource-Policies-and-Cross-Account)** - Cross-account testing

---

**See all format examples:** [test/scenarios/](https://github.com/reaandrew/politest/tree/main/test/scenarios) directory →
