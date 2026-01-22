# ev-node Release Guide

This document covers the release process for ev-node components:

- **Docker Image Releases** - Automated via GitHub workflows (for deployable applications)
- **Go Module Releases** - Manual process for library packages and dependencies

---

## Support Policy

**Version Support:**

- We provide support for **two major versions** (current and previous)
- Supported versions receive:
  - Security fixes
  - Critical bug fixes
- Older versions are considered end-of-life and will not receive updates

**Example:**

- If current version is v2.x.x, we support v2.x.x and v1.x.x
- When v3.0.0 is released, v1.x.x reaches end-of-life

---

## Release Workflow Overview

### Automated Steps

When a SemVer-compliant tag is pushed (matching pattern `**/v*.*.*`), the GitHub workflow automatically:

1. ✅ Creates a new branch `release/<tag-name>` from the tagged commit
2. ✅ Analyzes git changes since the last tag (commits, diffs, file statistics)
3. ✅ Generates a basic structured changelog (grouped by conventional commit types)
4. ✅ Commits the generated changelog to the release branch
5. ✅ Creates a Pull Request for the release branch
6. ✅ Builds multi-platform Docker images (amd64, arm64)
7. ✅ Publishes images to GitHub Container Registry (GHCR)
8. ✅ Creates a draft GitHub Release with a Claude AI prompt for manual enhancement

### Manual Steps

After the automated workflow completes:

1. 📝 Review the Pull Request for `release/<tag-name>`
2. 📝 (Optional) Enhance the changelog using Claude AI:
   - Copy the Claude prompt from the draft GitHub Release
   - Use it in your IDE with Claude integration
   - Update the changelog file with the enhanced output
   - Commit and push changes to the release branch
3. 📝 Edit and refine the generated changelog as needed
4. 📝 Add **recommended upgrade priority** (Critical/High/Medium/Low) to the GitHub Release
5. 📝 Add **general description** of the release to the GitHub Release
6. 📝 Ensure **tested upgrade paths** are documented (1-2 lines in changelog)
7. ✅ Merge the Pull Request to bring changelog into `main`
8. ✅ Update the draft GitHub Release with your refined changelog (if enhanced)
9. ✅ Publish the GitHub Release (change from draft to published)
10. 📢 Publish announcement message in **public Slack channel**
11. 📢 Publish announcement message in **public Telegram channel**

### Release Branch Lifecycle

The `release/<tag-name>` branch is automatically created and should follow this lifecycle:

**1. Review and Refine (Before Publishing)**

- Review the auto-generated changelog in the release branch
- Make any necessary edits to improve clarity
- Add upgrade priority, general description, and tested upgrade paths

**2. Update GitHub Release**

- Copy the refined changelog to the draft GitHub Release
- Verify all information is accurate

**3. Publish Release**

- Publish the GitHub Release (change from draft)

**4. Merge Pull Request to Main**

- **Always merge the Pull Request to bring the changelog into `main`**
- This keeps the repository's changelog files in sync
- Ensures the release notes are preserved in the repository
- The workflow creates a PR automatically for easy review and merge

```bash
# Option 1: Merge via GitHub UI (recommended)
# Review the PR and click "Merge Pull Request"

# Option 2: Merge via command line
git checkout main
git pull origin main
git merge release/evm/v0.2.0
git push origin main

# The release branch can be deleted after merging (GitHub will prompt you)
```

**Why merge back?**

- Keeps CHANGELOG files updated in the repository
- Provides a git history of all releases
- Ensures the next release can build upon previous changelogs
- Makes changelog available for the next release

### Release Priority Guidelines

When adding upgrade priority to releases, use these guidelines:

- **Critical**: Security vulnerabilities, data loss bugs, or breaking issues requiring immediate upgrade
- **High**: Important bug fixes, significant performance improvements, or recommended features
- **Medium**: Regular feature releases, minor bug fixes, general improvements
- **Low**: Optional features, documentation updates, or non-critical enhancements

---

## Changelog Management

### Overview

- **Source of Truth**: GitHub Releases (manually curated)
- **Basic Generation**: Automated bash script (conventional commit grouping)
- **Enhancement Tool**: Claude AI (manual, via provided prompt)
- **Analysis Method**: Git commits, diffs, and file statistics since last tag

### Changelog Workflow

The release workflow automatically generates a basic structured changelog:

1. When you push a tag, the workflow identifies the previous tag
2. Collects git commits, file statistics, and code diffs since the previous tag
3. Generates a structured changelog grouped by conventional commit types (feat, fix, chore, other)
4. Creates a new branch `release/<tag-name>`
5. Commits the basic changelog to this branch
6. Creates a Pull Request for review
7. Creates a draft GitHub Release with a Claude AI prompt for manual enhancement

**The draft release includes:**

- Basic changelog from the release branch
- A ready-to-use Claude prompt in a collapsible section
- Instructions for enhancing the changelog using Claude in your IDE

**You can then:**

- (Optional) Copy the Claude prompt and use it in your IDE to generate an enhanced changelog
- Update the changelog file in the release branch with Claude's output
- Review and edit the changelog as needed
- Merge the Pull Request to bring changes into `main`
- Update the GitHub Release with your final changelog

### Maintaining CHANGELOG.md

Follow these best practices for maintaining `./CHANGELOG.md`:

**Format:**

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- New feature X
- New feature Y

### Changed

- Modified behavior of Z

### Fixed

- Bug fix for issue #123

### Security

- Security patch for vulnerability ABC

### Tested Upgrades

- Tested: v0.2.x → v0.3.0
- Tested: v0.1.5 → v0.3.0 (multi-version jump)

## [0.3.0] - 2026-01-15

### Added

- Feature A
- Feature B

### Fixed

- Critical bug C

### Tested Upgrades

- Tested: v0.2.3 → v0.3.0
```

**Categories:**

- **Added**: New features
- **Changed**: Changes to existing functionality
- **Deprecated**: Features marked for removal
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security-related changes
- **Tested Upgrades**: 1-2 lines documenting which upgrade paths were tested in E2E tests

**Important:** Always include **Tested Upgrades** section in the changelog for each release, documenting which version upgrade paths were validated through E2E testing.

### Example: Documenting Tested Upgrades

After running E2E upgrade tests, document the results in your changelog. This helps users understand which upgrade scenarios have been validated:

**Good Examples:**

```markdown
### Tested Upgrades

- ✅ v1.0.0-beta.10 → v1.0.0-beta.11 (single version)
- ✅ v1.0.0-beta.9 → v1.0.0-beta.11 (skip version)
- ⚠️ v1.0.0-beta.8 → v1.0.0-beta.11 (requires manual data migration - see upgrade guide)
```

```markdown
### Tested Upgrades

- ✅ v0.2.3 → v0.3.0 (standard upgrade)
- ✅ v0.2.0 → v0.3.0 (multi-version jump)
- ❌ v0.1.x → v0.3.0 (not supported - upgrade to v0.2.x first)
```

**What to include:**

- Use ✅ for successful/recommended upgrades
- Use ⚠️ for upgrades that work but require special steps
- Use ❌ for unsupported/untested upgrade paths
- Note if manual intervention is required
- Link to upgrade guides for complex migrations

### Commit Message Best Practices

While the changelog is based on `./CHANGELOG.md`, following conventional commit messages helps with project organization:

**Examples:**

```bash
feat(evm): add support for blob transactions
fix(da): resolve namespace collision in batch submission
docs: update installation guide
perf(block): optimize block validation pipeline
```

**Breaking Changes:**

Mark breaking changes prominently in CHANGELOG.md:

```markdown
### Changed

- **BREAKING**: Sequencer interface now requires context parameter
```

---

## Communication and Announcements

### GitHub Releases

**GitHub Releases** are the official source of truth for all releases.

Each release should include:

- Version number and tag
- **Upgrade Priority**: Critical/High/Medium/Low
- **General Description**: Overview of the release
- **Changelog**: Generated and curated list of changes
- **Tested Upgrade Paths**: 1-2 lines from CHANGELOG.md
- **Breaking Changes**: Highlighted prominently (if any)
- **Installation Instructions**: Links to documentation
- **Known Issues**: Any outstanding issues (if applicable)

### Slack Announcements

After publishing a GitHub Release, post an announcement to the **public Slack channel**:

**Template:**

```
🚀 **ev-node v0.3.0 Released**

Upgrade Priority: [Medium]

This release includes [brief description].

Key highlights:
• Feature X
• Bug fix Y
• Performance improvement Z

Tested upgrade paths: v0.2.x → v0.3.0

📦 Release notes: https://github.com/evstack/ev-node/releases/tag/v0.3.0
📚 Documentation: https://docs.evstack.io

Docker images:
• ghcr.io/evstack/ev-node-evm:v0.3.0
```

### Telegram Announcements

Post the same announcement to the **public Telegram channel**:

**Template:**

```
🚀 ev-node v0.3.0 Released

⚡ Upgrade Priority: Medium

This release includes [brief description].

Key highlights:
✅ Feature X
✅ Bug fix Y
✅ Performance improvement Z

Tested upgrade paths: v0.2.x → v0.3.0

Release notes: https://github.com/evstack/ev-node/releases/tag/v0.3.0
Documentation: https://docs.evstack.io

Docker: ghcr.io/evstack/ev-node-evm:v0.3.0
```

---

## Docker Image Releases (Automated)

### When to Use

Release deployable applications (EVM nodes, test apps, etc.) as Docker images.

### Quick Steps

```bash
# 1. Ensure CI passes on main
# 2. Run E2E upgrade tests and document results
# 3. Create and push tag (workflow will generate changelog from git history)
git tag evm/v0.2.0
git push origin evm/v0.2.0

# 4. Monitor workflow
# GitHub → Actions → Release workflow

# 5. Review Pull Request (release/evm/v0.2.0)
# GitHub → Pull Requests → Review the automated PR

# 6. (Optional) Enhance changelog with Claude AI
# Copy prompt from draft GitHub Release
# Use in IDE with Claude, then update changelog file

# 7. Merge Pull Request
# GitHub → Pull Requests → Merge PR (or use git commands)

# 8. Complete GitHub Release (add priority, description, tested upgrade paths)
# 9. Publish the release
# 10. Announce in Slack and Telegram
```

**Note:** The workflow generates a basic structured changelog automatically by grouping conventional commits. You can optionally enhance it using Claude AI via the provided prompt in the draft GitHub Release. Just ensure your commit messages are clear and descriptive.

### Tag Format

Use the tag format: `{app-name}/v{major}.{minor}.{patch}`

The tag name corresponds to the app directory at `./apps/{app-name}/`

**Examples:**

- `evm/v0.2.0` → Releases `./apps/evm/`
- `testapp/v1.0.0` → Releases `./apps/testapp/`
- `grpc/v2.1.3` → Releases `./apps/grpc/`

**Note:** Tags do NOT include the "apps/" prefix, even though app directories are located at `./apps/<name>/`

### Automated Process

When you push a tag, the release workflow automatically:

1. ✅ Validates tag format and app directory structure
2. ✅ Creates release branch with generated changelog
3. ✅ Builds multi-platform Docker image (amd64, arm64)
4. ✅ Publishes to GitHub Container Registry (GHCR):
   - Version tag: `ghcr.io/evstack/ev-node-{app}:v0.2.0`
   - Latest tag: `ghcr.io/evstack/ev-node-{app}:latest`
5. ✅ Creates draft GitHub Release

### Requirements

- App directory must exist at `./apps/{app-name}/`
- Dockerfile must exist at `./apps/{app-name}/Dockerfile`
- Tag must match pattern `**/v*.*.*`
- CI must pass on main branch

---

## Go Module Releases (Manual)

This section outlines the release process for all Go packages in the ev-node repository. Packages must be released in a specific order due to inter-dependencies.

### Package Dependency Graph

```txt
                        ┌──────────┐
                        │   core   │ (zero dependencies)
                        └────┬─────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
   ┌─────────┐         ┌─────────┐      ┌──────────────┐
   │   da    │         │ ev-node │      │execution/evm │
   └─────────┘         └────┬────┘      └──────────────┘
                            │
                            │
                            ▼
                      ┌───────────┐
                      │ apps/evm  │
                      └───────────┘
```

### Release Order

Packages must be released in the following order:

#### Phase 1: Core Package

1. **github.com/evstack/ev-node/core**
   - Path: `./core`
   - Dependencies: None (zero-dependency package)
   - Foundation package containing all interfaces and types

#### Phase 2: First-Level Dependencies

These packages only depend on `core` and can be released in parallel after `core`:

1. **github.com/evstack/ev-node/da** - Path: `./da`
2. **github.com/evstack/ev-node** - Path: `./` (root)
3. **github.com/evstack/ev-node/execution/evm** - Path: `./execution/evm`

#### Phase 3: Application Packages

These packages have the most dependencies and should be released last:

- **github.com/evstack/ev-node/apps/evm** - Path: `./apps/evm`

### Release Process

**IMPORTANT**: Each module must be fully released and available on the Go proxy before updating dependencies in dependent modules.

**Before Starting:**

- Create a protected version branch (e.g., `v0` for major versions, `v0.3` for minor breaking changes)
- Ensure `CHANGELOG.md` is up to date with all changes properly categorized
- Remove all `replace` directives from go.mod files
- Run E2E tests and document tested upgrade paths in CHANGELOG.md

#### Phase 1: Release Core

```bash
cd core

# Ensure all changes merged, tests pass, go mod tidy
git tag core/v0.3.0
git push origin core/v0.3.0

# Wait 5-10 minutes for Go proxy propagation
go list -m github.com/evstack/ev-node/core@v0.3.0
```

#### Phase 2: Release First-Level Dependencies

After core is available:

```bash
# Update and release da
cd da
go get github.com/evstack/ev-node/core@v0.3.0
go mod tidy
git tag da/v0.3.0
git push origin da/v0.3.0

# Update and release main ev-node
cd ..
go get github.com/evstack/ev-node/core@v0.3.0
go mod tidy
git tag v0.3.0
git push origin v0.3.0

# Update and release execution/evm
cd execution/evm
go get github.com/evstack/ev-node/core@v0.3.0
go mod tidy
git tag execution/evm/v0.3.0
git push origin execution/evm/v0.3.0

# Verify all are available
go list -m github.com/evstack/ev-node/da@v0.3.0
go list -m github.com/evstack/ev-node@v0.3.0
go list -m github.com/evstack/ev-node/execution/evm@v0.3.0
```

#### Phase 3: Release Applications

After all dependencies are available:

```bash
# Update and release apps/evm
cd apps/evm
go get github.com/evstack/ev-node/core@v0.3.0
go get github.com/evstack/ev-node/da@v0.3.0
go get github.com/evstack/ev-node/execution/evm@v0.3.0
go get github.com/evstack/ev-node@v0.3.0
go mod tidy
git tag apps/evm/v0.3.0
git push origin apps/evm/v0.3.0

# Verify availability
go list -m github.com/evstack/ev-node/apps/evm@v0.3.0
```

**Note:** For Go modules, the `apps/evm` tag IS used because it's a Go module path. The restriction on not using "apps/" prefix applies only to Docker image release tags.

---

## Common Release Scenarios

### Scenario 1: Release Single App (Docker Only)

```bash
# Update CHANGELOG.md with release notes
# Tag and push - automation handles the rest
git tag evm/v0.2.0
git push origin evm/v0.2.0

# Review Pull Request
# Optionally enhance with Claude AI (prompt in draft release)
# Merge PR
# Complete GitHub Release with priority and description
# Publish and announce in Slack and Telegram
```

### Scenario 2: Release Multiple Apps

```bash
# Release apps independently
git tag evm/v0.2.0
git tag testapp/v1.0.0
git push origin evm/v0.2.0 testapp/v1.0.0

# Each triggers its own workflow, creates separate release branches and PRs
```

### Scenario 3: Full Go Module Release

```bash
# 1. Update CHANGELOG.md with all changes
# 2. Release core
git tag core/v0.3.0 && git push origin core/v0.3.0

# 3. Wait 5-10 min, update deps, then release first-level
git tag da/v0.3.0 && git push origin da/v0.3.0
git tag v0.3.0 && git push origin v0.3.0
git tag execution/evm/v0.3.0 && git push origin execution/evm/v0.3.0

# 4. Wait, update deps, then release apps
git tag apps/evm/v0.3.0 && git push origin apps/evm/v0.3.0

# 5. Review and merge Pull Requests, complete GitHub Releases
# 6. Announce in Slack and Telegram
```

### Scenario 4: Hotfix/Patch Release

```bash
# For Docker images - delete and recreate
git tag -d evm/v0.2.0
git push origin :refs/tags/evm/v0.2.0

# Fix code, update CHANGELOG.md, create new tag
git tag evm/v0.2.1
git push origin evm/v0.2.1

# For Go modules - create new patch version
# Do NOT delete Go module tags - create v0.3.1 instead
```

---

## Verification

### Docker Image Release

```bash
# Check workflow status
# GitHub → Actions → Release

# Check Pull Request exists
# GitHub → Pull Requests → release/evm/v0.2.0

# Or check release branch
git fetch origin
git checkout release/evm/v0.2.0

# Pull and test image
docker pull ghcr.io/evstack/ev-node-evm:v0.2.0
docker run ghcr.io/evstack/ev-node-evm:v0.2.0 --version

# Check GHCR
# GitHub → Packages → ev-node-evm

# Verify GitHub Release exists (draft initially)
# GitHub → Releases
```

### Go Module Release

```bash
# Verify module is available
go list -m github.com/evstack/ev-node/core@v0.3.0

# Test in a consumer project
go get github.com/evstack/ev-node/core@v0.3.0
```

---

## Troubleshooting

### Docker Releases

**"App directory does not exist"**

- Ensure tag name corresponds to app directory: `evm/v0.2.0` → `./apps/evm/`
- Check spelling and case sensitivity
- Remember: tags do NOT include "apps/" prefix

**"Dockerfile not found"**

- Verify Dockerfile exists at `./apps/{app-name}/Dockerfile`
- Check filename is exactly `Dockerfile`

**"Image not found" in tests**

- Wait for Docker build workflow to complete
- Check workflow dependencies in Actions tab

**"Release branch not created"**

- Check workflow logs for errors
- Verify previous tag detection is working correctly
- Ensure git history is accessible with full depth

**"Pull Request creation failed"**

- Check workflow logs for errors
- Verify repository permissions for github-actions bot
- Ensure the release branch was created successfully

**"Claude prompt not showing in release"**

- Check the draft GitHub Release body
- Look for the collapsible "Click to expand Claude Prompt" section
- Verify git diff data was captured properly in the workflow

### Go Module Releases

**Go proxy delay**

- Wait 5-30 minutes for propagation
- Use `go list -m` to verify availability
- Check <https://proxy.golang.org/>

**Dependency version conflicts**

- Ensure all dependencies are released before dependent modules
- Verify go.mod has correct versions
- Remove `replace` directives

---

## Best Practices

### Before Releasing

- ✅ All changes merged to `main`
- ✅ CI workflow passes
- ✅ `CHANGELOG.md` updated with all changes
- ✅ Tested upgrade paths documented in CHANGELOG.md (1-2 lines)
- ✅ Documentation updated
- ✅ E2E tests completed successfully
- ✅ Local testing complete
- ✅ Remove `replace` directives from go.mod files

### Semantic Versioning

- **Major (v2.0.0)**: Breaking changes, incompatible API changes
- **Minor (v1.1.0)**: New features, backward compatible
- **Patch (v1.0.1)**: Bug fixes, backward compatible

### Version Synchronization

While modules can have independent versions, keep major versions synchronized across related modules for easier dependency management.

### Tag Messages

```bash
# Good: Annotated tag with description
git tag -a evm/v0.2.0 -m "Release EVM v0.2.0

Features:
- Added feature X
- Improved performance Y

Bug fixes:
- Fixed issue Z
"

# Avoid: Lightweight tag without description
git tag evm/v0.2.0  # Less informative
```

### Release Checklist Template

Use this checklist for each release:

```markdown
## Release v0.3.0 Checklist

### Pre-Release

- [ ] All PRs merged to main
- [ ] CI passing on main
- [ ] CHANGELOG.md updated with tested upgrade paths
- [ ] E2E tests passed
- [ ] Documentation updated
- [ ] Replace directives removed

### Release

- [ ] Tag created and pushed
- [ ] Workflow completed successfully
- [ ] Pull Request created and reviewed
- [ ] Changelog enhanced with Claude (if desired)
- [ ] Changelog refined as needed
- [ ] Pull Request merged
- [ ] Docker images verified
- [ ] Go modules verified (if applicable)

### Post-Release

- [ ] GitHub Release published with:
  - [ ] Upgrade priority
  - [ ] General description
  - [ ] Tested upgrade paths
  - [ ] Breaking changes (if any)
- [ ] Slack announcement posted
- [ ] Telegram announcement posted
- [ ] Documentation site updated
```

---

## Important Notes

1. **Breaking Changes**: If a module introduces breaking changes, all dependent modules must be updated and released with appropriate version bumps.

2. **Testing**: Always test the release process in a separate branch first, especially when updating multiple modules.

3. **Go Proxy Cache**: The Go module proxy may take up to 30 minutes to fully propagate new versions. Be patient and verify availability before proceeding to dependent modules.

4. **Rollback Plan**:
   - **Docker images**: Can delete and recreate tags
   - **Go modules**: NEVER delete tags. Create a new patch version instead (e.g., v0.3.1) to avoid Go proxy issues.

5. **Protected Branches**: Create version branches (e.g., `v0`, `v0.3`) for maintaining release history and backporting fixes.

6. **Release Branches and Pull Requests**: The workflow automatically creates `release/<tag-name>` branches and corresponding Pull Requests. Review the PR, optionally enhance the changelog with Claude AI using the provided prompt, then merge the PR to bring changes into `main`.

7. **Changelog as Source**: The `./CHANGELOG.md` file in the repository is the base for all generated release notes. Keep it well-maintained and up-to-date.

8. **Communication is Key**: Always announce releases in both Slack and Telegram channels to keep the community informed.

9. **Tested Upgrade Paths**: Always document which upgrade paths were tested in E2E tests in the CHANGELOG.md (1-2 lines per release). This helps users understand which upgrade scenarios have been validated.

10. **Tag Format Distinction**:
    - **Docker releases**: Use tags WITHOUT "apps/" prefix (e.g., `evm/v0.2.0`)
    - **Go module releases**: Use full module path in tags (e.g., `apps/evm/v0.3.0`)
