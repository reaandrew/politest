# politest Demo Examples

A curated set of 7 demos designed for a 10-minute presentation showcasing politest's capabilities.

## Quick Start

```bash
# Run from the project root
cd examples/demo

# Run any demo
politest --scenario 01-basic-allow/scenario.yml
politest --scenario 02-policy-with-scp/scenario.yml
# ... etc
```

## Demo Flow (10 minutes)

### 1. Basic Allow (1 min)
**Directory**: `01-basic-allow/`

The simplest case - a policy that allows S3 read operations.
- Shows basic politest usage
- Demonstrates allowed vs implicitly denied actions

### 2. Policy with SCP (1 min)
**Directory**: `02-policy-with-scp/`

Introduces SCPs as permission boundaries.
- Both policy AND SCP must allow the action
- Shows how SCPs work when they're permissive

### 3. Explicit Deny (1.5 min)
**Directory**: `03-explicit-deny/`

Demonstrates explicit deny statements.
- Explicit deny vs implicit deny
- Shows `explicitDeny` vs `implicitDeny` in output

### 4. Implicit Deny (1 min)
**Directory**: `04-implicit-deny/`

Shows the default "deny by default" behavior.
- No matching Allow statement = implicitly denied
- Shows `matched: -` when no policy statement applies

### 5. ⭐ Failure with Multiple SCPs (2.5 min) ⭐
**Directory**: `05-failure-with-multiple-scps/`

**THE SHOWCASE DEMO** - This is the highlight!

Scenario:
- Admin policy that allows everything
- 10 organizational SCPs (simulating a real enterprise)
- One SCP among the 10 denies S3 delete operations
- Test attempts S3 delete and fails

What politest shows:
```
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
```

**Key Message**: Instead of manually searching through 10 SCP files, politest immediately shows:
- ✅ Which file contains the denying statement
- ✅ The exact Sid and line numbers
- ✅ The statement content from the source file

### 6. Context Conditions (1.5 min)
**Directory**: `07-context-conditions/`

Shows how IAM conditions control access based on runtime context.
- IP address restrictions (trusted vs untrusted)
- MFA requirements (with vs without MFA)
- Tag-based access control (Engineering vs Finance)
- Time-based explicit denies
- Demonstrates both **passing** and **failing** tests for each condition

### 7. Cross-Account Resource Policy (1.5 min)
**Directory**: `06-cross-account-resource-policy/`

Demonstrates complex policy interactions.
- Identity policy + resource policy + SCP
- Cross-account S3 bucket access scenario
- Shows how all three policies must align (or explicit deny wins)

## Demo Tips

### For a Live Demo

1. **Start simple** (demos 1-2): Show basic success cases quickly
2. **Show failures** (demos 3-4): Demonstrate explicit vs implicit deny
3. **Highlight the value** (demo 5): This is your "wow" moment - spend the most time here
4. **Show conditions** (demo 6): Demonstrate context-based access control
5. **Show complexity** (demo 7): Finish with cross-account to show real-world scenarios

### Key Points to Emphasize

- **Declarative testing**: Define expected outcomes in YAML
- **Fast feedback**: Run hundreds of tests in seconds
- **Enhanced debugging**: Source file tracking with line numbers (demo 5)
- **Context-aware testing**: Test policies with runtime conditions like IP, MFA, tags (demo 6)
- **Real-world complexity**: Handle SCPs, resource policies, cross-account (demo 7)
- **CI/CD ready**: Tests can run in your pipeline

### Common Questions

**Q**: "Can I test my own policies?"
**A**: Yes! Just point `policy_json:` to your policy file.

**Q**: "How do I test with my organization's SCPs?"
**A**: Use `scp_paths:` with a glob pattern: `scp/*.json`

**Q**: "Does this make real AWS API calls?"
**A**: Yes, it uses `iam:SimulateCustomPolicy` - but it's a read-only simulation API, no resources are created.

**Q**: "What about testing policies with conditions?"
**A**: Demo 6 shows this - use `context:` to specify runtime context keys like IP address, MFA status, or principal tags.

**Q**: "What about resource policies?"
**A**: Demo 7 shows this - use `resource_policy_json:` and `caller_arn:`.

## Running All Demos

```bash
# Quick test - run all demos
for dir in 0{1..7}-*/; do
    echo "Running $(basename $dir)..."
    politest --scenario "$dir/scenario.yml"
done
```

## Demo Structure

Each demo contains:
- `scenario.yml` - The test configuration
- `policies/` - Identity and resource policies
- `scp/` - Service Control Policies (where applicable)
- `README.md` - Explanation of what the demo shows

## Next Steps

After the demo, direct people to:
- **Main README**: Project overview and installation
- **CLAUDE.md**: Technical implementation details
- **test/scenarios/**: 19 integration test examples
- **examples/athena-policy/**: Real-world example (369 tests)
