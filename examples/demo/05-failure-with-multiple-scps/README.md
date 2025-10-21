# Demo 5: Failure with Multiple SCPs

**⭐ This is the showcase demo** - demonstrates politest's enhanced failure output when debugging complex SCP configurations.

## What This Demonstrates

- **The Problem**: You have an admin policy and 10 organizational SCPs. Something is being denied, but which SCP is the culprit?
- **The Solution**: politest immediately shows you:
  - Which SCP contains the denying statement
  - The exact Sid of the denying statement
  - The source file and line numbers
  - The actual statement content from the file

## Run This Demo

```bash
politest --scenario scenario.yml
```

## Expected Output

The first 3 tests will execute, then test 3 will fail:

```
[3/4] S3 DeleteObject denied by SCP (FAILS - showcases output)
  ✗ FAIL:
    Expected: allowed
    Action:   s3:DeleteObject
    Resource: arn:aws:s3:::my-bucket/data.txt
    Got:      explicitDeny

  Matched statements:
    • PermissionsBoundaryPolicyInputList.1 (Sid: PreventDataLoss)
      Source: scp/06-deny-s3-delete.json:4-10

      4:     {
      5:       "Sid": "PreventDataLoss",
      6:       "Effect": "Deny",
      7:       "Action": [
      8:         "s3:DeleteObject",
      9:         "s3:DeleteBucket",
      10:        "s3:DeleteObjectVersion"
      11:      ]
```

## Why This Matters

Without politest, debugging this would require:
1. Manually inspecting all 10 SCP files
2. Searching for which one contains `s3:DeleteObject`
3. Finding the exact statement
4. Understanding why it's denying

With politest, you **immediately see**:
- ✅ The problem is in `scp/06-deny-s3-delete.json`
- ✅ It's the statement with Sid `PreventDataLoss`
- ✅ It's on lines 4-10 of that file
- ✅ Here's the exact statement content

This is invaluable when working with:
- Large organizations with dozens of SCPs
- Inherited policy configurations
- Debugging permission issues in production

## The 10 SCPs

1. `01-allow-compute.json` - EC2, Lambda
2. `02-allow-networking.json` - VPC, Subnets, Security Groups
3. `03-allow-storage-read.json` - S3/EBS read operations
4. `04-allow-databases.json` - RDS, DynamoDB
5. `05-allow-monitoring.json` - CloudWatch, Logs
6. **`06-deny-s3-delete.json`** ← **This one denies deletions!**
7. `07-allow-security.json` - IAM, KMS
8. `08-allow-messaging.json` - SNS, SQS
9. `09-allow-containers.json` - ECS, EKS
10. `10-allow-analytics.json` - Athena, Glue
