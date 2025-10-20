# politest Wiki

## Table of Contents

- [Getting Started](#getting-started)
- [Features](#features)
- [Understanding What politest Tests](#️-understanding-what-politest-tests)
- [Scenario Configuration](#scenario-configuration)
- [Variable Formats](#variable-formats)
- [Policy Fields](#policy-fields)
- [Test Formats](#test-formats)
- [Actions and Resources](#actions-and-resources)
- [Context Conditions](#context-conditions)
- [SCP/RCP Merging](#scprcp-merging)
- [Inheritance](#inheritance)
- [Resource Policies](#resource-policies)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Getting Started

### Installation

```bash
# Clone the repository
git clone https://github.com/reaandrew/politest.git
cd politest

# Build the binary
go build -o politest .

# Or run directly
go run . --scenario path/to/scenario.yml
```

### Quick Start

1. Create a scenario YAML file
2. Define your policy (template or JSON)
3. Specify actions and resources to test
4. Set expectations
5. Run politest

```bash
politest --scenario scenarios/my-test.yml
```

---

## Features

- **YAML-based scenarios**
  - Inheritance via `extends:`

- **Multiple variable formats**
  - `{{.VAR}}`, `${VAR}`, `$VAR`, `<VAR>` syntax support

- **Policy templates**
  - Use `policy_template` for policies with variables
  - Or `policy_json` for pre-rendered JSON policies

- **Flexible test formats**
  - Legacy format: `actions` + `resources` arrays
  - Collection format: `tests` array with named test cases

- **SCP/RCP merging**
  - From multiple files/globs into permissions boundaries

- **AWS IAM SimulateCustomPolicy integration**
  - Test policies before deployment

- **Expectation assertions**
  - For CI/CD integration

- **Clean table output**
  - With optional raw JSON export

---

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

---

## Scenario Configuration

### Required Fields

Every scenario must have:

**1. A Policy** - Choose one:

- `policy_template: "path/to/policy.json"`
  - For policies with variables
  - Supports `{{.VAR}}`, `${VAR}`, `$VAR`, `<VAR>` formats
  - Variables substituted before evaluation

- `policy_json: "path/to/policy.json"`
  - For policies without variables
  - Used as-is, no template rendering

**2. Tests** - Choose one format:

**Legacy Format** (backward compatible):
```yaml
actions: ["action1", "action2"]
resources: ["arn:..."]
expect:
  "action1": "allowed"
  "action2": "denied"
```

**Collection Format** (recommended):
```yaml
tests:
  - name: "Test description"
    action: "action1"
    resource: "arn:..."
    expect: "allowed"
```

### Optional Fields

- `extends: "parent.yml"`
  - Inherit from parent scenario

- `vars_file: "vars.yml"`
  - External variable file

- `vars: {key: value}`
  - Inline variables (overrides vars_file)

- `scp_paths: ["scp/*.json"]`
  - Service Control Policies to merge

- `context: [{ContextKeyName, ContextKeyValues, ContextKeyType}]`
  - IAM condition context

- `caller_arn: "arn:aws:iam::123456789012:user/alice"`
  - Principal ARN for resource policy testing

- `resource_owner: "arn:aws:iam::123456789012:root"`
  - Resource owner for cross-account testing

---

## Variable Formats

politest supports four variable formats that can be used interchangeably:

### Format 1: Go Template Syntax (Original)

```yaml
vars:
  bucket_name: "my-bucket"

# Usage
resource: "arn:aws:s3:::{{.bucket_name}}/*"
```

### Format 2: Shell Variable with Braces

```json
{
  "Resource": "arn:aws:s3:::${BUCKET_NAME}/*"
}
```

### Format 3: Shell Variable without Braces

```yaml
resource: "arn:aws:s3:::$BUCKET/*"
```

### Format 4: Angle Bracket Style

```json
{
  "Resource": "arn:aws:s3:::<BUCKET_NAME>/*"
}
```

### Mixing Formats

All formats can be used in the same file:

```yaml
vars:
  account_id: "123456789012"
  ACCOUNT_ID: "123456789012"
  bucket: "data"

policy_template: "policy.json"  # Contains ${ACCOUNT_ID}
resources:
  - "arn:aws:s3:::{{.bucket}}/*"  # Go template
  - "arn:aws:iam::$account_id:role/*"  # Shell style
  - "arn:aws:iam::<ACCOUNT_ID>:user/*"  # Angle brackets
```

**Note:** All formats are converted to Go templates internally, so variable names are case-sensitive.

---

## Policy Fields

### policy_template

Use when your policy contains variables:

```yaml
policy_template: "policies/s3-policy.json.tpl"
vars:
  bucket_name: "my-bucket"
  account_id: "123456789012"
```

Policy file (`policies/s3-policy.json.tpl`):
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::${BUCKET_NAME}/*"
  }]
}
```

### policy_json

Use for policies without variables:

```yaml
policy_json: "policies/static-policy.json"
```

### Key Differences

| Feature | policy_template | policy_json |
|---------|----------------|-------------|
| Variables | ✅ Supported | ❌ Not processed |
| Performance | Slower (rendering) | Faster (direct use) |
| Use Case | Dynamic policies | Static policies |
| File Extension | Any (`.json`, `.tpl`) | Usually `.json` |

**Important:** These fields are mutually exclusive - use one or the other, never both.

---

## Test Formats

### Legacy Format

Test multiple actions against multiple resources:

```yaml
policy_json: "policies/s3.json"

actions:
  - "s3:GetObject"
  - "s3:PutObject"
  - "s3:DeleteObject"

resources:
  - "arn:aws:s3:::bucket1/*"
  - "arn:aws:s3:::bucket2/*"

expect:
  "s3:GetObject": "allowed"
  "s3:PutObject": "allowed"
  "s3:DeleteObject": "denied"
```

**How it works:**
- Creates a cartesian product: 3 actions × 2 resources = 6 tests
- All tests use the same context and conditions
- Good for bulk testing similar scenarios

### Collection Format

Individual test cases with descriptive names:

```yaml
policy_json: "policies/s3.json"

tests:
  - name: "Read access should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket1/*"
    expect: "allowed"

  - name: "Write access should be allowed"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::bucket1/*"
    expect: "allowed"

  - name: "Delete should be denied"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::bucket1/*"
    expect: "denied"
```

**Benefits:**
- Descriptive test names for better output
- Each test can have different resources
- Each test can have different context
- Better for complex scenarios
- Easier to debug failures

### Choosing a Format

**Use Legacy Format when:**
- Testing many similar actions/resources
- All tests share the same context
- You want concise configuration

**Use Collection Format when:**
- You need descriptive test names
- Tests have different contexts
- Tests have different resources
- You need granular control

---

## Actions and Resources

### Single vs Array

Both formats support singular and plural forms:

#### In Legacy Format (Scenario Level)

```yaml
# Required - array of actions
actions:
  - "s3:GetObject"
  - "s3:PutObject"

# Optional - array of resources (defaults to "*")
resources:
  - "arn:aws:s3:::bucket/*"
```

#### In Collection Format (Test Level)

**Single action/resource:**
```yaml
tests:
  - name: "Single action test"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
```

**Multiple actions:**
```yaml
tests:
  - name: "Multiple actions test"
    actions:
      - "s3:GetObject"
      - "s3:PutObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
```

**Multiple resources:**
```yaml
tests:
  - name: "Multiple resources test"
    action: "s3:GetObject"
    resources:
      - "arn:aws:s3:::bucket1/*"
      - "arn:aws:s3:::bucket2/*"
    expect: "allowed"
```

**Both arrays:**
```yaml
tests:
  - name: "Cartesian product test"
    actions:
      - "s3:GetObject"
      - "s3:PutObject"
    resources:
      - "arn:aws:s3:::bucket1/*"
      - "arn:aws:s3:::bucket2/*"
    expect: "allowed"
    # Creates: 2 actions × 2 resources = 4 test executions
```

---

## Context Conditions

Context keys allow testing IAM conditions:

### Basic Context

```yaml
tests:
  - name: "MFA required for delete"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::secure-bucket/*"
    context:
      - ContextKeyName: "aws:MultiFactorAuthPresent"
        ContextKeyType: "boolean"
        ContextKeyValues: ["true"]
    expect: "allowed"
```

### Multiple Context Keys

```yaml
tests:
  - name: "Access with multiple conditions"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["10.0.1.0/24"]

      - ContextKeyName: "aws:PrincipalTag/Department"
        ContextKeyType: "string"
        ContextKeyValues: ["Engineering"]

      - ContextKeyName: "aws:SecureTransport"
        ContextKeyType: "boolean"
        ContextKeyValues: ["true"]
    expect: "allowed"
```

### Supported Context Types

- `string`
  - Single string value

- `stringList`
  - Multiple string values

- `numeric`
  - Single numeric value

- `numericList`
  - Multiple numeric values

- `boolean`
  - Boolean value (true/false)

- `booleanList`
  - Multiple boolean values

**Note:** IpAddress and IpAddressList types are NOT supported by the AWS SDK.

### Context with Variables

```yaml
vars:
  vpc_id: "vpc-123456"
  region: "us-east-1"

tests:
  - name: "VPC condition"
    action: "ec2:RunInstances"
    resource: "*"
    context:
      - ContextKeyName: "ec2:Vpc"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.vpc_id}}"]

      - ContextKeyName: "aws:RequestedRegion"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.region}}"]
    expect: "allowed"
```

---

## SCP/RCP Merging

Test how organizational policies (SCPs/RCPs) interact with identity policies:

### Basic SCP Usage

```yaml
policy_json: "policies/identity-policy.json"

scp_paths:
  - "scp/010-base.json"
  - "scp/020-deny-regions.json"

tests:
  - name: "Action allowed by identity but denied by SCP"
    action: "s3:DeleteBucket"
    resource: "*"
    expect: "explicitDeny"  # SCP overrides identity policy
```

### Using Globs

```yaml
scp_paths:
  - "scp/*.json"  # All JSON files in scp/ directory
  - "scp/critical/*.json"  # All files in subdirectory
```

### How Merging Works

All SCP files are merged into a single permissions boundary:

1. Read each SCP file
2. Extract all `Statement` arrays
3. Combine into one policy document
4. Apply as `PermissionsBoundaryPolicyInputList`

**Example:**

`scp/010-base.json`:
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "*",
    "Resource": "*"
  }]
}
```

`scp/020-deny-delete.json`:
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Deny",
    "Action": "*:Delete*",
    "Resource": "*"
  }]
}
```

**Result:** Both statements are combined and evaluated together.

---

## Inheritance

Use `extends:` to reuse base scenarios:

### Basic Inheritance

**Base scenario** (`scenarios/_common.yml`):
```yaml
vars:
  account_id: "123456789012"
  region: "us-east-1"

scp_paths:
  - "../scp/010-base.json"

context:
  - ContextKeyName: "aws:RequestedRegion"
    ContextKeyValues: ["{{.region}}"]
    ContextKeyType: "string"
```

**Child scenario** (`scenarios/s3-test.yml`):
```yaml
extends: "_common.yml"

policy_template: "../policies/s3-policy.json.tpl"

tests:
  - name: "GetObject test"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::my-bucket/*"
    expect: "allowed"
```

### Override Behavior

**Maps are deep-merged:**
```yaml
# Parent vars
vars:
  account_id: "111111111111"
  bucket: "parent-bucket"

# Child vars
vars:
  bucket: "child-bucket"  # Overrides
  region: "us-west-2"      # Adds new

# Result
vars:
  account_id: "111111111111"  # Inherited
  bucket: "child-bucket"       # Overridden
  region: "us-west-2"          # Added
```

**Arrays are completely replaced:**
```yaml
# Parent
actions:
  - "s3:GetObject"
  - "s3:PutObject"

# Child
actions:
  - "s3:DeleteObject"

# Result (parent actions are NOT inherited)
actions:
  - "s3:DeleteObject"
```

### Recursive Inheritance

```yaml
# scenarios/base.yml
vars:
  account_id: "123456789012"

# scenarios/base-with-scp.yml
extends: "base.yml"
scp_paths:
  - "../scp/*.json"

# scenarios/s3-test.yml
extends: "base-with-scp.yml"
policy_template: "../policies/s3.json.tpl"
```

---

## Resource Policies

Test cross-account access and resource-based policies:

### Basic Resource Policy Testing

```yaml
# Identity policy for Alice
policy_json: "policies/alice-identity.json"

# S3 bucket resource policy
resource_policy_json: "policies/bucket-policy.json"

# Simulate as Alice
caller_arn: "arn:aws:iam::111111111111:user/alice"

# Bucket owned by different account
resource_owner: "arn:aws:iam::222222222222:root"

tests:
  - name: "Cross-account read"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    expect: "allowed"
```

### Test-Level Overrides

Override policy or caller for specific tests:

```yaml
caller_arn: "arn:aws:iam::111111111111:user/alice"  # Default

tests:
  - name: "Test as Alice"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"

  - name: "Test as Bob"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    caller_arn: "arn:aws:iam::111111111111:user/bob"  # Override
    expect: "denied"

  - name: "Test with different resource policy"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::other-bucket/*"
    resource_policy_json: "../policies/other-bucket-policy.json"  # Override
    expect: "allowed"
```

### Supported Parameters

- `resource_policy_json`
  - Resource-based policy file path

- `resource_policy_template`
  - Resource-based policy template file path

- `caller_arn`
  - IAM principal ARN to simulate as

- `resource_owner`
  - Account ARN that owns the resource

- `resource_handling_option`
  - EC2 scenario type (e.g., "EC2-VPC-InstanceStore")

---

## Examples

### Example 1: Simple S3 Policy Test

```yaml
policy_json: "policies/s3-read-only.json"

tests:
  - name: "Read allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::my-bucket/*"
    expect: "allowed"

  - name: "Write denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::my-bucket/*"
    expect: "implicitDeny"
```

### Example 2: Multi-Region with Variables

```yaml
vars:
  account_id: "123456789012"
  allowed_region: "us-east-1"
  denied_region: "eu-west-1"

policy_template: "policies/region-restricted.json.tpl"

tests:
  - name: "Allowed region"
    action: "ec2:RunInstances"
    resource: "*"
    context:
      - ContextKeyName: "aws:RequestedRegion"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.allowed_region}}"]
    expect: "allowed"

  - name: "Denied region"
    action: "ec2:RunInstances"
    resource: "*"
    context:
      - ContextKeyName: "aws:RequestedRegion"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.denied_region}}"]
    expect: "explicitDeny"
```

### Example 3: SCP with Inheritance

**Base** (`_common.yml`):
```yaml
vars:
  account_id: "123456789012"

scp_paths:
  - "../scp/010-allow-all.json"
```

**Test** (`athena-test.yml`):
```yaml
extends: "_common.yml"

policy_template: "../policies/athena.json.tpl"

scp_paths:
  - "../scp/020-deny-athena-primary.json"

tests:
  - name: "Custom workgroup allowed"
    action: "athena:StartQueryExecution"
    resource: "arn:aws:athena:us-east-1:{{.account_id}}:workgroup/custom"
    expect: "allowed"

  - name: "Primary workgroup denied by SCP"
    action: "athena:StartQueryExecution"
    resource: "arn:aws:athena:us-east-1:{{.account_id}}:workgroup/primary"
    expect: "explicitDeny"
```

### Example 4: Resource Policy with Variables

```yaml
vars:
  source_account: "111111111111"
  bucket_account: "222222222222"
  bucket_name: "shared-data"

policy_template: "policies/cross-account-s3.json.tpl"
resource_policy_template: "policies/bucket-policy.json.tpl"

caller_arn: "arn:aws:iam::${source_account}:user/alice"
resource_owner: "arn:aws:iam::${bucket_account}:root"

tests:
  - name: "Cross-account read"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::${bucket_name}/*"
    expect: "allowed"

  - name: "Cross-account write denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::${bucket_name}/*"
    expect: "explicitDeny"
```

---

## Best Practices

### 1. Organize Scenarios

```
scenarios/
├── _common.yml           # Shared base configuration
├── _common-with-scp.yml  # Base + SCPs
├── s3/
│   ├── read-only.yml
│   ├── read-write.yml
│   └── cross-account.yml
└── ec2/
    ├── basic.yml
    └── restricted.yml
```

### 2. Use Collection Format for New Tests

Collection format provides better:
- Test naming
- Debugging
- Flexibility
- Readability

### 3. Variable Naming Conventions

Choose consistent variable naming:

```yaml
# Option 1: lowercase with underscores (recommended for Go templates)
vars:
  account_id: "123456789012"
  bucket_name: "my-bucket"

# Option 2: UPPERCASE for shell-style variables
vars:
  ACCOUNT_ID: "123456789012"
  BUCKET_NAME: "my-bucket"

# Or both if mixing formats
vars:
  account_id: "123456789012"   # For {{.account_id}}
  ACCOUNT_ID: "123456789012"   # For ${ACCOUNT_ID}
```

### 4. Template Variable Safety

Always quote variables in JSON:

```json
{
  "Resource": "arn:aws:s3:::{{.bucket}}/*"
}
```

**Not:** `"Resource": {{.bucket}}` (invalid JSON if variable contains quotes)

### 5. Use Globs for SCPs

```yaml
# Good - automatically includes all SCPs
scp_paths:
  - "../scp/*.json"

# Not recommended - must update for each new SCP
scp_paths:
  - "../scp/010-base.json"
  - "../scp/020-regions.json"
  - "../scp/030-services.json"
```

### 6. Meaningful Test Names

```yaml
# Good
tests:
  - name: "Admin users can delete S3 buckets with MFA"
    action: "s3:DeleteBucket"
    ...

# Not as helpful
tests:
  - name: "Test 1"
    action: "s3:DeleteBucket"
    ...
```

### 7. CI/CD Integration

```bash
# In your CI pipeline
politest --scenario scenarios/production-policies.yml

# Check exit code
if [ $? -eq 0 ]; then
  echo "All policy tests passed"
else
  echo "Policy tests failed"
  exit 1
fi
```

### 8. Debug with --save

```bash
# Save raw AWS response for debugging
politest --scenario test.yml --save /tmp/response.json

# Inspect MatchedStatements to see which policy statements applied
cat /tmp/response.json | jq '.EvaluationResults[].MatchedStatements'
```

---

## Troubleshooting

### Issue: Test fails with "implicitDeny" instead of "allowed"

**Possible causes:**
1. Policy doesn't grant the action
2. SCP denies the action
3. Resource ARN doesn't match policy Resource pattern
4. Missing or incorrect context conditions

**Debug:**
```bash
politest --scenario test.yml --save /tmp/debug.json
cat /tmp/debug.json | jq '.EvaluationResults[0]'
```

### Issue: Variables not substituted

**Check:**
1. Using `policy_template` (not `policy_json`)?
2. Variable defined in `vars` or `vars_file`?
3. Variable name case-sensitive match?
4. Correct variable syntax (`{{.VAR}}`, `${VAR}`, `$VAR`, `<VAR>`)?

**Example:**
```yaml
# Won't work - using policy_json with variables
policy_json: "policy.json"  # Policy contains {{.bucket}}
vars:
  bucket: "my-bucket"

# Fix - use policy_template
policy_template: "policy.json"
vars:
  bucket: "my-bucket"
```

### Issue: "No such file" error

**Check:**
1. Paths are relative to scenario file location
2. File exists at the specified path
3. Correct file permissions

**Example:**
```yaml
# If scenario is at: scenarios/s3-test.yml
# And policy is at: policies/s3-policy.json
# Use:
policy_json: "../policies/s3-policy.json"
```

### Issue: SCP not being applied

**Check:**
1. SCP files are valid JSON
2. SCP paths use correct glob patterns
3. Files contain `Statement` arrays
4. Using `--save` to inspect merged policy

**Debug:**
```bash
# Check SCP files
for f in scp/*.json; do
  echo "=== $f ==="
  cat $f | jq '.Statement'
done
```

### Issue: Context conditions not working as expected

**Remember:**
- AWS SimulateCustomPolicy has limitations
- Not all condition operators fully supported
- Test in real AWS environment for validation

**Alternative:**
- Use integration tests in actual AWS accounts
- politest is for catching obvious issues, not 100% accuracy

### Issue: Exit code 1 vs 2

**Exit codes:**
- `0` - Success
- `1` - Error (invalid scenario, AWS error, etc.)
- `2` - Expectation failures

**To ignore expectation failures:**
```bash
politest --scenario test.yml --no-assert
```

### Issue: Performance - tests are slow

**Optimization tips:**
1. Use `policy_json` instead of `policy_template` when possible
2. Reduce number of test combinations (actions × resources)
3. Use collection format with specific tests instead of cartesian products
4. Run tests in parallel (multiple scenario files)

---

## Additional Resources

- [README.md](README.md) - Project overview and quick start
- [CLAUDE.md](CLAUDE.md) - Developer documentation
- [Examples](examples/) - Real-world example scenarios
- [GitHub Issues](https://github.com/reaandrew/politest/issues) - Report bugs or request features
