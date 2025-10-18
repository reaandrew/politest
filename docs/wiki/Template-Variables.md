# Template Variables

Make your IAM policy tests environment-agnostic and reusable with Go template variables.

## Table of Contents

- [Why Use Template Variables](#why-use-template-variables)
- [Basic Usage](#basic-usage)
- [Variable Sources](#variable-sources)
- [Where Variables Work](#where-variables-work)
- [Advanced Templating](#advanced-templating)
- [Real-World Examples](#real-world-examples)

## Why Use Template Variables

Hard-coding values makes tests brittle:

❌ **Without variables:**

```yaml
policy_json: "policy.json"
tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::prod-data-bucket/file.txt"
    expect: "allowed"
```

Problems:
- Can't reuse for dev/staging/prod
- Account IDs hard-coded
- Bucket names change between environments

✓ **With variables:**

```yaml
vars_file: "vars/prod.yml"
policy_template: "policy.json.tpl"
tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/file.txt"
    expect: "allowed"
```

Benefits:
- Same test works across all environments
- Change variables, not tests
- Easy to maintain

## Basic Usage

### Step 1: Create a Variable File

**`vars/common.yml`:**

```yaml
account_id: "123456789012"
bucket_name: "my-app-data"
region: "us-east-1"
environment: "production"
```

### Step 2: Use Variables in Your Scenario

**`scenario.yml`:**

```yaml
vars_file: "vars/common.yml"
policy_json: "policy.json"

tests:
  - name: "Test in {{.environment}} environment"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/data.txt"
    expect: "allowed"
```

### Step 3: Reference Variables with {{.variable_name}}

politest uses Go template syntax:
- `{{.variable_name}}` - Insert variable value
- Variables are case-sensitive
- Use dot notation for nested values

## Variable Sources

### 1. Variable Files (vars_file)

**Recommended for:**
- Shared variables across multiple scenarios
- Environment-specific configs
- Complex nested structures

**Example:**

From [`test/vars/common-vars.yml`](https://github.com/reaandrew/politest/blob/main/test/vars/common-vars.yml):

```yaml
account_id: "123456789012"
region: "us-east-1"
environment: "production"
bucket_name: "my-app-data"
department: "Engineering"
project_name: "politest-demo"
```

**Usage:**

```yaml
vars_file: "../vars/common-vars.yml"
tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/{{.project_name}}/file.txt"
    expect: "allowed"
```

### 2. Inline Variables (vars)

**Recommended for:**
- Scenario-specific values
- Overriding variable file values
- Quick tests

**Example:**

```yaml
vars:
  bucket_name: "test-bucket"
  account_id: "987654321098"

tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/*"
    expect: "allowed"
```

### 3. Combining Both

Inline variables **override** variable file values:

```yaml
vars_file: "vars/common.yml"  # bucket_name: "prod-data"

vars:
  bucket_name: "dev-data"  # Overrides to "dev-data"

tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/*"  # Uses "dev-data"
    expect: "allowed"
```

## Where Variables Work

Variables can be used in **all** YAML fields:

### In Policies

**Policy template (`policy.json.tpl`):**

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

### In Actions and Resources

```yaml
vars:
  service: "s3"
  action_suffix: "GetObject"

tests:
  - action: "{{.service}}:{{.action_suffix}}"
    resource: "arn:aws:{{.service}}:::{{.bucket_name}}/*"
    expect: "allowed"
```

### In Context Values

```yaml
vars:
  trusted_ip: "10.0.1.100"
  required_department: "Engineering"

tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    context:
      - ContextKeyName: "aws:SourceIp"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.trusted_ip}}"]
      - ContextKeyName: "aws:PrincipalTag/Department"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.required_department}}"]
    expect: "allowed"
```

### In ARNs

```yaml
vars:
  source_account: "111111111111"
  target_account: "222222222222"

caller_arn: "arn:aws:iam::{{.source_account}}:user/alice"
resource_owner: "arn:aws:iam::{{.target_account}}:root"
```

## Advanced Templating

### Nested Variables

**`vars/config.yml`:**

```yaml
accounts:
  dev: "111111111111"
  prod: "222222222222"

buckets:
  data: "my-data-bucket"
  logs: "my-logs-bucket"
```

**Note**: politest currently supports flat variables only. For nested data, flatten in your variable file:

```yaml
dev_account: "111111111111"
prod_account: "222222222222"
data_bucket: "my-data-bucket"
logs_bucket: "my-logs-bucket"
```

### Conditional Values

Use variable files per environment:

**`vars/dev.yml`:**

```yaml
environment: "dev"
bucket_name: "dev-data"
account_id: "111111111111"
```

**`vars/prod.yml`:**

```yaml
environment: "prod"
bucket_name: "prod-data"
account_id: "222222222222"
```

**Run with different vars:**

```bash
# Test dev environment
politest --scenario test.yml  # Uses vars/dev.yml

# Test prod environment
# Change vars_file in scenario or create separate scenario
```

## Real-World Examples

### Example 1: Environment-Specific Testing

From [`test/scenarios/14-base-with-variables.yml`](https://github.com/reaandrew/politest/blob/main/test/scenarios/14-base-with-variables.yml):

**Variables (`vars/common-vars.yml`):**

```yaml
account_id: "123456789012"
bucket_name: "my-app-data"
department: "Engineering"
```

**Policy template (`policies/s3-templated.json.tpl`):**

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["s3:GetObject", "s3:PutObject", "s3:ListBucket"],
    "Resource": [
      "arn:aws:s3:::{{.bucket_name}}",
      "arn:aws:s3:::{{.bucket_name}}/*"
    ],
    "Condition": {
      "StringEquals": {
        "aws:PrincipalTag/Department": "{{.department}}"
      }
    }
  }]
}
```

**Scenario:**

```yaml
vars_file: "../vars/common-vars.yml"
policy_template: "../policies/s3-templated.json.tpl"
caller_arn: "arn:aws:iam::{{.account_id}}:user/alice"

tests:
  - name: "GetObject allowed with correct department tag"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/data.txt"
    context:
      - ContextKeyName: "aws:PrincipalTag/Department"
        ContextKeyType: "string"
        ContextKeyValues: ["{{.department}}"]
    expect: "allowed"
```

**To test different environment:**

1. Create `vars/prod.yml` with production values
2. Change `vars_file: "../vars/prod.yml"`
3. Run same scenario

### Example 2: Cross-Account with Variables

From [`test/scenarios/18-comprehensive-all-features.yml`](https://github.com/reaandrew/politest/blob/main/test/scenarios/18-comprehensive-all-features.yml):

**Variables (`vars/cross-account-vars.yml`):**

```yaml
source_account: "111111111111"
target_account: "222222222222"
alice_arn: "arn:aws:iam::111111111111:user/alice"
target_account_root: "arn:aws:iam::222222222222:root"
shared_bucket: "shared-bucket"
data_prefix: "data"
```

**Resource policy template (`policies/s3-bucket-policy-templated.json.tpl`):**

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "AWS": "{{.alice_arn}}"
    },
    "Action": ["s3:GetObject", "s3:ListBucket"],
    "Resource": [
      "arn:aws:s3:::{{.shared_bucket}}",
      "arn:aws:s3:::{{.shared_bucket}}/{{.data_prefix}}/*"
    ]
  }]
}
```

**Scenario:**

```yaml
vars_file: "../vars/cross-account-vars.yml"
policy_json: "../policies/user-alice-identity.json"
resource_policy_template: "../policies/s3-bucket-policy-templated.json.tpl"

caller_arn: "{{.alice_arn}}"
resource_owner: "{{.target_account_root}}"

tests:
  - name: "Alice can read from allowed prefix"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.shared_bucket}}/{{.data_prefix}}/file.txt"
    expect: "allowed"
```

### Example 3: Multi-Region Testing

**`vars/us-east-1.yml`:**

```yaml
region: "us-east-1"
bucket_name: "us-east-1-data"
vpc_id: "vpc-12345"
```

**`vars/eu-west-1.yml`:**

```yaml
region: "eu-west-1"
bucket_name: "eu-west-1-data"
vpc_id: "vpc-67890"
```

**Scenario:**

```yaml
vars_file: "vars/us-east-1.yml"  # Or eu-west-1.yml

tests:
  - name: "Access to {{.region}} bucket"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::{{.bucket_name}}/*"
    expect: "allowed"
```

## Best Practices

### 1. Use Descriptive Variable Names

✓ **Good:**

```yaml
prod_bucket_name: "production-data"
dev_account_id: "111111111111"
engineering_department: "Engineering"
```

❌ **Bad:**

```yaml
b: "production-data"
a: "111111111111"
d: "Engineering"
```

### 2. Organize Variables by Environment

```
vars/
  ├── common.yml        # Shared across all environments
  ├── dev.yml          # Dev-specific
  ├── staging.yml      # Staging-specific
  └── prod.yml         # Prod-specific
```

### 3. Document Your Variables

```yaml
# AWS Account Configuration
account_id: "123456789012"  # Production AWS account
region: "us-east-1"          # Primary region

# S3 Configuration
bucket_name: "my-app-data"   # Main data bucket
backup_bucket: "my-backups"  # Backup bucket

# Access Control
department: "Engineering"    # Required department tag
```

### 4. Validate Variables Before Testing

Create a simple scenario to check variables render correctly:

```yaml
vars_file: "vars/prod.yml"

tests:
  - name: "Validate variable rendering"
    action: "s3:ListBucket"
    resource: "arn:aws:s3:::{{.bucket_name}}"
    expect: "allowed"
```

Run with `--save` to see rendered values:

```bash
politest --scenario validate.yml --save /tmp/output.json
cat /tmp/output.json | jq '.EvaluationResults[0].EvalResourceName'
# Should show: "arn:aws:s3:::my-app-data"
```

## Troubleshooting

### Error: "template: <unknown>:1: undefined variable"

**Problem**: Referenced variable doesn't exist in vars file or inline vars.

**Solution**: Check variable name spelling and ensure it's defined:

```yaml
vars_file: "vars/common.yml"

# Or add inline
vars:
  missing_variable: "value"
```

### Variables Not Rendering

**Problem**: Variables show as `{{.variable}}` in output.

**Solution**: Ensure you're using a template file (`.json.tpl`, not `.json`) for policies:

```yaml
policy_template: "policy.json.tpl"  # ✓ Correct
policy_json: "policy.json"          # ✗ Won't render variables in policy
```

Note: Variables in scenario fields (actions, resources, context) always render, regardless of policy format.

## Next Steps

- **[Learn Scenario Inheritance](Scenario-Inheritance)** - Combine variables with extends
- **[See Template Examples](https://github.com/reaandrew/politest/tree/main/test/scenarios)** - Scenarios 14-18
- **[Advanced Patterns](Advanced-Patterns)** - Complex variable usage

---

**See working examples:** [`test/scenarios/14-base-with-variables.yml`](https://github.com/reaandrew/politest/blob/main/test/scenarios/14-base-with-variables.yml) →
