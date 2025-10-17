---
name: Conventional Commit Formatter
description: Generate properly formatted conventional commit messages for semantic-release. Use before every commit or when commit-msg hook rejects message.
---

# Conventional Commit Formatter Agent

## Purpose
Generate properly formatted conventional commit messages that pass the commit-msg hook and enable semantic-release versioning.

## Trigger
`@claude format-commit`

## What It Does

1. **Analyze Staged Changes**
   - Runs `git diff --cached` to see what's being committed
   - Analyzes file types and changes (added/modified/deleted)
   - Determines scope based on directory structure

2. **Determine Commit Type**
   - **feat**: New feature or functionality
   - **fix**: Bug fix
   - **docs**: Documentation changes only
   - **style**: Code style/formatting (no logic change)
   - **refactor**: Code restructuring without behavior change
   - **test**: Adding or updating tests
   - **build**: Build system or dependency changes
   - **ci**: CI/CD configuration changes
   - **chore**: Maintenance tasks, tooling updates
   - **revert**: Reverting previous commit

3. **Detect Breaking Changes**
   - Scans diff for API changes, removed functions, signature changes
   - Checks for changes that would break existing code
   - Adds `!` to type or `BREAKING CHANGE:` footer if needed

4. **Generate Commit Message**
   ```
   Type(scope): Brief description

   Detailed explanation of:
   - Why this change is needed
   - What problem it solves
   - Any important context

   BREAKING CHANGE: Description of breaking change (if applicable)

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```

5. **Example Outputs**

   **Feature Addition:**
   ```
   feat(cli): add JSON output format option

   Adds --format flag to CLI with json/text options. JSON output includes
   phrase, acronym, and all options used for generation. Useful for
   programmatic consumption and integration testing.

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```

   **Bug Fix:**
   ```
   fix(core): remove undefined variable reference

   Removes test_var assignment to undefined_name_error that was causing
   NameError in clean_phrase method. This was blocking all acronym
   generation functionality.

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```

   **Breaking Change:**
   ```
   feat(api)!: change AcronymOptions constructor signature

   Refactor AcronymOptions to use keyword-only arguments for better
   clarity and future extensibility.

   BREAKING CHANGE: AcronymOptions now requires keyword arguments.
   Change `AcronymOptions(True, 3)` to `AcronymOptions(include_articles=True, min_word_length=3)`

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```

6. **Validation**
   - Runs commit message through `conventional-pre-commit` validator
   - Ensures message will pass commit-msg hook
   - Verifies format is compatible with semantic-release

## Important Rules

- **DO follow** conventional commit format strictly
- **DO explain WHY** not what (what is in the diff)
- **DO keep** subject line under 72 characters
- **DO use** imperative mood ("add" not "added")
- **NEVER commit automatically** - show message and wait for approval
- **DO include** Co-Authored-By for Claude contributions

## Semantic Versioning Impact

The commit type determines version bump:
- `feat:` → Minor version (0.1.0 → 0.2.0)
- `fix:` → Patch version (0.1.0 → 0.1.1)
- `feat!:` or `BREAKING CHANGE:` → Major version (0.1.0 → 1.0.0)
- `docs:`, `style:`, `chore:`, etc. → No version bump

## Usage Example

```bash
# Stage your changes
git add src/acronymcreator/core.py

# Generate commit message
@claude format-commit

# Agent shows:
# ---
# feat(core): add syllable-based acronym generation
#
# Implements create_syllable_acronym method that generates acronyms
# using 2-3 character syllables from each word instead of just first
# letters. Provides more pronounceable acronyms for longer phrases.
# ---
#
# Does this commit message look correct? [yes/no]

# User approves
yes

# Agent commits with the message
```

## When to Use

- Before every commit to ensure proper format
- When commit-msg hook rejects your message
- To understand what type of commit you're making
- To ensure semantic-release will version correctly

## Integration with Workflow

This agent works with:
- **commit-msg hook**: Validates format (enforces compliance)
- **semantic-release**: Uses message to determine version bump
- **changelog generation**: Parses commits to build release notes

## Tools Required

- `Bash` - for git commands
- `Read` - for reading staged changes
- `Grep` - for analyzing diffs
