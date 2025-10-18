# Integration Tests

This directory contains integration tests that verify `politest` works correctly with the actual AWS IAM SimulateCustomPolicy API.

## Test Scenarios

The tests verify various policy evaluation scenarios:

1. **01-policy-allows-no-boundaries.yml** - Policy grants access, no SCPs/RCPs blocking
2. **02-policy-allows-scp-denies.yml** - Policy allows but SCP denies (tests SCP enforcement)
3. **03-policy-allows-scp-allows-different-service.yml** - SCP denies unrelated service (no effect)
4. **04-policy-denies-even-without-boundaries.yml** - Explicit deny in policy
5. **05-policy-allows-rcp-denies.yml** - Policy allows but RCP denies (tests RCP enforcement)
6. **06-policy-denies-scp-allows.yml** - Policy doesn't grant, SCP is permissive (still denied)
7. **07-multiple-scps-all-must-allow.yml** - Multiple SCPs, any deny blocks access

## Directory Structure

```
test/
├── scenarios/        # Test scenario YAML files
├── policies/         # IAM policy JSON files
├── scp/             # Service Control Policies
├── rcp/             # Resource Control Policies
├── run-tests.sh     # Test runner script
└── README.md        # This file
```

## Running Tests Locally

### Prerequisites

1. **AWS Credentials**: Configure AWS credentials with access to IAM SimulateCustomPolicy
2. **IAM Permission**: Your IAM principal needs `iam:SimulateCustomPolicy` permission
3. **Build politest**: Run `go build -o politest` from the project root

### Running All Tests

```bash
./test/run-tests.sh
```

### Running Individual Tests

```bash
./politest --scenario test/scenarios/01-policy-allows-no-boundaries.yml
```

## AWS Setup for CI/CD

The integration tests run in GitHub Actions on pushes to `main` branch. You need to configure:

### 1. Create IAM Role for GitHub Actions

Create an IAM role with the following trust policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::YOUR_ACCOUNT_ID:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:YOUR_GITHUB_ORG/politest:*"
        }
      }
    }
  ]
}
```

### 2. Attach IAM Policy to Role

Attach this minimal policy to the role:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowPolicySimulation",
      "Effect": "Allow",
      "Action": "iam:SimulateCustomPolicy",
      "Resource": "*"
    }
  ]
}
```

### 3. Set Up GitHub OIDC Provider

If not already configured, create an OIDC provider in IAM:

- Provider URL: `https://token.actions.githubusercontent.com`
- Audience: `sts.amazonaws.com`

### 4. Configure GitHub Secrets

Add this secret to your GitHub repository:

- `AWS_ROLE_ARN`: The ARN of the IAM role created above (e.g., `arn:aws:iam::123456789012:role/GitHubActionsRole`)

## Understanding the Tests

### Service Control Policies (SCPs)

SCPs are permissions boundaries applied at the AWS Organization level. Even if a policy grants access, an SCP can deny it.

- **permissive.json**: Allows all actions (doesn't restrict)
- **deny-s3.json**: Denies all S3 actions
- **deny-ec2.json**: Denies all EC2 actions

### Resource Control Policies (RCPs)

RCPs are similar to SCPs but applied at the resource level. In our tests, we use the same `scp_paths` field to pass them as permissions boundaries.

- **permissive.json**: Allows all actions
- **deny-dynamodb.json**: Denies all DynamoDB actions

### Expected Decisions

IAM policy evaluation can result in:

- **allowed**: Policy grants access, no boundaries deny it
- **implicitDeny**: Policy doesn't grant access, or boundaries deny it
- **explicitDeny**: Policy has an explicit Deny statement

## Troubleshooting

### Tests fail with "AWS credentials not configured"

- Ensure `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are set
- Or configure `~/.aws/credentials`
- Or ensure GitHub Actions has the correct role configured

### Tests fail with "AccessDenied" or "UnauthorizedOperation"

- Verify your IAM principal has `iam:SimulateCustomPolicy` permission
- Check the role's trust policy allows your GitHub repository

### Tests fail with unexpected decisions

- Review the test scenario's `expect:` block
- Run with `--no-assert` to see actual decisions: `./politest --scenario test/scenarios/01-policy-allows-no-boundaries.yml --no-assert`
- Use `--save /tmp/response.json` to inspect raw AWS response

## CI/CD Integration

The GitHub Actions workflow (`.github/workflows/ci.yml`) includes an `integration-tests` job that:

1. Only runs on pushes to `main` branch
2. Uses AWS OIDC to assume the configured IAM role
3. Runs all test scenarios via `./test/run-tests.sh`
4. Blocks release if any tests fail

This ensures policy simulation logic is tested against real AWS behavior before releasing.
