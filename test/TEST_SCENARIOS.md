# Integration Test Scenarios

This directory contains integration tests for `politest` that run against the real AWS `SimulateCustomPolicy` API.

## Test Structure

### Scenarios Directory (`scenarios/`)

Contains YAML test scenario files numbered sequentially:

#### Core Functionality Tests (01-07)
- **01-policy-allows-no-boundaries.yml** - Basic policy evaluation without SCPs/RCPs
- **02-policy-allows-scp-denies.yml** - SCP denies override identity policy allows
- **03-policy-allows-scp-allows-different-service.yml** - SCP allows different service
- **04-policy-denies-even-without-boundaries.yml** - Explicit deny in identity policy
- **05-policy-allows-rcp-denies.yml** - RCP denies override identity policy allows
- **06-policy-denies-scp-allows.yml** - Explicit deny overrides SCP allows
- **07-multiple-scps-all-must-allow.yml** - Multiple SCPs combined

#### Collection Format Tests (08-09)
- **08-collection-format-s3.yml** - Named test cases with expectations
- **09-collection-format-scp-denies.yml** - Collection format with SCP

#### Context Conditions Tests (10-11)
- **10-context-conditions.yml** - IP, MFA, tags conditions
- **11-context-stringlist.yml** - StringList context type

#### Resource Policy Tests (12-13)
- **12-resource-policy-cross-account.yml** - Cross-account access via resource policies
- **13-test-level-overrides.yml** - Test-level policy/caller overrides

#### Variable Templating Tests (14-16)
- **14-base-with-variables.yml** - Template variables from vars_file
- **15-extends-add-scp.yml** - Scenario inheritance with SCP addition
- **16-extends-override-vars.yml** - Variable override via extends

#### Advanced Tests (17-18)
- **17-collection-format-rcp.yml** - RCP support
- **18-comprehensive-all-features.yml** - All features combined

#### Non-IAM Field Stripping Tests (19-21)
- **19-strip-non-iam-fields-default.yml** - Default behavior: silently strip non-IAM fields from identity policy
- **20-strip-non-iam-with-scp.yml** - Strip non-IAM fields from both identity policy and SCP
- **21-strip-resource-policy-metadata.yml** - Strip non-IAM fields from resource policies

#### Failure Scenarios (fail-*)
These scenarios are **expected to fail** when run with specific flags. All failure scenarios are prefixed with `fail-`.

- **fail-strict-policy-identity.yml** - Fails with --strict-policy: identity policy contains non-IAM fields
- **fail-strict-policy-resource.yml** - Fails with --strict-policy: resource policy contains non-IAM fields
- **fail-strict-policy-scp.yml** - Fails with --strict-policy: SCP contains non-IAM fields

### Policy Files (`policies/`, `scp/`, `rcp/`)

Test policy documents used by scenarios:

#### Identity Policies
- `allow-s3.json` - Basic S3 permissions
- `allow-s3-with-conditions.json` - S3 with condition keys
- `allow-s3-with-metadata.json` - **NEW**: S3 policy with non-IAM fields (Description, Author, Tags, etc.)
- `allow-dynamodb.json` - DynamoDB permissions
- `allow-ec2.json` - EC2 permissions
- `deny-s3-delete.json` - Explicit deny for S3 delete
- `ec2-with-conditions.json` - EC2 with conditions
- `user-alice-identity.json` - User identity for cross-account tests

#### Resource Policies
- `s3-bucket-policy-cross-account.json` - Cross-account bucket policy
- `s3-bucket-policy-templated.json.tpl` - Template version
- `s3-resource-policy-with-metadata.json` - **NEW**: Resource policy with non-IAM metadata fields

#### SCPs
- `scp/permissive.json` - Allow-all SCP
- `scp/deny-s3-write.json` - Deny S3 write operations
- `scp/deny-s3-delete-with-metadata.json` - **NEW**: SCP with non-IAM metadata fields

#### RCPs
- `rcp/deny-dynamodb.json` - Deny DynamoDB operations

## Running Tests

### Run All Standard Integration Tests
```bash
cd test
bash run-tests.sh
```

This runs scenarios 01-21 with default behavior (strips non-IAM fields silently).

### Run Failure Scenario Tests
```bash
cd test
bash test-failures.sh
```

Runs scenarios prefixed with `fail-*` that are **expected to fail**:
- Tests `--strict-policy` flag with policies containing non-IAM fields
- Verifies proper error messages are returned
- All fail scenarios must have the `fail-` prefix

### Run All Tests (Standard + Failures)
```bash
cd test
bash run-all-tests.sh
```

Runs both standard integration tests (01-21) and failure scenarios (fail-*).

## Non-IAM Field Stripping Feature

### Valid IAM Schema Fields

**Top-level**: `Version`, `Id`, `Statement`

**Statement-level**: `Sid`, `Effect`, `Principal`, `NotPrincipal`, `Action`, `NotAction`, `Resource`, `NotResource`, `Condition`

### Default Behavior (Silent Stripping)

All non-IAM fields are automatically removed from policies before AWS API calls:

```bash
# Runs scenario with policy containing metadata fields
# Non-IAM fields are stripped automatically, tests pass
politest --scenario scenarios/19-strip-non-iam-fields-default.yml
```

### Strict Mode

Fails if policies contain non-IAM fields:

```bash
# Fails if policy has non-IAM fields like Description, Author, etc.
politest --scenario scenarios/19-strip-non-iam-fields-default.yml --strict-policy
```

### Test Coverage

The non-IAM stripping tests cover:
- ✅ Identity policies with metadata (scenario 19)
- ✅ SCPs with metadata (scenario 20)
- ✅ Resource policies with metadata (scenario 21)
- ✅ Strict mode validation via fail scenarios (fail-strict-policy-*.yml)

## Prerequisites

- AWS credentials configured (default credential chain)
- IAM permission: `iam:SimulateCustomPolicy`
- Go 1.21+ for building the binary

## CI/CD Integration

Integration tests run automatically in GitHub Actions:
- `.github/workflows/ci.yml` - integration-tests job
- Runs on every PR and push to main
- Uses OIDC for AWS authentication (no long-lived credentials)
