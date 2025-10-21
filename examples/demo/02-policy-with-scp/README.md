# Demo 2: Policy with SCP (Both Allow)

Demonstrates how Service Control Policies (SCPs) work as permission boundaries.

## What This Demonstrates

- SCPs act as **permission boundaries** - both the identity policy AND the SCP must allow an action
- In this case, both allow S3 operations, so everything works
- politest automatically merges SCPs using `PermissionsBoundaryPolicyInputList`

## Run This Demo

```bash
politest --scenario scenario.yml
```

## Expected Output

```
Running 3 test(s)...

[1/3] GetObject allowed by both policy and SCP
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/3] PutObject allowed by both policy and SCP
  ✓ PASS: allowed (matched: PolicyInputList.1)

[3/3] DeleteBucket allowed by both policy and SCP
  ✓ PASS: allowed (matched: PolicyInputList.1)

========================================
Test Results: 3 passed, 0 failed
========================================
```

## How SCPs Work

```
Action allowed? = (Identity Policy allows) AND (SCP allows)
```

In this demo:
- ✅ Identity policy allows `s3:*`
- ✅ SCP allows `s3:*`
- ✅ Result: All S3 actions are allowed

See demo 5 for what happens when an SCP denies an action.
