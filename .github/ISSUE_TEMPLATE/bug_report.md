---
name: Bug Report
about: Report a bug or unexpected behavior
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description

A clear and concise description of the bug.

## To Reproduce

Steps to reproduce the behavior:

1. Create scenario file with '...'
2. Run command '...'
3. See error '...'

**Scenario File:**
```yaml
# Paste your scenario YAML here (redact any sensitive info)
```

**Command:**
```bash
# Paste the exact command you ran
```

## Expected Behavior

What you expected to happen.

## Actual Behavior

What actually happened.

**Error Output:**
```
# Paste error messages or unexpected output
```

## Environment

- **politest version:** (run `politest --version` or `git describe --tags`)
- **Go version:** (run `go version`)
- **OS:** (e.g., Ubuntu 22.04, macOS 14, Windows 11)
- **Installation method:** (binary, `go install`, from source)

## Additional Context

### AWS Environment (if applicable)

- **AWS Region:**
- **Credential method:** (IAM role, profile, environment variables)
- **IAM permissions:** (does your user/role have `iam:SimulateCustomPolicy`?)

### Scenario Details

- **Scenario format:** (legacy or collection)
- **Uses `extends`:** (yes/no)
- **Uses templates:** (yes/no)
- **Uses SCPs:** (yes/no)
- **Uses resource policies:** (yes/no)

### Logs/Debug Output

If you ran with verbose mode or have additional logs, paste them here:

```
# Logs here
```

### Possible Solution

If you have an idea of what might be causing the bug or how to fix it, please share.

## Checklist

- [ ] I have searched existing issues to ensure this is not a duplicate
- [ ] I have redacted any sensitive information (credentials, account IDs, etc.)
- [ ] I have included my politest version and environment details
- [ ] I have provided steps to reproduce the issue
