# Quick Start: Release Guide

This is a quick reference guide for releasing ev-node components.

## Docker Image Release (Recommended for Apps)

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
docker pull ghcr.io/evstack/ev-node-evm-single:v0.2.0
```

### Tag Format
`{app-path}/v{major}.{minor}.{patch}`

**Examples:**
- `evm/single/v0.2.0` → Releases `apps/evm/single/`
- `testapp/v1.0.0` → Releases `apps/testapp/`
- `grpc/single/v2.1.3` → Releases `apps/grpc/single/`

### What Happens Automatically
1. ✅ Validates tag and app directory
2. ✅ Builds multi-platform Docker image (amd64, arm64)
3. ✅ Publishes to GHCR:
   - Version tag: `ghcr.io/evstack/ev-node-{app}:v0.2.0`
   - Latest tag: `ghcr.io/evstack/ev-node-{app}:latest`

### Requirements
- App directory exists: `./apps/{app-path}/`
- Dockerfile exists: `./apps/{app-path}/Dockerfile`
- Tag format: `**/v*.*.*`
- CI passes on main branch

---

## Go Module Release (For Libraries)

### When to Use
Release Go library packages (core, da, sequencers, etc.) for use as dependencies.

### Quick Steps

```bash
# 1. Release in dependency order
# 2. Wait for Go proxy propagation between releases
# 3. Update dependent modules before releasing them

# Example: Release core module
cd core
git tag core/v0.3.0
git push origin core/v0.3.0

# Wait 5-10 minutes for Go proxy
go list -m github.com/evstack/ev-node/core@v0.3.0
```

### Release Order
1. **Phase 1**: `core` (no dependencies)
2. **Phase 2**: `da`, `ev-node`, `execution/evm` (depend on core)
3. **Phase 3**: `sequencers/*` (depend on core + ev-node)
4. **Phase 4**: `apps/*` (depend on all previous)

**See [RELEASE.md](../RELEASE.md#go-module-releases-manual) for complete dependency graph and detailed steps.**

---

## Common Release Scenarios

### Scenario 1: Release Single App (Docker)
```bash
# Tag and push
git tag evm/single/v0.2.0
git push origin evm/single/v0.2.0

# Done! Automated workflow handles the rest
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

# 2. Wait 5-10 min, then first-level deps
git tag da/v0.3.0 && git push origin da/v0.3.0
git tag v0.3.0 && git push origin v0.3.0
git tag execution/evm/v0.3.0 && git push origin execution/evm/v0.3.0

# 3. Wait, then sequencers
git tag sequencers/single/v0.3.0 && git push origin sequencers/single/v0.3.0

# 4. Wait, then apps
git tag apps/evm/single/v0.3.0 && git push origin apps/evm/single/v0.3.0
```

### Scenario 4: Hotfix/Rollback
```bash
# Delete bad tag
git tag -d evm/single/v0.2.0
git push origin :refs/tags/evm/single/v0.2.0

# Fix code, create new tag
git tag evm/single/v0.2.1
git push origin evm/single/v0.2.1
```

---

## Verification

### Docker Image Release
```bash
# Check workflow status
# GitHub → Actions → Release

# Pull and test image
docker pull ghcr.io/evstack/ev-node-evm-single:v0.2.0
docker run ghcr.io/evstack/ev-node-evm-single:v0.2.0 --version

# Check GHCR
# GitHub → Packages → ev-node-evm-single
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

### "App directory does not exist"
- Ensure tag matches app path: `apps/evm/single/` → `evm/single/v0.2.0`
- Check spelling and case sensitivity

### "Dockerfile not found"
- Verify Dockerfile exists at `apps/{app-path}/Dockerfile`
- Check file name is exactly `Dockerfile`

### "Image not found" in tests
- Wait for Docker build workflow to complete
- Check workflow dependencies in Actions tab

### Go proxy delay
- Wait 5-30 minutes for propagation
- Use `go list -m` to verify availability
- Check https://proxy.golang.org/

---

## Best Practices

### Before Releasing

- ✅ All changes merged to `main`
- ✅ CI workflow passes
- ✅ CHANGELOG.md updated
- ✅ Documentation updated
- ✅ Local testing complete

### Semantic Versioning

- **Major (v2.0.0)**: Breaking changes
- **Minor (v1.1.0)**: New features, backward compatible
- **Patch (v1.0.1)**: Bug fixes, backward compatible

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

## Quick Links

- **Workflow Details**: [.github/workflows/README.md](workflows/README.md)
- **Complete Release Process**: [RELEASE.md](../RELEASE.md)
- **CI Workflow**: [.github/workflows/ci.yml](workflows/ci.yml)
- **Release Workflow**: [.github/workflows/release.yml](workflows/release.yml)
- **GitHub Actions**: https://github.com/evstack/ev-node/actions
- **GitHub Packages**: https://github.com/orgs/evstack/packages

---

## Need Help?

1. **Workflow documentation**: See [.github/workflows/README.md](workflows/README.md)
2. **Release process**: See [RELEASE.md](../RELEASE.md)
3. **CI failures**: Check GitHub Actions logs
4. **Questions**: Open an issue or discussion
