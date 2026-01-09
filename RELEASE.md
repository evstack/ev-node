# ev-node Release Guide

This document covers the release process for ev-node components:

- **Docker Image Releases** - Automated via GitHub workflows (for deployable applications)
- **Go Module Releases** - Manual process for library packages and dependencies

---

## Docker Image Releases (Automated)

### When to Use

Release deployable applications (EVM nodes, test apps, etc.) as Docker images.

### Quick Steps

```bash
# 1. Ensure CI passes on main
# 2. Create and push tag
git tag evm/single/v0.2.0
git push origin evm/single/v0.2.0

# 3. Monitor workflow
# GitHub → Actions → Release workflow

# 4. Verify release
docker pull ghcr.io/evstack/ev-node-evm:v0.2.0
```

### Tag Format

Use the hierarchical tag format: `{app-path}/v{major}.{minor}.{patch}`

**Examples:**

- `apps/evm/v0.2.0` → Releases `apps/evm/`
- `apps/testapp/v1.0.0` → Releases `apps/testapp/`
- `apps/grpc/v2.1.3` → Releases `apps/grpc/`

### Automated Process

When you push a tag, the release workflow automatically:

1. ✅ Validates tag format and app directory structure
2. ✅ Builds multi-platform Docker image (amd64, arm64)
3. ✅ Publishes to GitHub Container Registry (GHCR):
   - Version tag: `ghcr.io/evstack/ev-node-{app}:v0.2.0`
   - Latest tag: `ghcr.io/evstack/ev-node-{app}:latest`

### Requirements

- App directory must exist at `./apps/{app-path}/`
- Dockerfile must exist at `./apps/{app-path}/Dockerfile`
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

1. **github.com/evstack/ev-node** - Path: `./` (root)
2. **github.com/evstack/ev-node/execution/evm** - Path: `./execution/evm`

#### Phase 3: Application Packages

These packages have the most dependencies and should be released last:

- **github.com/evstack/ev-node/apps/evm** - Path: `./apps/evm`

### Release Process

**IMPORTANT**: Each module must be fully released and available on the Go proxy before updating dependencies in dependent modules.

**Before Starting:**

- Create a protected version branch (e.g., `v0` for major versions, `v0.3` for minor breaking changes)
- Ensure CHANGELOG.md is up to date with all changes properly categorized
- Remove all `replace` directives from go.mod files

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

### Scenario 1: Release Single App (Docker Only)

```bash
# Tag and push - automation handles the rest
git tag evm/v0.2.0
git push origin evm/v0.2.0
```

### Scenario 2: Release Multiple Apps

```bash
# Release apps independently
git tag evm/single/v0.2.0
git tag testapp/v1.0.0
git push origin evm/single/v0.2.0 testapp/v1.0.0

# Each triggers its own workflow
```

### Scenario 3: Full Go Module Release

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

### Scenario 4: Hotfix/Patch Release

```bash
# For Docker images - delete and recreate
git tag -d evm/single/v0.2.0
git push origin :refs/tags/evm/single/v0.2.0

# Fix code, create new tag
git tag evm/single/v0.2.1
git push origin evm/single/v0.2.1

# For Go modules - create new patch version
# Do NOT delete Go module tags - create v0.3.1 instead
```

---

## Verification

### Docker Image Release

```bash
# Check workflow status
# GitHub → Actions → Release

# Pull and test image
docker pull ghcr.io/evstack/ev-node-evm:v0.2.0
docker run ghcr.io/evstack/ev-node-evm:v0.2.0 --version

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

### Docker Releases

**"App directory does not exist"**

- Ensure tag matches app path: `apps/evm/` → `apps/evm/v0.2.0`
- Check spelling and case sensitivity

**"Dockerfile not found"**

- Verify Dockerfile exists at `apps/{app-path}/Dockerfile`
- Check filename is exactly `Dockerfile`

**"Image not found" in tests**

- Wait for Docker build workflow to complete
- Check workflow dependencies in Actions tab

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
- ✅ CHANGELOG.md updated
- ✅ Documentation updated
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
git tag -a evm/single/v0.2.0 -m "Release EVM single v0.2.0

Features:
- Added feature X
- Improved performance Y

Bug fixes:
- Fixed issue Z
"

# Avoid: Lightweight tag without description
git tag evm/single/v0.2.0  # Less informative
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
