# Security Policy

## Supported Versions

We actively support the latest release of politest. Security updates are provided for:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

We recommend always using the latest version to ensure you have all security patches.

## Security Features

### Automated Security Scanning

Every commit is scanned by multiple security tools:

- **govulncheck** - Go vulnerability scanner (checks standard library and dependencies)
- **GitGuardian** - Secret detection in git history
- **SonarCloud** - Code security analysis and vulnerability detection
- **Semgrep** - Static application security testing (SAST)

All scans must pass before code is merged to the main branch.

### Dependency Management

- Go dependencies are regularly updated
- Automated vulnerability scanning on every push
- **No `continue-on-error`** - builds FAIL if vulnerabilities are detected
- Monthly dependency updates recommended

### AWS Credentials

politest requires AWS credentials to run IAM policy simulations:

**Best Practices:**
- Use IAM roles with temporary credentials (recommended)
- Use AWS credential profiles (never hardcode credentials)
- Minimum required permission: `iam:SimulateCustomPolicy` (read-only)
- The tool does NOT modify any AWS resources

**CI/CD:**
- GitHub Actions uses OIDC (OpenID Connect) for AWS authentication
- No long-lived AWS access keys are stored in the repository
- IAM role: `GitHubActionsPolitest` with least-privilege permissions

### No Secret Storage

This tool:
- Does NOT store AWS credentials
- Does NOT transmit credentials anywhere except AWS APIs
- Does NOT log sensitive information
- Uses AWS SDK's default credential chain

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please follow these steps:

### 1. DO NOT Open a Public Issue

Security vulnerabilities should be reported privately to avoid exploitation.

### 2. Report via GitHub Security Advisories

**Preferred Method:** Use GitHub's private vulnerability reporting:

1. Go to the [Security tab](https://github.com/reaandrew/politest/security)
2. Click "Report a vulnerability"
3. Fill out the advisory form with details

### 3. Report via Email (Alternative)

If you prefer email, send to: **security@andrewrea.co.uk**

### What to Include

Please provide as much information as possible:

- **Type of vulnerability** (e.g., credential leakage, code execution, denial of service)
- **Affected versions** (if known)
- **Steps to reproduce** the vulnerability
- **Potential impact** and attack scenarios
- **Suggested fix** (if you have one)
- **Your contact information** for follow-up questions

### Example Report

```
Subject: [SECURITY] Potential credential exposure in scenario loading

Description:
When loading scenario files, credentials may be exposed in debug output
if --verbose flag is used with templates containing AWS credentials.

Steps to Reproduce:
1. Create scenario with template: {{.aws_secret_key}}
2. Run: politest --scenario test.yml --verbose
3. Observe credentials in stdout

Impact:
Medium - requires user to explicitly enable verbose mode and use
credentials in templates (bad practice), but could expose secrets
in CI logs.

Suggested Fix:
Redact template variable values in verbose output that match common
secret patterns (AWS keys, tokens, etc.)

Affected Versions:
All versions prior to 1.2.0
```

## Response Timeline

We aim to respond to security reports within:

- **24 hours** - Initial acknowledgment
- **7 days** - Assessment and severity classification
- **30 days** - Fix development and testing
- **60 days** - Public disclosure (coordinated with reporter)

## Severity Classification

We use the CVSS v3.1 scoring system:

| Severity | CVSS Score | Response Time | Fix Timeline |
|----------|------------|---------------|--------------|
| Critical | 9.0-10.0   | 24 hours      | 7 days       |
| High     | 7.0-8.9    | 48 hours      | 14 days      |
| Medium   | 4.0-6.9    | 7 days        | 30 days      |
| Low      | 0.1-3.9    | 14 days       | 60 days      |

## Disclosure Policy

### Coordinated Disclosure

We practice **coordinated disclosure**:

1. Security fix is developed privately
2. Reporter is kept informed of progress
3. Fix is released with version bump
4. Security advisory is published after fix is available
5. Reporter is credited (unless they prefer anonymity)

### Public Advisory

After a fix is released, we will:

- Publish a GitHub Security Advisory
- Update CHANGELOG.md with security note
- Tag the release with security fix details
- Notify users via GitHub releases

### CVE Assignment

For critical and high severity vulnerabilities, we will:

- Request a CVE (Common Vulnerabilities and Exposures) identifier
- Publish details to the National Vulnerability Database (NVD)

## Security Best Practices for Users

### Using politest Safely

1. **Never commit AWS credentials** to git repositories
   ```bash
   # Use environment variables
   export AWS_PROFILE=my-profile
   go run . --scenario test.yml

   # Or use AWS credential files
   # ~/.aws/credentials
   ```

2. **Use `.gitignore` for sensitive files**
   ```gitignore
   .env
   *.env
   credentials.json
   secrets.yml
   ```

3. **Enable GitGuardian pre-commit hook**
   ```bash
   pip install ggshield
   ggshield install -m local
   ```

4. **Review scenario files before running**
   - Check for unexpected template variables
   - Verify policy documents don't contain secrets
   - Inspect `vars_file` contents

5. **Use least-privilege IAM permissions**
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

6. **Keep politest updated**
   ```bash
   # Check for updates
   gh release list --repo reaandrew/politest --limit 1

   # Update to latest
   go install github.com/reaandrew/politest@latest
   ```

### CI/CD Security

1. **Use OIDC for AWS authentication** (not access keys)
2. **Enable branch protection** on main branch
3. **Require status checks** before merging
4. **Enable Dependabot** for dependency updates
5. **Review security alerts** promptly

## Security Scanning Results

Current security status is visible in:

- [GitHub Security Advisories](https://github.com/reaandrew/politest/security/advisories)
- [Dependabot Alerts](https://github.com/reaandrew/politest/security/dependabot)
- [SonarCloud Dashboard](https://sonarcloud.io/project/overview?id=politest)
- CI badge on README.md

## Known Security Limitations

### Read-Only Simulation

politest uses `iam:SimulateCustomPolicy` which is a **read-only API**:
- Does NOT create, modify, or delete IAM resources
- Does NOT attach policies to users/roles
- Does NOT modify AWS account state
- Safe to run in production accounts (with appropriate credentials)

### Template Rendering

politest uses Go's `text/template` for rendering:
- Templates are **NOT sandboxed**
- Malicious templates could read local files
- **Only use trusted scenario files**
- Review scenario files before execution

### Input Validation

YAML parsing uses `gopkg.in/yaml.v3`:
- Standard YAML security considerations apply
- Avoid loading untrusted YAML files
- YAML bombs and billion laughs attacks are possible

## Security Contact

For urgent security issues, please contact:

**security@andrewrea.co.uk**

For general security questions:

- Open a [Discussion](https://github.com/reaandrew/politest/discussions)
- Tag with "security" label

## Attribution

We appreciate responsible disclosure. Security researchers who report vulnerabilities will be credited in:

- GitHub Security Advisory
- CHANGELOG.md
- Release notes

Thank you for helping keep politest secure!
