# Demo 4: Implicit Deny

Demonstrates what happens when a policy grants NO permissions for an action.

## What This Demonstrates

- **Implicit Deny**: The default state when no Allow statement matches
- By default, everything is denied unless explicitly allowed
- This is AWS's "deny by default" security model

## Run This Demo

```bash
politest --scenario scenario.yml
```

## Expected Output

```
Running 4 test(s)...

[1/4] EC2 DescribeInstances is allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/4] S3 GetObject is implicitly denied (no statement)
  ✓ PASS: implicitDeny (matched: -)

[3/4] IAM CreateUser is implicitly denied (no statement)
  ✓ PASS: implicitDeny (matched: -)

[4/4] Lambda InvokeFunction is implicitly denied (no statement)
  ✓ PASS: implicitDeny (matched: -)

========================================
Test Results: 4 passed, 0 failed
========================================
```

## AWS Default Security Model

```
Everything is DENIED by default
↓
Only explicitly allowed actions are permitted
↓
Explicit Denies override any Allows
```

In this policy:
- ✅ `ec2:Describe*` - Explicitly allowed
- ❌ S3, IAM, Lambda, etc. - **Implicitly denied** (no matching Allow statement)

Notice how `matched: -` indicates no policy statement matched these actions.
