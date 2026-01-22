# GitHub Workflows Documentation

Comprehensive guide to ev-node CI/CD workflows, orchestration, and tag-based release process.

## Table of Contents

- [Workflow Architecture](#workflow-architecture)
- [CI Workflow](#ci-workflow)
- [Release Workflow](#release-workflow)
- [Tag-Based Release Process](#tag-based-release-process)
- [Troubleshooting](#troubleshooting)

## Workflow Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Trigger Events                          │
│  • push to main  • pull_request  • merge_group  • tags       │
└───────────────────────────┬──────────────────────────────────┘
                            │
        ┌───────────────────┴────────────────────┐
        │                                        │
        v                                        v
┌───────────────────┐                  ┌────────────────────┐
│  CI Workflow      │                  │ Release Workflow   │
│  (ci.yml)         │                  │ (release.yml)      │
│                   │                  │                    │
│  1. Image Tag     │                  │  Trigger: Tag      │
│     Generation    │                  │  **/v*.*.*         │
│     • PR: pr-#    │                  │                    │
│     • Branch: ref │                  │  1. Parse Tag      │
│                   │                  │     • Validate     │
│  2. Parallel:     │                  │     • Extract      │
│     • Lint        │                  │                    │
│     • Docker      │                  │  2. Build & Push   │
│     • Tests       │                  │     • Multi-arch   │
│     • Proto       │                  │     • Version tag  │
│                   │                  │     • Latest tag   │
│  3. Sequential:   │                  │                    │
│     • Docker E2E  │                  │                    │
└───────────────────┘                  └────────────────────┘
```

## CI Workflow

**Entry Point:** `ci.yml` - Orchestrates all CI processes

**Triggers:** `push` to main | `pull_request` | `merge_group`

### Image Tag Generation

First job generates consistent Docker tags for all downstream jobs:
- **PRs:** `pr-{number}` (e.g., `pr-123`)
- **Branches:** Sanitized ref name (e.g., `main`)

### Parallel Jobs

**Lint** (`lint.yml`) - Code quality checks:
- golangci-lint, hadolint, yamllint, markdown-lint, goreleaser-check

**Docker** (`docker.yml`) - Image builds:
- Multi-platform (linux/amd64, linux/arm64)
- Pushes to GHCR with generated tag
- Skips for merge groups (uses PR cache)

**Tests** (`test.yml`) - Comprehensive testing:
- Build all binaries, Go mod tidy check
- Unit tests, Integration tests, E2E tests, EVM tests
- Combined coverage upload to Codecov

**Proto** (`proto.yml`) - Protobuf validation using Buf

### Sequential Jobs

**Docker E2E Tests** (`docker-tests.yml`):
- Waits for Docker build completion
- Runs E2E and upgrade tests using built images
- Can be manually triggered via `workflow_dispatch`

### Dependencies

```
determine-image-tag
    ├─→ lint
    ├─→ docker ──→ docker-tests
    ├─→ test
    └─→ proto
```

## Release Workflow

**Entry Point:** `release.yml` - Automated Docker image releases

**Triggers:** Tag push matching `**/v*.*.*` (e.g., `evm/single/v0.2.0`)

### Parse Release Tag

Extracts and validates:
```yaml
Input:  evm/single/v0.2.0
Output:
  app-path: evm/single
  version: v0.2.0
  image-name: ev-node-evm-single
  dockerfile: apps/evm/single/Dockerfile
```

**Validation:**
- ✅ App directory exists: `./apps/{app-path}`
- ✅ Dockerfile exists: `apps/{app-path}/Dockerfile`
- ✅ Tag matches: `**/v*.*.*`
- ✅ Semantic versioning format

### Build and Push

- Multi-platform build (amd64, arm64)
- Publishes to GHCR:
  - `ghcr.io/{owner}/ev-node-{app}:v0.2.0` (version tag)
  - `ghcr.io/{owner}/ev-node-{app}:latest` (latest tag)

## Tag-Based Release Process

### Tag Format

**Pattern:** `{app-path}/v{major}.{minor}.{patch}`

Maps directly to `./apps/` directory structure:

| Tag | App Path | Version | Image Name |
|-----|----------|---------|------------|
| `evm/single/v0.2.0` | `evm/single` | `v0.2.0` | `ev-node-evm-single` |
| `testapp/v1.0.0` | `testapp` | `v1.0.0` | `ev-node-testapp` |
| `grpc/single/v2.1.3` | `grpc/single` | `v2.1.3` | `ev-node-grpc-single` |

### Release Steps

#### 1. Pre-Release Checklist
- [ ] Changes committed and pushed to `main`
- [ ] CI passes
- [ ] CHANGELOG.md updated
- [ ] Dockerfile exists in app directory

#### 2. Create Tag

```bash
# Annotated tag with release notes
git tag -a evm/single/v0.2.0 -m "Release EVM single v0.2.0

Features:
- Added feature X
- Improved performance Y

Bug fixes:
- Fixed issue Z
"

git push origin evm/single/v0.2.0
```

#### 3. Automated Process

Workflow automatically:
1. Validates tag and app structure
2. Builds multi-platform images
3. Publishes to GHCR with version + latest tags

#### 4. Verify

```bash
docker pull ghcr.io/evstack/ev-node-evm-single:v0.2.0
docker run ghcr.io/evstack/ev-node-evm-single:v0.2.0 --version
```

### Multiple Releases

```bash
# Individual tags
git tag evm/single/v0.2.0 && git push origin evm/single/v0.2.0
git tag testapp/v1.0.0 && git push origin testapp/v1.0.0

# Multiple at once
git push origin evm/single/v0.2.0 testapp/v1.0.0
```

### Semantic Versioning

- **MAJOR (v2.0.0):** Breaking changes
- **MINOR (v1.1.0):** New features, backward compatible
- **PATCH (v1.0.1):** Bug fixes, backward compatible

### Rollback

```bash
# Delete tag
git tag -d evm/single/v0.2.0
git push origin :refs/tags/evm/single/v0.2.0

# Fix and recreate
git tag -a evm/single/v0.2.0 -m "Release message"
git push origin evm/single/v0.2.0
```

### Best Practices

1. Use annotated tags with descriptive messages
2. Update CHANGELOG.md before tagging
3. Ensure CI passes on main
4. Test locally before release
5. Document breaking changes
6. Version tags are immutable references
7. Latest tag points to most recent release

### Adding New Apps

1. Create directory: `apps/new-app/Dockerfile`
2. Add to CI yaml (if needed):
   ```yaml
   apps: [{"name": "ev-node-new-app", "dockerfile": "apps/new-app/Dockerfile"}]
   ```
3. Create tag: `git tag new-app/v1.0.0 && git push origin new-app/v1.0.0`

## Environment & Secrets

### Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `EV_NODE_IMAGE_REPO` | GHCR repository | `ghcr.io/evstack` |
| `EV_NODE_IMAGE_TAG` | Image tag | `pr-123`, `main` |

### Secrets

| Secret | Description | Used By |
|--------|-------------|---------|
| `GITHUB_TOKEN` | GHCR authentication | Docker workflows |
| `CODECOV_TOKEN` | Coverage upload | Test workflow |

## Troubleshooting

### Release Errors

**"App directory does not exist"**
- Tag must match app path: `apps/evm/single/` → `evm/single/v0.2.0`

**"Dockerfile not found"**
- Verify `apps/{app-path}/Dockerfile` exists

**"Invalid tag format"**
- Use `evm/single/v0.2.0` not `evm/single/0.2.0`

### CI Errors

**Docker E2E "image not found"**
- Wait for Docker build job completion

**Coverage upload fails**
- Verify `CODECOV_TOKEN` in repository settings

**Lint failures**
- Run locally: `make lint`

### Manual Triggers

**Docker E2E tests:**
```
GitHub → Actions → "Docker Tests" → Run workflow
Enter tag: pr-123, main, v0.2.0
```

**Check images:**
```bash
gh api /orgs/evstack/packages/container/ev-node-evm-single/versions
docker pull ghcr.io/evstack/ev-node-evm-single:v0.2.0
```

## Additional Resources

- **Complete Release Guide:** [RELEASE.md](../RELEASE.md)
- **Quick Start:** [RELEASE_QUICK_START.md](../RELEASE_QUICK_START.md)
- **GitHub Actions:** https://docs.github.com/en/actions
- **Semantic Versioning:** https://semver.org/
- **GHCR Docs:** https://docs.github.com/en/packages
