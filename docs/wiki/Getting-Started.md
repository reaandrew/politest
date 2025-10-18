# Getting Started Guide

This guide teaches you the fundamentals of IAM policy testing with politest through practical examples.

## Table of Contents

- [Core Concepts](#core-concepts)
- [Your First Test](#your-first-test)
- [Understanding Test Output](#understanding-test-output)
- [Testing Denied Actions](#testing-denied-actions)
- [Testing Multiple Actions](#testing-multiple-actions)
- [Common Patterns](#common-patterns)
- [Next Steps](#next-steps)

## Core Concepts

### How IAM Policy Testing Works

politest uses the AWS `SimulateCustomPolicy` API to evaluate policy documents **without deploying them**. This means you can:

- Test policies before applying them to users/roles
- Validate policy changes won't break existing permissions
- Ensure policies grant only intended permissions
- Catch misconfigurations early in development

### Key Components

1. **Policy Document**: The IAM policy you want to test (JSON format)
2. **Test Scenario**: YAML file defining what to test (actions, resources, expectations)
3. **Test Assertions**: Expected outcomes (`allowed`, `implicitDeny`, `explicitDeny`)

### Decision Types

- **`allowed`**: The action is permitted by the policy
- **`implicitDeny`**: The action is not explicitly allowed (default deny)
- **`explicitDeny`**: The action is explicitly denied in a policy statement

## Your First Test

Let's create a simple S3 read-only policy and test it.

### Step 1: Create Your Policy

Create `s3-readonly-policy.json`:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Sid": "S3ReadOnly",
    "Effect": "Allow",
    "Action": [
      "s3:GetObject",
      "s3:ListBucket"
    ],
    "Resource": "*"
  }]
}
```

This policy allows reading S3 objects and listing buckets, but nothing else.

### Step 2: Create Your Test Scenario

Create `test-s3-readonly.yml`:

```yaml
# Reference the policy to test
policy_json: "s3-readonly-policy.json"

# Define test cases
tests:
  - name: "Reading objects should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::my-bucket/data.txt"
    expect: "allowed"

  - name: "Listing buckets should be allowed"
    action: "s3:ListBucket"
    resource: "arn:aws:s3:::my-bucket"
    expect: "allowed"

  - name: "Writing objects should be denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::my-bucket/data.txt"
    expect: "implicitDeny"

  - name: "Deleting objects should be denied"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::my-bucket/data.txt"
    expect: "implicitDeny"
```

### Step 3: Run the Test

```bash
politest --scenario test-s3-readonly.yml
```

**Output:**

```
Running 4 test(s)...

[1/4] Reading objects should be allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/4] Listing buckets should be allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)

[3/4] Writing objects should be denied
  ✓ PASS: implicitDeny

[4/4] Deleting objects should be denied
  ✓ PASS: implicitDeny

========================================
Test Results: 4 passed, 0 failed
========================================
```

Congratulations! You've successfully tested an IAM policy.

## Understanding Test Output

Let's break down what each part of the output means:

```
[1/4] Reading objects should be allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)
```

- **`[1/4]`**: Test number (1 out of 4 total)
- **`Reading objects should be allowed`**: Your test name
- **`✓ PASS`**: Test passed
- **`allowed`**: The actual decision from AWS
- **`(matched: PolicyInputList.1)`**: Which policy statement allowed the action (statement 1 in the identity policy)

### When Tests Fail

If a test fails, you'll see:

```
[3/4] Writing objects should be denied
  ✗ FAIL: expected implicitDeny, got allowed (matched: PolicyInputList.1)
```

This means:
- You expected the action to be denied (`implicitDeny`)
- But AWS actually allowed it (`allowed`)
- The action was allowed by statement 1 in your policy

This indicates your policy is too permissive!

## Testing Denied Actions

There are two types of denies in IAM:

### Implicit Deny (Not Explicitly Allowed)

When an action isn't mentioned in any `Allow` statement:

```yaml
tests:
  - name: "EC2 actions not in policy should be implicitly denied"
    action: "ec2:RunInstances"
    resource: "*"
    expect: "implicitDeny"
```

### Explicit Deny (Explicitly Blocked)

When an action is in a `Deny` statement:

**Policy with explicit deny:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    },
    {
      "Effect": "Deny",
      "Action": "s3:DeleteBucket",
      "Resource": "*"
    }
  ]
}
```

**Test:**

```yaml
tests:
  - name: "S3 read allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"

  - name: "Delete bucket explicitly denied"
    action: "s3:DeleteBucket"
    resource: "arn:aws:s3:::bucket"
    expect: "explicitDeny"
```

**Key difference**: `explicitDeny` always wins, even if there's an `Allow` statement!

## Testing Multiple Actions

### Legacy Format (Quick Testing)

For testing the same resource with multiple actions:

```yaml
policy_json: "policy.json"

# Actions to test
actions:
  - "s3:GetObject"
  - "s3:PutObject"
  - "s3:DeleteObject"

# Resources to test against
resources:
  - "arn:aws:s3:::my-bucket/*"

# Expected outcomes for each action
expect:
  "s3:GetObject": "allowed"
  "s3:PutObject": "allowed"
  "s3:DeleteObject": "implicitDeny"
```

This is equivalent to running 3 separate tests (one per action).

### Collection Format (Recommended)

For more complex scenarios with different resources per test:

```yaml
policy_json: "policy.json"

tests:
  - name: "Read from prod bucket"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::prod-bucket/*"
    expect: "allowed"

  - name: "Write to dev bucket"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::dev-bucket/*"
    expect: "allowed"

  - name: "Delete from prod bucket (should fail)"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::prod-bucket/*"
    expect: "implicitDeny"
```

**When to use each:**
- **Legacy format**: Simple tests, same resource, multiple actions
- **Collection format**: Complex tests, descriptive names, different resources

See [Scenario Formats](Scenario-Formats) for detailed comparison.

## Common Patterns

### Pattern 1: Testing Resource Restrictions

**Policy limiting actions to specific buckets:**

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "s3:*",
    "Resource": [
      "arn:aws:s3:::dev-*",
      "arn:aws:s3:::dev-*/*"
    ]
  }]
}
```

**Test:**

```yaml
policy_json: "dev-only-policy.json"

tests:
  - name: "Access to dev bucket allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::dev-data/file.txt"
    expect: "allowed"

  - name: "Access to prod bucket denied"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::prod-data/file.txt"
    expect: "implicitDeny"
```

### Pattern 2: Testing Action Restrictions

**Policy allowing only read operations:**

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "s3:GetObject",
      "s3:GetObjectVersion",
      "s3:ListBucket"
    ],
    "Resource": "*"
  }]
}
```

**Test:**

```yaml
policy_json: "read-only-policy.json"

tests:
  - name: "Read operations allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::any-bucket/*"
    expect: "allowed"

  - name: "Write operations denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::any-bucket/*"
    expect: "implicitDeny"

  - name: "Delete operations denied"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::any-bucket/*"
    expect: "implicitDeny"
```

### Pattern 3: Testing Wildcard Policies

**Policy with wildcards:**

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "s3:*",
    "Resource": "*"
  }]
}
```

**Test to verify wildcards work as expected:**

```yaml
policy_json: "s3-admin-policy.json"

tests:
  - name: "All S3 actions should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket1/*"
    expect: "allowed"

  - action: "s3:PutObject"
    resource: "arn:aws:s3:::bucket2/*"
    expect: "allowed"

  - action: "s3:DeleteBucket"
    resource: "arn:aws:s3:::bucket3"
    expect: "allowed"

  - name: "Non-S3 actions still denied"
    action: "ec2:RunInstances"
    resource: "*"
    expect: "implicitDeny"
```

## Debugging Failed Tests

### Use --save to Inspect AWS Response

```bash
politest --scenario test.yml --save /tmp/response.json
```

This saves the full AWS API response, including:
- Which policy statements matched
- Evaluation details for each test
- Missing dependencies

**Example response:**

```json
{
  "EvaluationResults": [{
    "EvalActionName": "s3:GetObject",
    "EvalResourceName": "arn:aws:s3:::bucket/*",
    "EvalDecision": "allowed",
    "MatchedStatements": [{
      "SourcePolicyId": "PolicyInputList.1",
      "SourcePolicyType": "IAM Policy",
      "StartPosition": { "Line": 4, "Column": 5 },
      "EndPosition": { "Line": 9, "Column": 6 }
    }]
  }]
}
```

### Use --no-assert to See All Results

Skip failing on mismatches to see all test outcomes:

```bash
politest --scenario test.yml --no-assert
```

Useful for:
- Exploring what a policy actually allows
- Updating test expectations
- Debugging complex policy interactions

## Next Steps

Now that you understand the basics:

1. **[Learn Scenario Formats](Scenario-Formats)** - Understand legacy vs collection format
2. **[Use Template Variables](Template-Variables)** - Make tests environment-agnostic
3. **[Test Resource Policies](Resource-Policies-and-Cross-Account)** - Cross-account S3, KMS, etc.
4. **[Add SCPs and RCPs](SCPs-and-RCPs)** - Test organizational policies
5. **[Test Conditions](Context-Conditions)** - IP restrictions, MFA, tags, etc.

---

**Need more examples?** Check out the [test/scenarios/](https://github.com/reaandrew/politest/tree/main/test/scenarios) directory for 18+ real examples →
