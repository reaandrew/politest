# Demo 6: Cross-Account Resource Policy

Demonstrates how identity policies, resource policies, and SCPs interact in cross-account scenarios.

## What This Demonstrates

- **Cross-account access**: Alice (account 111111111111) accessing a bucket in account 222222222222
- **Policy intersection**: Action requires BOTH identity policy AND resource policy to allow
- **Explicit denies**: Bucket policy can deny actions even if identity policy allows
- **SCPs apply**: Organizational SCPs still restrict what Alice can do

## Run This Demo

```bash
politest --scenario scenario.yml
```

## Expected Output

```
Running 3 test(s)...

[1/3] Cross-account GetObject allowed by both policies
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/3] Cross-account PutObject denied by bucket policy
  ✓ PASS: explicitDeny (matched: ResourcePolicy.1)

[3/3] DeleteObject denied by identity policy (no permission)
  ✓ PASS: implicitDeny (matched: -)

========================================
Test Results: 3 passed, 0 failed
========================================
```

## Cross-Account Policy Evaluation

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────┐
│ Identity Policy │ ∩  │ Resource Policy  │ ∩  │   SCP   │
│  (Alice's IAM)  │    │  (Bucket Policy) │    │  (Org)  │
└─────────────────┘    └──────────────────┘    └─────────┘
```

For cross-account access, ALL three must allow the action (unless there's an explicit Deny, which always wins).

## Test Breakdown

### Test 1: GetObject (Allowed)
- ✅ Identity policy allows `s3:GetObject`
- ✅ Bucket policy allows Alice to `s3:GetObject`
- ✅ SCP allows `s3:*`
- **Result**: Allowed

### Test 2: PutObject (Denied by Bucket Policy)
- ✅ Identity policy allows `s3:PutObject`
- ❌ **Bucket policy explicitly DENIES Alice's `s3:PutObject`**
- ✅ SCP allows `s3:*`
- **Result**: Explicit Deny (bucket policy wins)

### Test 3: DeleteObject (No Identity Permission)
- ❌ Identity policy does NOT grant `s3:DeleteObject`
- (Bucket policy not checked because identity policy already denies)
- **Result**: Implicit Deny

## Use Cases

This pattern is essential for:
- **Shared data buckets** between AWS accounts
- **Cross-account KMS key access**
- **SNS/SQS cross-account subscriptions**
- **Lambda function access to S3 in different accounts**
