# politest Examples

This directory contains real-world examples demonstrating how to use politest for IAM policy testing. These examples show how to migrate from traditional bash scripts to declarative YAML-based testing.

## Available Examples

### [athena-policy](./athena-policy/)

**Complexity:** Advanced
**What it demonstrates:** Complete migration from bash-based testing framework

A comprehensive example showing:
- **30 test cases** covering both allow and deny scenarios
- **Complex context conditions** (CalledVia, ResourceOrgPaths, VpceAccount)
- **Template variables** for multi-account testing
- **SCP integration** for organizational boundaries
- **Encryption requirements** (KMS key conditions)
- **Cross-environment access control** (prod vs non-prod)

**Based on:** Real-world EC2 instance profile policy for Athena workloads with strict security requirements.

**Before:** 40+ lines of bash with manual `envsubst`, `sed`, and `jq` manipulation
**After:** Clean YAML scenario with automatic variable substitution

[View Example →](./athena-policy/)

## Quick Start

1. Build politest:
```bash
cd ..
go build -o politest .
```

2. Run an example:
```bash
cd examples/athena-policy
../../politest --scenario scenario.yml
```

## Why Use These Examples?

### Learn by Example

Each example includes:
- ✅ Complete, runnable scenarios
- ✅ Commented YAML showing best practices
- ✅ README explaining the use case
- ✅ Before/after comparison with legacy approaches

### Migration Guide

If you're currently using:
- **Bash scripts** with `aws iam simulate-custom-policy`
- **JSON test files** with manual variable substitution
- **Complex jq expressions** for SCP merging

These examples show you exactly how to convert to politest.

### Real-World Patterns

Examples are based on actual production policies, not toy examples:
- Multi-account organizations
- VPC endpoint requirements
- Encryption enforcement
- Cross-environment boundaries
- Service Control Policies

## Example Structure

Each example follows this structure:

```
example-name/
├── README.md              # Detailed explanation
├── scenario.yml           # Main test scenario
├── vars.yml               # Environment variables
├── scp/                   # Service Control Policies
│   └── *.json
└── policies/              # IAM policy templates
    └── *.json.tpl
```

## Common Patterns

### Pattern 1: Template Variables

Replace hardcoded values with variables:

**Before (bash):**
```bash
ACCOUNT_ID=123456789012
sed "s/<ACCOUNT_NO>/${ACCOUNT_ID}/g" policy.json
```

**After (politest):**
```yaml
# vars.yml
account_id: "123456789012"

# policy.json.tpl
"Resource": "arn:aws:s3:::{{.account_id}}-bucket/*"
```

### Pattern 2: SCP Merging

Automatically merge multiple SCPs:

**Before (bash):**
```bash
ls scp/*.json | xargs cat | jq -s 'reduce .[] as $item ([]; . + $item.Statement)'
```

**After (politest):**
```yaml
scp_paths:
  - "scp/*.json"
```

### Pattern 3: Context Conditions

Declare context conditions in YAML:

**Before (bash):**
```bash
read -r -d '' CONTEXT_DOC <<JSON
[
  {
    "ContextKeyName": "aws:RequestedRegion",
    "ContextKeyValues": ["eu-west-2"],
    "ContextKeyType": "string"
  }
]
JSON
```

**After (politest):**
```yaml
context:
  - ContextKeyName: "aws:RequestedRegion"
    ContextKeyValues: ["eu-west-2"]
    ContextKeyType: "string"
```

### Pattern 4: Multiple Test Cases

Use test collection format:

**Before (bash):**
```bash
# Need separate script invocation for each test
./test-action1.sh
./test-action2.sh
./test-action3.sh
```

**After (politest):**
```yaml
tests:
  - name: "Test 1"
    action: "s3:GetObject"
    expect: "allowed"

  - name: "Test 2"
    action: "s3:PutObject"
    expect: "allowed"

  - name: "Test 3"
    action: "s3:DeleteBucket"
    expect: "explicitDeny"
```

## Migration Checklist

Migrating from bash to politest:

- [ ] Extract policy file paths → `policy_template:` or `policy_json:`
- [ ] Extract environment variables → `vars.yml` or `vars:`
- [ ] Convert SCP file lists → `scp_paths:`
- [ ] Convert action lists → `actions:` (legacy) or `tests[].action:` (collection)
- [ ] Convert resource ARNs → `resources:` (legacy) or `tests[].resource:` (collection)
- [ ] Convert context JSON → `context:` YAML array
- [ ] Convert expected results → `expect:` map (legacy) or `tests[].expect:` (collection)
- [ ] Test with `politest --scenario scenario.yml`
- [ ] Add to CI/CD pipeline
- [ ] Delete bash scripts 🎉

## Running Examples

### Prerequisites

1. AWS credentials configured:
```bash
export AWS_PROFILE=your-profile
# or
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

2. IAM permission:
```json
{
  "Effect": "Allow",
  "Action": "iam:SimulateCustomPolicy",
  "Resource": "*"
}
```

### Run All Examples

```bash
# Build once
go build -o politest .

# Run each example
cd examples/athena-policy && ../../politest --scenario scenario.yml
```

### Integration with CI/CD

**GitHub Actions:**
```yaml
name: IAM Policy Tests
on: [push, pull_request]

jobs:
  test-policies:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'

    - name: Build politest
      run: go build -o politest .

    - name: Configure AWS Credentials
      uses: aws-actions/configure-aws-credentials@v4
      with:
        role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
        aws-region: us-east-1

    - name: Run policy tests
      run: |
        cd examples/athena-policy
        ../../politest --scenario scenario.yml
```

**GitLab CI:**
```yaml
test-iam-policies:
  image: golang:1.24
  before_script:
    - go build -o politest .
  script:
    - cd examples/athena-policy
    - ../../politest --scenario scenario.yml
  variables:
    AWS_ACCESS_KEY_ID: $AWS_ACCESS_KEY_ID
    AWS_SECRET_ACCESS_KEY: $AWS_SECRET_ACCESS_KEY
```

## Benefits Summary

| Aspect | Bash Scripts | politest |
|--------|-------------|----------|
| **Lines of code** | 40+ per test | 10 per test |
| **Readability** | Complex bash/jq | Declarative YAML |
| **Maintainability** | Hard to modify | Easy to update |
| **Version control** | Messy diffs | Clean diffs |
| **Reusability** | Copy/paste | Inheritance with `extends:` |
| **CI/CD** | Custom wrappers | Native exit codes |
| **Debugging** | Manual AWS CLI | `--save` flag |
| **Testing speed** | Sequential | Efficient |

## Tips

### 1. Start Simple

Begin with a single test case:
```yaml
policy_json: "policy.json"
actions: ["s3:GetObject"]
resources: ["arn:aws:s3:::bucket/*"]
expect:
  "s3:GetObject": "allowed"
```

Then expand to test collection format with more cases.

### 2. Use Variables for Everything

Don't hardcode values:
```yaml
# Bad
resource: "arn:aws:s3:::123456789012-bucket/*"

# Good
resource: "arn:aws:s3:::{{.account_id}}-bucket/*"
```

### 3. Test Both Success and Failure

Always test denies alongside allows:
```yaml
tests:
  - name: "Admin should be allowed"
    action: "s3:DeleteBucket"
    expect: "allowed"

  - name: "Regular user should be denied"
    action: "s3:DeleteBucket"
    expect: "explicitDeny"
```

### 4. Organize SCPs by Concern

```
scp/
  010-base-allow-all.json           # Foundation
  020-require-mfa.json              # Security
  030-restrict-regions.json         # Compliance
  040-deny-root-user.json           # Governance
```

### 5. Use Descriptive Test Names

```yaml
# Bad
- name: "Test 1"

# Good
- name: "Athena workgroup access should be allowed via dev VPC endpoint"
```

## Troubleshooting

### Issue: Variables not substituting

**Problem:** Seeing literal `{{.variable}}` in output

**Solution:** Check variable name matches exactly (case-sensitive):
```yaml
# vars.yml
my_account: "123456"  # underscore

# policy.tpl - WRONG
{{.my-account}}  # dash

# policy.tpl - CORRECT
{{.my_account}}  # underscore
```

### Issue: Tests pass locally but fail in CI

**Problem:** Different AWS credentials or regions

**Solution:** Make regions/accounts configurable:
```yaml
vars:
  region: "us-east-1"  # Default, can override in CI

# CI environment
AWS_DEFAULT_REGION=eu-west-1 politest --scenario scenario.yml
```

### Issue: Expected "allowed" but got "implicitDeny"

**Problem:** Policy conditions not met

**Solution:** Use `--save` to debug:
```bash
politest --scenario scenario.yml --save /tmp/debug.json
jq '.[] | .EvaluationResults[] | select(.EvalDecision != "allowed")' /tmp/debug.json
```

## Contributing Examples

Have a useful example? Contributions welcome!

Requirements:
1. **Complete** - Must be fully runnable
2. **Documented** - Include detailed README
3. **Real-world** - Based on actual use case
4. **Tested** - Must pass with real AWS credentials

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## Next Steps

1. **Try the examples** - Run them with your AWS credentials
2. **Modify for your needs** - Replace with your policies and SCPs
3. **Integrate with CI/CD** - Add to your pipeline
4. **Share improvements** - Submit PRs with enhancements

## Additional Resources

- [Main Documentation](../README.md)
- [Architecture Details](../CLAUDE.md)
- [Integration Tests](../test/)
- [AWS IAM Documentation](https://docs.aws.amazon.com/IAM/)

## Questions?

- Check the [main README](../README.md)
- Review [CLAUDE.md](../CLAUDE.md) for implementation details
- Open an [issue](https://github.com/reaandrew/politest/issues) on GitHub
