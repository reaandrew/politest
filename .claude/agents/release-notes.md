---
name: Release Notes Generator
description: Generate comprehensive, user-friendly release notes from conventional commits. Use before creating GitHub release or after merging to main.
---

# Release Notes Generator Agent

## Purpose
Generate comprehensive, user-friendly release notes from conventional commits. Complements the automated semantic-release system with human-readable documentation.

## Trigger
`@claude release-notes`

## What It Does

1. **Identify Release Scope**
   - Finds last release tag using `git describe --tags --abbrev=0`
   - Gets all commits since last release
   - Determines next version based on conventional commits
   ```
   Release Scope Analysis
   =====================

   Last Release: v0.1.0 (2025-09-15)
   Current HEAD: feat/json-export
   Commits Since Release: 23

   Next Version: v0.2.0 (Minor)
   Reason: Contains 'feat:' commits
   ```

2. **Parse Conventional Commits**
   - Extracts type, scope, description, body
   - Groups by commit type
   - Links to commit SHAs and PRs
   ```
   Commit Classification
   ====================

   Features (feat): 5 commits
   Bug Fixes (fix): 3 commits
   Documentation (docs): 2 commits
   Chores (chore): 8 commits
   Tests (test): 3 commits
   CI (ci): 2 commits
   ```

3. **Group by Category**
   - **Features**: New functionality
   - **Bug Fixes**: Corrections and fixes
   - **Breaking Changes**: API/behavior changes
   - **Performance**: Speed/efficiency improvements
   - **Security**: Security updates
   - **Documentation**: Docs improvements
   - **Internal**: Refactoring, tests, chores
   ```
   Category Breakdown
   =================

   🎉 Features (5):
   - feat(cli): add JSON output format
   - feat(core): add syllable-based acronym generation
   - feat(api): add batch processing endpoint
   - feat(cli): add --max-words option
   - feat(export): add CSV export format

   🐛 Bug Fixes (3):
   - fix(core): remove undefined variable reference
   - fix(cli): handle empty phrase input correctly
   - fix(test): fix flaky coverage test

   📚 Documentation (2):
   - docs(readme): add usage examples
   - docs(api): document JSON output format
   ```

4. **Extract Details from Commit Bodies**
   - Reads commit message bodies for context
   - Extracts issue/PR references (#123)
   - Notes breaking changes
   - Includes co-authors
   ```
   Detailed Commit Analysis
   =======================

   feat(cli): add JSON output format (#45)

   Body:
   Adds --format flag to CLI with json/text options. JSON output
   includes phrase, acronym, and all options used for generation.
   Useful for programmatic consumption.

   References: Closes #42
   Co-Authored-By: Claude <noreply@anthropic.com>
   PR: #45
   ```

5. **Identify Contributors**
   - Parses commit authors
   - Includes co-authors from commit messages
   - Groups unique contributors
   ```
   Contributors (3)
   ===============

   - @reaandrew (15 commits)
   - @claude-bot (8 commits via auto-fix)
   - Claude <noreply@anthropic.com> (Co-author on 12 commits)
   ```

6. **Generate Release Notes**
   ```markdown
   # Release v0.2.0

   **Release Date**: 2025-10-03

   ## 🎉 Features

   - **CLI**: Add JSON output format (#45)
     - New `--format json` option for programmatic consumption
     - Includes phrase, acronym, and all generation options
     - Closes #42

   - **Core**: Add syllable-based acronym generation (#48)
     - New `create_syllable_acronym()` method
     - Generates more pronounceable acronyms using 2-3 char syllables
     - Example: "Application Programming" → "APPRO"

   - **CLI**: Add `--max-words` option (#52)
     - Limit acronym to first N words
     - Useful for long phrases
     - Example: `--max-words 3` for "One Two Three Four" → "OTT"

   - **Export**: Add CSV export format (#55)
     - Export acronyms to CSV for batch processing
     - Includes all metadata and options

   - **API**: Add batch processing endpoint (#58)
     - Process multiple phrases in single request
     - Significant performance improvement for bulk operations

   ## 🐛 Bug Fixes

   - **Core**: Remove undefined variable reference (#50)
     - Fixed NameError in `clean_phrase()` method
     - Resolves CI failures and all acronym generation

   - **CLI**: Handle empty phrase input correctly (#53)
     - Returns proper error message instead of crashing
     - Exits with code 1 as expected

   - **Tests**: Fix flaky coverage test (#57)
     - Redirect coverage data to /tmp to avoid file conflicts
     - Ensures consistent 80% threshold enforcement

   ## 📚 Documentation

   - **README**: Add comprehensive usage examples (#51)
     - Command-line examples for all features
     - JSON output format documentation

   - **API**: Document JSON output format (#54)
     - JSON schema and field descriptions
     - Example requests and responses

   ## 🔧 Internal Changes

   - **CI**: Add auto-fix workflow for CI failures (#46)
   - **CI**: Fix auto-fix workflow permissions (#49)
   - **Tests**: Add JSON output test cases (#47)
   - **Chore**: Update pre-commit hook configuration (#56)
   - **Refactor**: Simplify syllable extraction logic (#59)

   ## 👥 Contributors

   Thank you to all contributors who made this release possible:

   - @reaandrew (15 commits)
   - @claude-bot (8 commits via auto-fix)
   - Claude <noreply@anthropic.com> (Co-author)

   ## 📊 Statistics

   - **Total Commits**: 23
   - **Files Changed**: 18
   - **Lines Added**: 342
   - **Lines Removed**: 87
   - **Test Coverage**: 85% (↑ 5% from v0.1.0)

   ## 🔗 Links

   - [Full Changelog](https://github.com/reaandrew/acronymcreator/compare/v0.1.0...v0.2.0)
   - [All Issues Closed](https://github.com/reaandrew/acronymcreator/milestone/2)
   - [Documentation](https://github.com/reaandrew/acronymcreator/blob/v0.2.0/README.md)

   ---

   **Full Changelog**: https://github.com/reaandrew/acronymcreator/compare/v0.1.0...v0.2.0
   ```

7. **Format for GitHub Release**
   - Markdown formatted
   - Emoji for visual appeal
   - Links to commits, PRs, issues
   - Collapsible sections for details
   ```markdown
   <details>
   <summary>🔧 Internal Changes (8 commits)</summary>

   - chore: update dependencies (#60)
   - test: add coverage for edge cases (#61)
   - ci: optimize workflow caching (#62)
   - refactor: simplify error handling (#63)
   - test: improve test reliability (#64)
   - chore: update pre-commit hooks (#65)
   - ci: add concurrency control (#66)
   - refactor: extract common test fixtures (#67)

   </details>
   ```

## Important Rules

- **DO format** in user-friendly markdown
- **DO link** to commits, PRs, and issues
- **DO group** logically by category
- **DO highlight** breaking changes prominently
- **DO credit** all contributors
- **DO NOT publish** release automatically - show for review
- **WAIT for approval** before creating GitHub release

## Usage Example

```bash
# Generate release notes
@claude release-notes

# Agent analyzes commits
# Groups by category
# Formats release notes
# Shows preview

# User reviews and approves
# Agent can create GitHub release:
@claude create release

# Agent uses gh CLI to create release with notes
```

## When to Use

- Before creating a GitHub release
- After merging feature branch to main
- To review what's changed since last release
- For changelog generation
- For communication to users

## Release Notes Sections

**Always Include**:
- Version number and date
- Features (user-facing)
- Bug fixes (user-facing)
- Breaking changes (if any)
- Contributors

**Optional Sections**:
- Performance improvements
- Security fixes
- Documentation updates
- Internal/technical changes (collapsible)
- Statistics
- Upgrade instructions

## Integration with Semantic Release

This agent complements semantic-release:
- **semantic-release**: Automatic version bump and tag creation
- **This agent**: Human-readable release notes for GitHub release

Workflow:
1. semantic-release determines version (0.1.0 → 0.2.0)
2. semantic-release creates git tag (v0.2.0)
3. This agent generates release notes
4. semantic-release publishes GitHub release with notes

## Tools Required

- `Bash` - for git commands
- `Read` - for reading commit messages
- `Grep` - for parsing commits
- `WebFetch` - for fetching PR/issue details
