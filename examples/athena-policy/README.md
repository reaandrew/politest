# ACME EC2 Instance Profile Athena Policy Example

This example demonstrates how to migrate from a bash-based IAM policy testing framework to politest's declarative YAML format.

## What This Tests

An IAM policy for EC2 instance profiles that:
- **Allows** Athena query operations via VPC endpoints
- **Allows** S3 access for Athena query results (with encryption requirements)
- **Denies** access to the `primary` workgroup
- **Denies** cross-environment access (prod vs non-prod)
- **Denies** S3 bucket management operations
- **Denies** S3 object ACL modifications
- **Requires** all S3 access to go through Athena (CalledVia condition)
- **Requires** KMS encryption for S3 PutObject operations

## Before: Legacy Bash Approach

The original test framework required:
- 40+ lines of bash script per policy
- Manual environment variable substitution with `envsubst` and `sed`
- JSON manipulation with `jq` to merge SCPs
- Complex HEREDOC strings for context conditions
- Separate test case JSON files
- Manual AWS CLI invocation for each test

Example:
```bash
#!/usr/bin/env bash
SCP_DIR="/path/to/scps"
POLICY_FILE_PATH="/path/to/policy.json"
ACCOUNT_ID=924279027186

# Manual substitutions
cat <<EOF > /tmp/replacements.sed.txt
s/123456789123/${ACCOUNT_ID}/g
s/<ACCOUNT_NO>/${ACCOUNT_ID}/g
EOF

envsubst < "$POLICY_FILE_PATH" | sed -f /tmp/replacements.sed.txt | jq '...' > /tmp/policy.txt

# Manual context construction
read -r -d '' CONTEXT_DOC <<JSON
[
  {
    "ContextKeyName": "aws:RequestedRegion",
    "ContextKeyValues": ["eu-west-2"],
    "ContextKeyType": "string"
  },
  ...
]
JSON

# Manual SCP merging
ls ../ServiceControlPolicies/*.json | grep -v ControlPlane | xargs cat | jq -s 'reduce .[]...' > /tmp/scp.json

# Finally run test
aws iam simulate-custom-policy \
  --policy-input-list "$(cat /tmp/policy.txt)" \
  --action-names athena:BatchGetNamedQuery \
  --permissions-boundary-policy-input-list "$(cat /tmp/scp.json)" \
  --resource-arns "arn:aws:athena:eu-west-2:$ACCOUNT_ID:workgroup/primary" \
  --context-entries "$CONTEXT_DOC" \
  --output json
```

## After: politest Approach

With politest, the entire test suite is now:

**vars.yml** - Environment variables
```yaml
dummy_account_id: "123456789126"
dns_hub_dev_account_id: "123456789123"
dns_hub_live_account_id: "123456789124"
data_plane_nonprod_org_path: "o-abc123xyz/r-def456/ou-nonprod-789"
data_plane_prod_org_path: "o-abc123xyz/r-def456/ou-prod-012"
region: "eu-west-2"
```

**scenario.yml** - Test cases (excerpt)
```yaml
vars_file: "vars.yml"
policy_template: "policies/ACME_EC2InstanceProfile_AthenaPolicy.json.tpl"
scp_paths: ["scp/*.json"]

tests:
  - name: "AthenaWorkgroupActionsAllow - BatchGetNamedQuery on dev account"
    action: "athena:BatchGetNamedQuery"
    resource: "arn:aws:athena:{{.region}}:{{.dns_hub_dev_account_id}}:workgroup/test-workgroup"
    context:
      - ContextKeyName: "aws:CalledVia"
        ContextKeyValues: ["athena.amazonaws.com"]
        ContextKeyType: "stringList"
      - ContextKeyName: "aws:ResourceOrgPaths"
        ContextKeyValues: ["{{.data_plane_nonprod_org_path}}/account1"]
        ContextKeyType: "string"
      - ContextKeyName: "aws:VpceAccount"
        ContextKeyValues: ["{{.dns_hub_dev_account_id}}"]
        ContextKeyType: "string"
    expect: "allowed"

  - name: "AthenaWorkgroupActionsDeny - Access to primary workgroup"
    action: "athena:BatchGetNamedQuery"
    resource: "arn:aws:athena:*:*:workgroup/primary"
    context: [...]
    expect: "explicitDeny"
```

**Run all 30 tests:**
```bash
politest --scenario scenario.yml
```

## File Structure

```
athena-policy/
├── README.md                                      # This file
├── vars.yml                                       # Environment variables
├── scenario.yml                                   # Test scenarios (30 test cases)
├── scp/
│   └── 010-allow-all.json                        # Base SCP (permissive baseline)
└── policies/
    └── ACME_EC2InstanceProfile_AthenaPolicy.json.tpl  # Policy template
```

## Running the Tests

### Prerequisites

1. Build politest:
```bash
cd ../..
go build -o politest .
```

2. Configure AWS credentials with `iam:SimulateCustomPolicy` permission:
```bash
export AWS_PROFILE=your-profile
# or
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

### Run All Tests

```bash
cd examples/athena-policy
../../politest --scenario scenario.yml
```

Expected output:
```
Running 30 test(s)...

[1/30] AthenaActionsAllow
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/30] AthenaActionsAllow - ListTagsForResource
  ✓ PASS: allowed (matched: PolicyInputList.1)

[3/30] AthenaActionsAllow - ListWorkGroups
  ✓ PASS: allowed (matched: PolicyInputList.1)

[4/30] AthenaWorkgroupActionsAllow - BatchGetNamedQuery on dev account
  ✓ PASS: allowed (matched: PolicyInputList.1)

...

[15/30] AthenaWorkgroupActionsDeny - Access to primary workgroup
  ✓ PASS: explicitDeny (matched: PolicyInputList.1.DenyAthenaOnPrimaryWorkgroup)

...

Summary: 30 passed, 0 failed
```

### Run Specific Test Cases

To run only specific tests, create a separate scenario file:

**scenario-athena-only.yml:**
```yaml
extends: "scenario.yml"

tests:
  - name: "AthenaActionsAllow"
    action: "athena:ListEngineVersions"
    resource: "arn:aws:athena:{{.region}}:{{.dummy_account_id}}:*"
    expect: "allowed"
```

```bash
../../politest --scenario scenario-athena-only.yml
```

### Debug Mode

Save raw AWS API responses for debugging:
```bash
../../politest --scenario scenario.yml --save /tmp/athena-policy-results.json
cat /tmp/athena-policy-results.json | jq '.[] | .EvaluationResults'
```

### CI/CD Integration

Exit codes:
- `0` - All tests passed
- `1` - Configuration error or AWS API error
- `2` - Test expectations failed

Example GitHub Actions:
```yaml
- name: Test Athena Policy
  run: |
    cd examples/athena-policy
    ../../politest --scenario scenario.yml
```

## Test Case Breakdown

### Allow Cases (11 tests)

1. **AthenaActionsAllow** (3 tests) - List operations without resource restrictions
2. **AthenaWorkgroupActionsAllow** (3 tests) - Query operations on allowed workgroups with proper context
3. **S3BucketActionsAllow** (2 tests) - Bucket metadata access via Athena
4. **S3ObjectActionsAllow** (3 tests) - Object read/write operations with encryption

### Deny Cases (19 tests)

1. **AthenaWorkgroupActionsDeny** (2 tests) - Blocked access to `primary` workgroup
2. **AthenaWorkgroupNonProdQueryActionsDeny** (1 test) - Wrong VPC endpoint for environment
3. **AthenaWorkgroupProdQueryActionsDeny** (1 test) - Cross-environment access blocked
4. **S3BucketAccountBucketActionsDeny** (3 tests) - Bucket management operations blocked
5. **S3ObjectAccountObjectActionsDeny** (1 test) - Object deletion on account-prefixed buckets
6. **S3ObjectAclActionsDeny** (2 tests) - ACL modifications blocked
7. **S3ObjectCalledViaActionsDeny** (2 tests) - Missing or wrong CalledVia context
8. **S3ObjectEncryptionActionsDeny** (1 test) - PutObject without KMS encryption
9. **S3ResourceOwnerActionsDeny** (1 test) - Wrong resource account

## Key Features Demonstrated

### 1. Template Variables

Policy template uses Go templates for dynamic values:
```json
{
  "Resource": [
    "arn:aws:athena:{{.region}}:{{.dns_hub_dev_account_id}}:workgroup/*",
    "arn:aws:athena:{{.region}}:{{.dns_hub_live_account_id}}:workgroup/*"
  ]
}
```

Variables from `vars.yml` are automatically substituted.

### 2. Context Conditions

Test different IAM condition contexts:
```yaml
context:
  - ContextKeyName: "aws:CalledVia"
    ContextKeyValues: ["athena.amazonaws.com"]
    ContextKeyType: "stringList"
  - ContextKeyName: "s3:ResourceAccount"
    ContextKeyValues: ["{{.dummy_account_id}}"]
    ContextKeyType: "string"
```

### 3. SCP Integration

Automatically merges all SCP files:
```yaml
scp_paths:
  - "scp/*.json"
```

### 4. Both Allow and Deny Testing

Tests positive cases (should work) and negative cases (should be blocked):
```yaml
tests:
  - name: "Should allow valid Athena access"
    expect: "allowed"

  - name: "Should deny primary workgroup"
    expect: "explicitDeny"
```

## Customization

### Test Different Accounts

Edit `vars.yml`:
```yaml
dummy_account_id: "999888777666"  # Your test account
```

### Add More SCPs

Add files to `scp/` directory:
```bash
# Files are merged in alphabetical order
scp/
  010-allow-all.json
  020-require-mfa.json
  030-restrict-regions.json
```

### Test Additional Actions

Add more test cases to `scenario.yml`:
```yaml
tests:
  - name: "Custom test case"
    action: "athena:UpdateWorkGroup"
    resource: "arn:aws:athena:{{.region}}:{{.dummy_account_id}}:workgroup/custom"
    expect: "allowed"
```

## Benefits Over Legacy Framework

| Legacy Bash Framework | politest |
|----------------------|----------|
| 40+ lines of bash per test | 10 lines of YAML per test |
| Manual temp file cleanup | No temp files |
| Sequential execution | Reusable scenarios |
| Complex jq expressions | Simple template syntax |
| No version control friendly | Git-friendly YAML |
| No CI/CD integration | Exit codes for CI/CD |
| No test reusability | Scenario inheritance with `extends:` |
| Hard to read/maintain | Self-documenting |

## Troubleshooting

### Test fails with "AccessDenied"

Ensure your AWS credentials have the `iam:SimulateCustomPolicy` permission:
```json
{
  "Effect": "Allow",
  "Action": "iam:SimulateCustomPolicy",
  "Resource": "*"
}
```

### Variables not substituting

Check that variable names match exactly (case-sensitive):
```yaml
# vars.yml
dummy_account_id: "123456789126"

# scenario.yml - correct
resource: "arn:aws:s3:::{{.dummy_account_id}}-bucket"

# scenario.yml - WRONG (typo)
resource: "arn:aws:s3:::{{.dummy_account}}-bucket"
```

### Expected "allowed" but got "implicitDeny"

Check:
1. Policy allows the action
2. Resource ARN matches
3. All required conditions are met
4. SCPs don't block the action

Use `--save` to inspect the raw response:
```bash
../../politest --scenario scenario.yml --save /tmp/debug.json
jq '.[] | select(.EvaluationResults[].EvalDecision != "allowed")' /tmp/debug.json
```

## Next Steps

1. **Modify for your policies** - Replace the policy template with your actual IAM policy
2. **Add your SCPs** - Copy your organization's SCPs to `scp/` directory
3. **Update variables** - Set actual account IDs and organization paths
4. **Integrate with CI/CD** - Add to your GitHub Actions or GitLab CI pipeline
5. **Extend test coverage** - Add more test cases for edge cases

## Related Examples

- [Simple S3 Example](../simple-s3/) - Basic S3 policy testing
- [Cross-Account Example](../cross-account-s3/) - Resource policies and cross-account access

## References

- [politest Main Documentation](../../README.md)
- [AWS SimulateCustomPolicy API](https://docs.aws.amazon.com/IAM/latest/APIReference/API_SimulateCustomPolicy.html)
- [IAM Policy Evaluation Logic](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_evaluation-logic.html)
- [IAM Condition Context Keys](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_condition-keys.html)
