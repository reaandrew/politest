# Demo 7: Context Conditions

Demonstrates how IAM condition keys control access based on runtime context like IP addresses, MFA status, and principal tags.

**Note**: This demo includes **intentional test failures** to showcase politest's diagnostic output!

## What This Demonstrates

- **IP address restrictions**: Allow access only from specific IP ranges
- **MFA enforcement**: Require multi-factor authentication for sensitive operations
- **Tag-based access control (TBAC)**: Grant permissions based on principal tags
- **Explicit deny with conditions**: Network-based and MFA-based deny statements
- **Time-based restrictions**: Explicit deny outside business hours
- **Multiple context keys**: Combining multiple conditions in a single test
- **Diagnostic output for explicit denies**: See exactly which deny statement matched, with Sid and line numbers

## Run This Demo

```bash
politest --scenario scenario.yml
```

## Expected Output

This demo will show **5 test failures** (3 implicit denies + 2 explicit denies) and **5 passes**:

```
Running 10 test(s)...

[1/10] GetObject from trusted IP (10.0.1.50)
  ✓ PASS: allowed (matched: PolicyInputList.1 - AllowS3ReadFromTrustedIP)

[2/10] GetObject from untrusted IP should fail
  ✗ FAIL: expected allowed, got implicitDeny (matched: -)

  The request was denied because no statement matched.

[3/10] PutObject with MFA enabled
  ✓ PASS: allowed (matched: PolicyInputList.1 - AllowS3WriteWithMFA)

[4/10] PutObject without MFA should fail
  ✗ FAIL: expected allowed, got implicitDeny (matched: -)

  The request was denied because no statement matched.

[5/10] StartInstances with Engineering tag
  ✓ PASS: allowed (matched: PolicyInputList.1 - AllowEC2ForEngineeringDept)

[6/10] StartInstances with Finance tag should fail
  ✗ FAIL: expected allowed, got implicitDeny (matched: -)

  The request was denied because no statement matched.

[7/10] DeleteObject from untrusted network should be explicitly denied
  ✗ FAIL: expected allowed, got explicitDeny (matched: PolicyInputList.1)

  Matched statements:
    • PolicyInputList.1 (Sid: DenyS3DeleteFromUntrustedNetwork)
      Source: policies/conditional-policy.json:64-73

      64:     {
      65:       "Sid": "DenyS3DeleteFromUntrustedNetwork",
      66:       "Effect": "Deny",
      67:       "Action": "s3:DeleteObject",
      68:       "Resource": "*",
      69:       "Condition": {
      70:         "NotIpAddress": {
      71:           "aws:SourceIp": "10.0.1.0/24"
      72:         }

[8/10] TerminateInstances without MFA should be explicitly denied
  ✗ FAIL: expected allowed, got explicitDeny (matched: PolicyInputList.1)

  Matched statements:
    • PolicyInputList.1 (Sid: DenyEC2WithoutMFA)
      Source: policies/conditional-policy.json:75-87

      75:     {
      76:       "Sid": "DenyEC2WithoutMFA",
      77:       "Effect": "Deny",
      78:       "Action": [
      79:         "ec2:TerminateInstances",
      80:         "ec2:DeleteVolume"
      81:       ],
      82:       "Resource": "*",
      83:       "Condition": {
      84:         "BoolIfExists": {
      85:           "aws:MultiFactorAuthPresent": "false"

[9/10] DeleteBucket outside business hours
  ✓ PASS: explicitDeny (matched: PolicyInputList.1 - DenyS3DeleteOutsideBusinessHours)

[10/10] ListBucket from trusted IP with MFA
  ✓ PASS: allowed (matched: PolicyInputList.1 - AllowS3ReadFromTrustedIP)

========================================
Test Results: 5 passed, 5 failed
========================================
```

## Policy Breakdown

### Statement 1: IP Address Restriction
```json
{
  "Sid": "AllowS3ReadFromTrustedIP",
  "Effect": "Allow",
  "Action": ["s3:GetObject", "s3:ListBucket"],
  "Resource": "arn:aws:s3:::secure-bucket/*",
  "Condition": {
    "IpAddress": {
      "aws:SourceIp": "10.0.1.0/24"
    }
  }
}
```
- ✅ **Test 1**: IP 10.0.1.50 is within range → **allowed** (PASS)
- ❌ **Test 2**: IP 192.168.1.1 is outside range → **implicitDeny** (FAIL - we expected "allowed")

### Statement 2: MFA Requirement
```json
{
  "Sid": "AllowS3WriteWithMFA",
  "Effect": "Allow",
  "Action": ["s3:PutObject", "s3:DeleteObject"],
  "Condition": {
    "Bool": {
      "aws:MultiFactorAuthPresent": "true"
    }
  }
}
```
- ✅ **Test 3**: MFA present = true → **allowed** (PASS)
- ❌ **Test 4**: MFA present = false → **implicitDeny** (FAIL - we expected "allowed")

### Statement 3: Tag-Based Access Control
```json
{
  "Sid": "AllowEC2ForEngineeringDept",
  "Effect": "Allow",
  "Action": ["ec2:StartInstances", "ec2:StopInstances"],
  "Condition": {
    "StringEquals": {
      "aws:PrincipalTag/Department": "Engineering"
    }
  }
}
```
- ✅ **Test 5**: Department tag = "Engineering" → **allowed** (PASS)
- ❌ **Test 6**: Department tag = "Finance" → **implicitDeny** (FAIL - we expected "allowed")

### Statement 4: Explicit Deny - Network-Based S3 Delete Protection
```json
{
  "Sid": "DenyS3DeleteFromUntrustedNetwork",
  "Effect": "Deny",
  "Action": "s3:DeleteObject",
  "Resource": "*",
  "Condition": {
    "NotIpAddress": {
      "aws:SourceIp": "10.0.1.0/24"
    }
  }
}
```
- ❌ **Test 7**: DeleteObject from IP 192.168.1.100 (untrusted) with MFA → **explicitDeny** (FAIL - we expected "allowed")
  - **Shows matched statement** with Sid and line numbers!

### Statement 5: Explicit Deny - MFA Required for Destructive EC2 Operations
```json
{
  "Sid": "DenyEC2WithoutMFA",
  "Effect": "Deny",
  "Action": ["ec2:TerminateInstances", "ec2:DeleteVolume"],
  "Resource": "*",
  "Condition": {
    "BoolIfExists": {
      "aws:MultiFactorAuthPresent": "false"
    }
  }
}
```
- ❌ **Test 8**: TerminateInstances without MFA (even with Engineering tag) → **explicitDeny** (FAIL - we expected "allowed")
  - **Shows matched statement** with Sid and line numbers!

### Statement 6: Time-Based Deny
```json
{
  "Sid": "DenyS3DeleteOutsideBusinessHours",
  "Effect": "Deny",
  "Action": "s3:DeleteBucket",
  "Condition": {
    "DateGreaterThan": {"aws:CurrentTime": "2024-01-01T17:00:00Z"},
    "DateLessThan": {"aws:CurrentTime": "2024-01-02T09:00:00Z"}
  }
}
```
- ✅ **Test 9**: Time is 20:00 (8 PM) → **explicitDeny** (PASS - correctly expected)

## Real-World Use Cases

This pattern is essential for:
- **Secure environments**: Restrict access to corporate network IPs
- **Compliance requirements**: Enforce MFA for sensitive operations
- **Organizational policies**: Department-based access control
- **Operational safety**: Prevent destructive operations outside business hours
- **Least privilege**: Grant permissions only when specific conditions are met

## Key Takeaways

1. **Context keys matter**: The same action can be allowed or denied based on runtime context
2. **Explicit deny diagnostic output**: When a deny statement matches, politest shows:
   - The exact Sid (e.g., "DenyS3DeleteFromUntrustedNetwork")
   - Source file and line numbers (e.g., "policies/conditional-policy.json:64-73")
   - The actual condition that triggered the deny
3. **Implicit vs Explicit deny failures**: Compare tests 2, 4, 6 (implicit - no match) vs tests 7, 8 (explicit - shows matched deny statement)
4. **Multiple conditions**: Tests can include multiple context keys (Test 10)
5. **Explicit deny wins**: Even with MFA and correct tags, deny statements take precedence
6. **Test both paths**: This demo intentionally includes failures to showcase diagnostic output
7. **Condition types**: String, Boolean, Numeric, IP Address, Date/Time all supported
