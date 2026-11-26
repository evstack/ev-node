# ev-node Release Guide

This document covers the release process for ev-node components:

- **ev-node releases** - Pushing a root semver tag builds the Docker images and publishes the GitHub Release
- **Go Module Releases** - Manual process for library packages and dependencies

---

## Cutting a Release

Pushing a root tag such as `v1.4.0` or `v1.4.0-rc.1` runs `.github/workflows/release.yml`, which:

1. Validates the tag (`v<MAJOR>.<MINOR>.<PATCH>[-PRERELEASE]`) and finds the previous release tag. Build metadata (`+BUILD`) is not supported: Docker tags cannot contain `+`
2. Builds and pushes multi-arch (amd64, arm64) images for every `apps/*/Dockerfile` to `ghcr.io/evstack/ev-node-<app>:<tag>`; stable tags also update `:latest`
3. Generates release notes from the PR titles since the previous tag with [git-cliff](https://git-cliff.org) (`cliff.toml`)
4. Publishes the GitHub Release, with the image list and a full diff link

Tags with a `-` suffix (`-rc.1`, `-beta.1`) are published as prereleases: they never become the latest release and never move `:latest`.

### Steps

```bash
# 1. Make sure CI is green on the commit you are tagging (usually main)

# 2. Optional: preview the release notes
git cliff --unreleased --tag v1.4.0
# or, without installing git-cliff
docker run --rm -v "$PWD":/app -w /app orhunp/git-cliff --unreleased --tag v1.4.0

# 3. Tag and push
git tag -a v1.4.0 -m v1.4.0
git push origin v1.4.0

# 4. Watch GitHub → Actions → Release
```

The release is published directly. Edit it afterwards on GitHub if you need to add context such as tested upgrade paths or operator steps.

### PR Titles Are the Changelog

PRs are squash-merged, so each PR title becomes one release-note line. Titles must follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) (enforced by `semantic-pull-request.yml`):

| Title prefix | Release notes section |
| --- | --- |
| `feat:` | Features |
| `fix:` | Bug fixes |
| `perf:` | Performance |
| `revert:` | Reverts |
| `chore:`, `docs:`, `test:`, `ci:`, `build:`, `refactor:`, `style:` | Not shown |
| `feat!:`, `fix(scope)!:`, any type with `!` | ⚠️ Breaking changes |

- Add `!` before the colon to mark a **breaking** change. Breaking changes are always listed, even for otherwise hidden types such as `refactor!:`.
- An optional scope is shown in bold: `fix(p2p): ...` → **p2p:** ...
- Write titles for node operators: say what changed for them, not how it was implemented.
- Do not edit `CHANGELOG.md`; it is frozen. Release notes live on [GitHub Releases](https://github.com/evstack/ev-node/releases).

### Docker Images

Every app image uses the root version: tag `v1.4.0` publishes `ev-node-evm:v1.4.0`, `ev-node-testapp:v1.4.0`, `ev-node-loadgen:v1.4.0`, and so on. `apps/evm` and `apps/testapp` use `replace github.com/evstack/ev-node => ../..`, so their images are built against the tagged ev-node tree; `apps/loadgen` does not depend on ev-node and is built from its own sources at the tagged commit. App tags (`apps/evm/v*`, ...) are Go module tags only and do not build images.

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

1. **github.com/evstack/ev-node** - Path: `./` (root)
2. **github.com/evstack/ev-node/execution/evm** - Path: `./execution/evm`

#### Phase 3: Application Packages

These packages have the most dependencies and should be released last:

- **github.com/evstack/ev-node/apps/evm** - Path: `./apps/evm`

### Release Process

**IMPORTANT**: Each module must be fully released and available on the Go proxy before updating dependencies in dependent modules.

**Before Starting:**

- Create a protected version branch (e.g., `v0` for major versions, `v0.3` for minor breaking changes)
- Pushing the root tag (`v0.3.0` below) also runs the release workflow: Docker images and the GitHub Release are published from that commit
- Remove `replace` directives from the library modules' go.mod files (`core`, root, `da`, `execution/evm`)
- Keep `replace github.com/evstack/ev-node => ../..` in `apps/*/go.mod`: the images built from the root tag rely on it to use the tagged tree, and no other module imports the apps

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
go list -m github.com/evstack/ev-node@v0.3.0
go list -m github.com/evstack/ev-node/execution/evm@v0.3.0
```

#### Phase 3: Release Applications

After all dependencies are available:

```bash

# Update and release apps/evm
go get github.com/evstack/ev-node/core@v0.3.0
go get github.com/evstack/ev-node/execution/evm@v0.3.0
go get github.com/evstack/ev-node@v0.3.0
go mod tidy
git tag apps/evm/v0.3.0
git push origin apps/evm/v0.3.0

# Verify availability
go list -m github.com/evstack/ev-node/apps/evm@v0.3.0
```

---

## Common Release Scenarios

### Scenario 1: Stable Release

```bash
git cliff --unreleased --tag v1.4.0   # optional preview
git tag -a v1.4.0 -m v1.4.0
git push origin v1.4.0
```

### Scenario 2: Release Candidate

```bash
git tag -a v1.4.0-rc.1 -m v1.4.0-rc.1
git push origin v1.4.0-rc.1
# Published as a prerelease; :latest is not updated
```

Release notes for an rc cover the changes since the previous tag (stable or rc). The final `v1.4.0` notes cover everything since the previous stable release, including all its rcs.

### Scenario 3: Backport Release

```bash
git checkout v1.3.x
# cherry-pick fixes, wait for CI
git tag -a v1.3.1 -m v1.3.1
git push origin v1.3.1
```

The previous tag is found by ancestry, so notes for `v1.3.1` diff against `v1.3.0`, not the newest tag on `main`.

### Scenario 4: Full Go Module Release

```bash
# 1. Core
git tag core/v0.3.0 && git push origin core/v0.3.0

# 2. Wait 5-10 min, update deps, then release first-level
git tag da/v0.3.0 && git push origin da/v0.3.0
git tag v0.3.0 && git push origin v0.3.0
git tag execution/evm/v0.3.0 && git push origin execution/evm/v0.3.0

# 3. Wait, update deps, then release apps
git tag apps/evm/v0.3.0 && git push origin apps/evm/v0.3.0
```

### Scenario 5: Hotfix/Patch Release

```bash
# Fix on main (or a backport branch), then cut a new patch version
git tag -a v1.4.1 -m v1.4.1
git push origin v1.4.1

# Do NOT delete or move released tags: Go modules and images depend on them
```

If a release workflow run fails for a transient reason, re-run it from the Actions tab: it overwrites the tag's images and updates the existing release. A re-run uses the tagged commit, so if the code needs a fix, cut a new patch version instead.

---

## Verification

### Docker Images and GitHub Release

```bash
# Check workflow status
# GitHub → Actions → Release

# Pull and test image
docker pull ghcr.io/evstack/ev-node-evm:v1.4.0
docker run ghcr.io/evstack/ev-node-evm:v1.4.0 --version

# Check GHCR
# GitHub → Packages → ev-node-evm
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

### Release Workflow

#### "Invalid tag format" error

- Ensure tag follows semantic versioning: `v1.2.3`
- Valid examples: `v1.2.3`, `v1.2.3-rc.4`, `v1.2.3-beta.1`
- Delete the bad tag (`git push origin :refs/tags/<tag>`) and push a correct one

#### A PR is missing from the release notes

- `chore`, `docs`, `test`, `ci`, `build`, `refactor` and `style` commits are hidden unless marked breaking (`!`)
- Commits whose subject is not a conventional commit are dropped
- Edit the published release on GitHub to add the line

#### Docker build failed

- Check the `Docker images` job logs in the Release run
- The GitHub Release is only published after all images are pushed; re-run the workflow for transient failures, or cut a new patch version if the code needs a fix
- Every `apps/<app>/Dockerfile` is built; an app without a Dockerfile is skipped

### Go Module Releases

#### Go proxy delay

- Wait 5-30 minutes for propagation
- Use `go list -m` to verify availability
- Check <https://proxy.golang.org/>

#### Dependency version conflicts

- Ensure all dependencies are released before dependent modules
- Verify go.mod has correct versions
- Remove `replace` directives from library modules (keep the `apps/*` ones)

---

## Best Practices

### Before Releasing

- ✅ All changes merged to `main`
- ✅ CI workflow passes
- ✅ PR titles describe the changes for operators (they become the release notes)
- ✅ Documentation updated
- ✅ Local testing complete
- ✅ Remove `replace` directives from library go.mod files (keep the `apps/*` ones)

### Semantic Versioning

- **Major (v2.0.0)**: Breaking changes, incompatible API changes
- **Minor (v1.1.0)**: New features, backward compatible
- **Patch (v1.0.1)**: Bug fixes, backward compatible

### Version Synchronization

While modules can have independent versions, keep major versions synchronized across related modules for easier dependency management.

### Tag Messages

```bash
# Annotated tag; the release notes come from PR titles, not the tag message
git tag -a v1.4.0 -m v1.4.0
```

---

## Important Notes

1. **Breaking Changes**: If a module introduces breaking changes, all dependent modules must be updated and released with appropriate version bumps.

2. **Testing**: Always test the release process in a separate branch first, especially when updating multiple modules.

3. **Go Proxy Cache**: The Go module proxy may take up to 30 minutes to fully propagate new versions. Be patient and verify availability before proceeding to dependent modules.

4. **Rollback Plan**:
   - **Releases and Docker images**: Cut a new patch version (e.g., v1.4.1); re-running a tag overwrites its images
   - **Go modules**: NEVER delete tags. Create a new patch version instead (e.g., v0.3.1) to avoid Go proxy issues.

5. **Protected Branches**: Create version branches (e.g., `v0`, `v0.3`) for maintaining release history and backporting fixes.
