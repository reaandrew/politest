# Demo 3: Explicit Deny

Demonstrates the difference between explicit deny and implicit deny.

## What This Demonstrates

- **Explicit Deny**: Policy has a `"Effect": "Deny"` statement
- **Implicit Deny**: No matching Allow statement exists
- Explicit denies **always win** over any Allow statements

## Run This Demo

```bash
politest --scenario scenario.yml
```

## Expected Output

```
Running 4 test(s)...

[1/4] GetObject is allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/4] DeleteObject is explicitly denied
  ✓ PASS: explicitDeny (matched: PolicyInputList.1)

[3/4] DeleteBucket is explicitly denied
  ✓ PASS: explicitDeny (matched: PolicyInputList.1)

[4/4] PutObject is implicitly denied (no statement)
  ✓ PASS: implicitDeny (matched: -)

========================================
Test Results: 4 passed, 0 failed
========================================
```

## Policy Evaluation Logic

```
1. Explicit Deny? → Denied (stops evaluation)
2. Allow exists? → Allowed
3. Otherwise → Implicit Deny
```

In this policy:
- ✅ `s3:GetObject`, `s3:ListBucket` - Explicitly allowed
- ❌ `s3:DeleteObject`, `s3:DeleteBucket` - **Explicitly denied**
- ❌ Everything else (e.g., `s3:PutObject`) - Implicitly denied (no matching Allow)
