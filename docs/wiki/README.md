# politest Wiki Documentation

This directory contains comprehensive GitHub Wiki documentation for the politest project.

## Wiki Pages Created

1. **Home.md** - Wiki homepage with overview and quick start
2. **Installation-and-Setup.md** - Installation methods and AWS credential configuration
3. **Getting-Started.md** - Tutorial-style guide for first-time users
4. **Scenario-Formats.md** - Legacy vs Collection format comparison
5. **Template-Variables.md** - Using Go templates for reusable tests
6. **_Sidebar.md** - Wiki navigation sidebar

## Additional Pages to Create

The following pages are referenced but not yet created. You can create them based on the patterns established:

7. **Scenario-Inheritance.md** - Using `extends:` to reuse scenarios
8. **Resource-Policies-and-Cross-Account.md** - Cross-account IAM testing
9. **SCPs-and-RCPs.md** - Service and Resource Control Policies
10. **Context-Conditions.md** - Testing IAM condition keys
11. **Advanced-Patterns.md** - Complex scenarios and best practices
12. **CI-CD-Integration.md** - Automating tests in GitHub Actions/GitLab CI
13. **Troubleshooting.md** - Common issues and solutions
14. **API-Reference.md** - Complete YAML schema reference

## Uploading to GitHub Wiki

### Method 1: Manual Upload (Recommended)

1. **Navigate to your repository wiki:**
   ```
   https://github.com/reaandrew/politest/wiki
   ```

2. **Create/Edit each page:**
   - Click "New Page" or "Edit"
   - Copy content from corresponding `.md` file
   - Use the filename (without .md) as the page title
   - Click "Save Page"

3. **Upload in this order:**
   - Home (must be first)
   - _Sidebar (navigation)
   - All other pages in any order

### Method 2: Git Clone (Advanced)

Once you've created at least one page manually to initialize the wiki:

```bash
# Clone the wiki repository
git clone https://github.com/reaandrew/politest.wiki.git

# Copy all wiki files
cp docs/wiki/*.md politest.wiki/

# Commit and push
cd politest.wiki
git add *.md
git commit -m "Add comprehensive wiki documentation"
git push
```

### Method 3: GitHub CLI

```bash
# Navigate to wiki directory
cd docs/wiki

# For each markdown file
for file in *.md; do
  echo "Create page for $file manually at:"
  echo "https://github.com/reaandrew/politest/wiki/_new"
done
```

## Wiki Structure

The documentation follows a progressive learning path:

```
1. Home
   └─> Quick start and overview

2. Installation and Setup
   └─> Get tool running

3. Getting Started
   └─> First test and basic concepts

4. Core Features (Scenario Formats → Variables → Inheritance → Advanced)
   └─> Build understanding progressively

5. Advanced Usage
   └─> Complex patterns and CI/CD

6. Reference
   └─> API docs and troubleshooting
```

## Content Guidelines

Each wiki page includes:

- **Clear heading hierarchy** - Table of contents for navigation
- **Real code examples** - From actual test scenarios in `test/scenarios/`
- **Links to source** - Direct GitHub links to working examples
- **Progressive complexity** - Simple examples first, then advanced
- **Practical focus** - Real-world use cases, not just API docs

## Updating Wiki Content

When updating test scenarios or adding features:

1. Update corresponding wiki page in `docs/wiki/`
2. Re-upload to GitHub Wiki (manual or git push)
3. Add cross-references between related pages
4. Update _Sidebar.md if adding new pages

## Examples Referenced

The wiki extensively references these test scenarios:

- `test/scenarios/01-07`: Basic policy testing
- `test/scenarios/08-09`: Collection format
- `test/scenarios/10-11`: Context conditions
- `test/scenarios/12-13`: Resource policies and overrides
- `test/scenarios/14-16`: Variables and inheritance
- `test/scenarios/17-18`: RCPs and comprehensive examples

All examples are **working, tested code** from the repository.

## Need Help?

- **Missing content?** Check `CLAUDE.md` for additional technical details
- **Want to contribute?** Create additional wiki pages following the established patterns
- **Found issues?** Open a GitHub issue or discussion

---

**Ready to upload?** Start with the [Manual Upload method](#method-1-manual-upload-recommended) above.
