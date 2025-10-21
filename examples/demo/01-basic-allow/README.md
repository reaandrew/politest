# Demo 1: Basic Allow

The simplest possible example - a policy that allows S3 read operations.

## What This Demonstrates

- Basic policy testing with `politest`
- **Allowed** actions appear as green checkmarks
- **Denied** actions (implicitDeny) show what's NOT permitted

## Run This Demo

```bash
politest --scenario scenario.yml
```

## Expected Output

```
Running 3 test(s)...

[1/3] GetObject is allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/3] ListBucket is allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)

[3/3] PutObject is denied (not in policy)
  ✓ PASS: implicitDeny (matched: -)

========================================
Test Results: 3 passed, 0 failed
========================================
```

## Policy Structure

The policy allows:
- ✅ `s3:GetObject` - Read objects from buckets
- ✅ `s3:ListBucket` - List bucket contents
- ❌ Everything else is implicitly denied (no matching statement)
