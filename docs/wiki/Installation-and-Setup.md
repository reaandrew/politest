# Installation and Setup

This guide walks you through installing politest and configuring AWS credentials for IAM policy testing.

## Table of Contents

- [System Requirements](#system-requirements)
- [Installation Methods](#installation-methods)
- [AWS Credentials Setup](#aws-credentials-setup)
- [Verification](#verification)
- [Next Steps](#next-steps)

## System Requirements

- **Operating System**: Linux, macOS, or Windows
- **AWS Account**: With IAM permissions to call `SimulateCustomPolicy`
- **AWS Credentials**: Configured via environment variables, AWS CLI, or IAM roles

### Required IAM Permission

politest requires the `iam:SimulateCustomPolicy` permission. This is a **read-only** API call that does not modify any AWS resources.

**Minimal IAM policy:**

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "iam:SimulateCustomPolicy",
    "Resource": "*"
  }]
}
```

## Installation Methods

### Method 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform:

**Linux (amd64):**
```bash
wget https://github.com/reaandrew/politest/releases/latest/download/politest-linux-amd64
chmod +x politest-linux-amd64
sudo mv politest-linux-amd64 /usr/local/bin/politest
```

**Linux (arm64):**
```bash
wget https://github.com/reaandrew/politest/releases/latest/download/politest-linux-arm64
chmod +x politest-linux-arm64
sudo mv politest-linux-arm64 /usr/local/bin/politest
```

**macOS (Intel):**
```bash
wget https://github.com/reaandrew/politest/releases/latest/download/politest-darwin-amd64
chmod +x politest-darwin-amd64
sudo mv politest-darwin-amd64 /usr/local/bin/politest
```

**macOS (Apple Silicon):**
```bash
wget https://github.com/reaandrew/politest/releases/latest/download/politest-darwin-arm64
chmod +x politest-darwin-arm64
sudo mv politest-darwin-arm64 /usr/local/bin/politest
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri https://github.com/reaandrew/politest/releases/latest/download/politest-windows-amd64.exe -OutFile politest.exe
# Add to PATH or move to desired location
```

### Method 2: Build from Source

Requires Go 1.21 or later:

```bash
# Clone the repository
git clone https://github.com/reaandrew/politest.git
cd politest

# Build the binary
go build -o politest .

# Optionally install to /usr/local/bin
sudo mv politest /usr/local/bin/
```

### Method 3: Go Install

```bash
go install github.com/reaandrew/politest@latest
```

## AWS Credentials Setup

politest uses the standard AWS credential chain. Configure credentials using one of these methods:

### Option 1: Environment Variables

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"  # Optional, defaults to us-east-1

# For temporary credentials (STS)
export AWS_SESSION_TOKEN="your-session-token"
```

### Option 2: AWS CLI Configuration

If you have the AWS CLI installed and configured:

```bash
aws configure
```

This creates `~/.aws/credentials` and `~/.aws/config` files that politest will automatically use.

### Option 3: AWS Vault (Recommended for Security)

[AWS Vault](https://github.com/99designs/aws-vault) securely stores credentials in your system keychain:

```bash
# Install aws-vault
brew install aws-vault  # macOS
# or download from https://github.com/99designs/aws-vault/releases

# Add credentials
aws-vault add personal

# Run politest with aws-vault
aws-vault exec personal -- politest --scenario test.yml
```

### Option 4: IAM Roles (EC2, ECS, Lambda)

When running in AWS services, politest automatically uses the IAM role attached to the resource. No credential configuration needed!

### Option 5: GitHub Actions OIDC

For CI/CD, use GitHub's OIDC provider to assume IAM roles without long-lived credentials:

```yaml
- name: Configure AWS Credentials
  uses: aws-actions/configure-aws-credentials@v4
  with:
    role-to-assume: arn:aws:iam::123456789012:role/GitHubActionsRole
    aws-region: us-east-1

- name: Run politest
  run: politest --scenario scenarios/test.yml
```

See the [CI/CD Integration](CI-CD-Integration) guide for complete setup.

## Verification

### Test AWS Credentials

Verify your credentials work:

```bash
# Using AWS CLI
aws sts get-caller-identity

# Output should show:
# {
#     "UserId": "AIDAI...",
#     "Account": "123456789012",
#     "Arn": "arn:aws:iam::123456789012:user/youruser"
# }
```

### Test politest Installation

Check politest is installed correctly:

```bash
# Check version
politest --version

# Should output:
# politest version X.Y.Z
```

### Run a Simple Test

Create a minimal test scenario:

```bash
# Create test directory
mkdir politest-test
cd politest-test

# Create a simple policy
cat > policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "s3:GetObject",
    "Resource": "*"
  }]
}
EOF

# Create test scenario
cat > test.yml <<EOF
policy_json: "policy.json"

tests:
  - name: "GetObject should be allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::test-bucket/*"
    expect: "allowed"

  - name: "PutObject should be denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::test-bucket/*"
    expect: "implicitDeny"
EOF

# Run the test
politest --scenario test.yml
```

**Expected output:**

```
Running 2 test(s)...

[1/2] GetObject should be allowed
  ✓ PASS: allowed (matched: PolicyInputList.1)

[2/2] PutObject should be denied
  ✓ PASS: implicitDeny

========================================
Test Results: 2 passed, 0 failed
========================================
```

If you see this output, congratulations! politest is working correctly.

## Troubleshooting Installation

### "command not found: politest"

**Solution**: Ensure the binary is in your PATH:

```bash
# Check if /usr/local/bin is in PATH
echo $PATH | grep "/usr/local/bin"

# If not, add it to ~/.bashrc or ~/.zshrc
export PATH="/usr/local/bin:$PATH"

# Or specify full path
/usr/local/bin/politest --scenario test.yml
```

### "AccessDenied" when running tests

**Solution**: Verify IAM permissions:

```bash
# Check what user/role you're using
aws sts get-caller-identity

# Test if you have the required permission
aws iam simulate-custom-policy \
  --policy-input-list '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}' \
  --action-names s3:GetObject \
  --resource-arns arn:aws:s3:::test/*
```

If this fails, contact your AWS administrator to grant `iam:SimulateCustomPolicy` permission.

### "no AWS credentials configured"

**Solution**: Set up credentials using one of the methods above. Verify with:

```bash
aws configure list
```

### Binary won't run on macOS (security warning)

**Solution**: macOS Gatekeeper may block unsigned binaries:

```bash
# Allow the binary to run
xattr -d com.apple.quarantine /usr/local/bin/politest

# Or right-click → Open in Finder
```

## Next Steps

Now that politest is installed and configured:

1. **[Follow the Getting Started guide](Getting-Started)** to learn core concepts
2. **[Explore Scenario Formats](Scenario-Formats)** to understand test structures
3. **[Review example tests](https://github.com/reaandrew/politest/tree/main/test/scenarios)** in the repository

---

**Need help?** [Open an issue on GitHub](https://github.com/reaandrew/politest/issues) →
