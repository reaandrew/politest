---
name: GitGuardian Secret Pre-scanner
description: Comprehensive secret detection before commits to prevent credentials from entering repository. Use before committing sensitive changes or configuration files.
---

# GitGuardian Secret Pre-scanner Agent

## Purpose
Comprehensive secret detection before commits to prevent credentials from entering the repository. Provides an extra security layer beyond the pre-commit hook.

## Trigger
`@claude scan-secrets`

## What It Does

1. **Run GitGuardian ggshield**
   - Executes `ggshield secret scan pre-commit` on staged files
   - Scans for 350+ types of secrets:
     - API keys (AWS, GitHub, Stripe, etc.)
     - OAuth tokens
     - Private keys (RSA, SSH, PGP)
     - Database passwords
     - JWT tokens
     - Cloud service credentials
   ```
   GitGuardian Scan: Running...
   Scanning 5 staged files...
   ✅ No secrets detected in staged changes
   ```

2. **Check Common Secret Patterns**
   - Additional patterns beyond GitGuardian:
     - Hardcoded passwords in config
     - API endpoints with credentials in URL
     - Base64 encoded credentials
     - Private keys in comments
     - Test credentials that look real
   ```
   Extended Pattern Scan: ⚠️  WARNING
   - src/config.py:12
     Possible hardcoded password: DEFAULT_PASSWORD = "admin123"
     Recommendation: Use environment variable instead
   ```

3. **Scan Environment Files**
   - Checks for accidentally staged sensitive files:
     - `.env` files
     - `credentials.json`
     - `secrets.yaml`
     - `*.pem`, `*.key` files
     - Cloud provider config files
   ```
   Environment Files Check: ✅ PASS
   - No sensitive files staged
   - .env is properly in .gitignore
   ```

4. **Check for URLs with Credentials**
   - Scans for URLs containing usernames/passwords:
     - `https://user:pass@example.com`
     - `mongodb://admin:secret@localhost`
     - `postgres://user:pw@host/db`
   ```
   URL Credentials Check: ✅ PASS
   - No credentials found in URLs
   ```

5. **Review Git History for Leaked Secrets**
   - Optional: Scans recent commits (last 10)
   - Checks if secrets were committed previously
   - Helps identify if secret rotation is needed
   ```
   History Scan (last 10 commits): ✅ PASS
   - No secrets found in recent commits
   - Last scan: 2025-10-03
   ```

6. **Generate Security Report**
   ```
   Secret Scanning Report
   =====================

   Staged Files: 5
   Scanned Lines: 342

   ✅ GitGuardian: No secrets detected
   ✅ Extended Patterns: No secrets detected
   ✅ Environment Files: No sensitive files staged
   ✅ URL Credentials: No credentials in URLs
   ✅ History Scan: No secrets in recent commits

   Overall: SAFE TO COMMIT ✅

   Recommendations:
   - Continue using environment variables for sensitive config
   - Keep .env files in .gitignore
   - Never commit real credentials, even in tests
   ```

## Important Rules

- **NEVER commit if secrets found** - absolute blocker
- **DO scan staged files** before every commit
- **DO check history** if suspicious patterns found
- **DO suggest** secure alternatives (env vars, secret managers)
- **DO report** exactly where secrets were found (file:line)
- **DO verify** .gitignore includes sensitive files

## Usage Example

```bash
# Before committing
git add .

# Scan for secrets
@claude scan-secrets

# Agent performs comprehensive scan
# Reports any findings with severity levels

# If secrets found:
# Agent shows:
# ❌ SECRET DETECTED: GitHub Token in src/config.py:45
# DO NOT COMMIT - Remove secret and use environment variable

# User removes secret, uses env var
# Re-scan to verify
@claude scan-secrets

# ✅ SAFE TO COMMIT
```

## When to Use

- Before every commit (paranoid mode)
- After adding new configuration files
- Before committing third-party code
- When integrating with external services
- Before pushing to public repositories

## Risk Levels

| Risk | Description | Action |
|------|-------------|--------|
| 🔴 CRITICAL | Real secrets (API keys, tokens) | BLOCK COMMIT |
| 🟠 HIGH | Likely secrets (passwords, keys) | BLOCK COMMIT |
| 🟡 MEDIUM | Possible secrets (patterns match) | WARN, REVIEW |
| 🟢 LOW | Test/dummy credentials | WARN, VERIFY |

## Two-Layer Security Model

This agent complements the pre-commit hook:

1. **Pre-commit Hook** (automatic)
   - Runs on every commit
   - GitGuardian ggshield
   - Fast, blocks bad commits

2. **This Agent** (on-demand)
   - Deeper scanning
   - Extended pattern matching
   - History checking
   - Education and recommendations

## What to Do if Secrets Found

1. **STOP** - Do not commit
2. **Remove** secret from code
3. **Rotate** the secret (invalidate compromised one)
4. **Use** environment variables or secret manager
5. **Update** .gitignore if needed
6. **Re-scan** to verify clean
7. **Commit** only after all clear

## Tools Required

- `Bash` - for running ggshield
- `Read` - for reading staged files
- `Grep` - for pattern matching
- `Glob` - for finding sensitive file patterns
