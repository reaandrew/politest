# politest Wiki

Welcome to the **politest** documentation! This comprehensive guide will help you master AWS IAM policy testing.

## What is politest?

**politest** is a powerful, single-binary CLI tool for testing AWS IAM policies using the official AWS `SimulateCustomPolicy` API. It helps development teams validate IAM policies before deployment, preventing security misconfigurations and access control issues.

### Key Features

- **Policy Simulation**: Test identity policies, resource policies, SCPs, and RCPs without deploying to AWS
- **Template Variables**: Use Go templates for environment-specific testing
- **Scenario Inheritance**: Reuse base scenarios with `extends:` to avoid duplication
- **Cross-Account Testing**: Simulate cross-account access patterns with `caller_arn` and `resource_owner`
- **Context Conditions**: Test IAM condition keys (IP addresses, MFA, tags, etc.)
- **Dual Formats**: Legacy format for quick tests, collection format for comprehensive test suites
- **CI/CD Integration**: Perfect for pre-deployment validation and automated testing
- **Zero Dependencies**: Single Go binary, no external tools required

## Quick Start

### Installation

```bash
# Download latest release
wget https://github.com/reaandrew/politest/releases/latest/download/politest-linux-amd64
chmod +x politest-linux-amd64
sudo mv politest-linux-amd64 /usr/local/bin/politest

# Or build from source
git clone https://github.com/reaandrew/politest.git
cd politest
go build -o politest .
```

### Your First Test

Create a simple test scenario (`test.yml`):

```yaml
# Identity policy to test
policy_json: "policy.json"

tests:
  - name: "S3 read access should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::my-bucket/data.txt"
    expect: "allowed"

  - name: "S3 write access should be denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::my-bucket/data.txt"
    expect: "implicitDeny"
```

Create your policy file (`policy.json`):

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["s3:GetObject", "s3:ListBucket"],
    "Resource": "*"
  }]
}
```

Run the test:

```bash
# Requires AWS credentials with iam:SimulateCustomPolicy permission
politest --scenario test.yml
```

Output:

```
Running 2 test(s)...

[1/2] S3 read access should be allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/2] S3 write access should be denied
  ✓ PASS: implicitDeny

========================================
Test Results: 2 passed, 0 failed
========================================
```

## Documentation Structure

### Getting Started
- **[Installation and Setup](Installation-and-Setup)** - Download, install, and configure politest
- **[Getting Started Guide](Getting-Started)** - Your first test and basic concepts
- **[Scenario Formats](Scenario-Formats)** - Legacy vs Collection formats

### Core Features
- **[Template Variables](Template-Variables)** - Using Go templates for dynamic policies
- **[Scenario Inheritance](Scenario-Inheritance)** - Reusing scenarios with `extends:`
- **[Resource Policies and Cross-Account](Resource-Policies-and-Cross-Account)** - Testing resource-based policies
- **[SCPs and RCPs](SCPs-and-RCPs)** - Service Control Policies and Resource Control Policies
- **[Context Conditions](Context-Conditions)** - Testing IAM condition keys

### Advanced Usage
- **[Advanced Patterns](Advanced-Patterns)** - Complex scenarios and best practices
- **[CI/CD Integration](CI-CD-Integration)** - Automating policy tests in pipelines
- **[Troubleshooting](Troubleshooting)** - Common issues and solutions
- **[API Reference](API-Reference)** - Complete YAML schema reference

## Real-World Examples

### Example 1: S3 Bucket Policy Testing

Test cross-account access to an S3 bucket:

```yaml
# Alice's identity policy (allows all S3)
policy_json: "policies/user-alice-identity.json"

# Bucket's resource policy (allows read, denies write)
resource_policy_json: "policies/s3-bucket-policy.json"

# Simulate as Alice from account 111111111111
caller_arn: "arn:aws:iam::111111111111:user/alice"

# Bucket owned by account 222222222222
resource_owner: "arn:aws:iam::222222222222:root"

tests:
  - name: "Cross-account read allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    expect: "allowed"

  - name: "Cross-account write denied by resource policy"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::shared-bucket/data.txt"
    expect: "explicitDeny"
```

### Example 2: Organizational SCPs

Test that SCPs properly restrict permissions:

```yaml
# Developer policy (allows everything)
policy_json: "policies/developer-policy.json"

# Organization SCP (denies production access)
scp_paths:
  - "scp/deny-production-*.json"

tests:
  - name: "Dev environment access allowed"
    action: "ec2:TerminateInstances"
    resource: "arn:aws:ec2:us-east-1:123456789012:instance/i-dev-*"
    expect: "allowed"

  - name: "Production access denied by SCP"
    action: "ec2:TerminateInstances"
    resource: "arn:aws:ec2:us-east-1:123456789012:instance/i-prod-*"
    expect: "explicitDeny"
```

### Example 3: Conditional Access with MFA

Test MFA requirements:

```yaml
policy_json: "policies/require-mfa-for-delete.json"

tests:
  - name: "Delete allowed with MFA"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::secure-bucket/file.txt"
    context:
      - ContextKeyName: "aws:MultiFactorAuthPresent"
        ContextKeyType: "boolean"
        ContextKeyValues: ["true"]
    expect: "allowed"

  - name: "Delete denied without MFA"
    action: "s3:DeleteObject"
    resource: "arn:aws:s3:::secure-bucket/file.txt"
    context:
      - ContextKeyName: "aws:MultiFactorAuthPresent"
        ContextKeyType: "boolean"
        ContextKeyValues: ["false"]
    expect: "implicitDeny"
```

## Why Use politest?

### Traditional IAM Testing Problems

1. **Manual Testing is Slow**: Deploying policies to test them wastes time
2. **Production Errors are Costly**: Wrong permissions cause outages or security breaches
3. **Complex Policies are Hard to Validate**: Multi-policy interactions are difficult to reason about
4. **No Test History**: Manual checks leave no audit trail

### How politest Helps

✅ **Test Before Deploy**: Catch errors before they reach AWS
✅ **Fast Feedback**: Run hundreds of tests in seconds
✅ **Version Control**: Store tests alongside infrastructure code
✅ **CI/CD Ready**: Automated validation on every commit
✅ **Comprehensive Coverage**: Test all policy types (identity, resource, SCP, RCP)
✅ **Real AWS API**: Uses official AWS simulation, not approximations

## Community and Support

- **GitHub Issues**: [Report bugs or request features](https://github.com/reaandrew/politest/issues)
- **Discussions**: [Ask questions and share patterns](https://github.com/reaandrew/politest/discussions)
- **Examples**: See [test/scenarios/](https://github.com/reaandrew/politest/tree/main/test/scenarios) for 18+ working examples

## Next Steps

1. **[Install politest](Installation-and-Setup)** and set up AWS credentials
2. **[Follow the Getting Started guide](Getting-Started)** to create your first test
3. **[Explore Scenario Formats](Scenario-Formats)** to understand testing patterns
4. **[Learn Template Variables](Template-Variables)** for environment-specific testing
5. **[Review Real Examples](https://github.com/reaandrew/politest/tree/main/test/scenarios)** in the repository

---

**Ready to start?** Head to [Installation and Setup](Installation-and-Setup) →
