# Release Process

This document describes the release process for the LiteLLM Operator, including how to create releases, what happens during the release workflow, and how to handle the automated updates.

## Overview

The LiteLLM Operator uses a fully automated release process that:

1. **Runs tests and validation** to ensure code quality
2. **Builds and publishes** the operator image and artifacts using GoReleaser
3. **Creates a GitHub release** with release notes and assets
4. **Automatically updates** manifests and Helm charts (for stable releases only)
5. **Publishes Helm charts** to the OCI registry

## Prerequisites

Before creating a release, ensure you have:

- [ ] All tests passing
- [ ] Documentation updated
- [ ] CHANGELOG.md updated with release notes
- [ ] Proper versioning strategy in place

## Creating a Release

### Step 1: Prepare for Release

1. **Update the CHANGELOG.md** with release notes for the new version
2. **Ensure all changes are committed** to the main branch
3. **Verify the current state** of your repository

### Step 2: Create and Push a Release Tag

The release process is triggered by pushing a semantic version tag:

```bash
# Create a new release tag
git tag v1.0.0

# Push the tag to trigger the release workflow
git push origin v1.0.0
```

**Supported tag formats:**
- `v1.0.0` - Major release
- `v1.0.1` - Patch release  
- `v1.1.0` - Minor release
- `v1.0.0-rc.1` - Release candidate (will skip manifest updates)
- `v1.0.0-alpha.1` - Alpha release (will skip manifest updates)
- `v1.0.0-beta.1` - Beta release (will skip manifest updates)

**Important**: The workflow is configured to ignore pushes to certain paths, so ensure your changes are not in:
- Documentation files (`**/*.md`, `docs/**`, `README.md`, etc.)
- Helm chart files (`deploy/charts/litellm-operator/**`, `helm/**`)

### Step 3: Monitor the Release Workflow

The release workflow will automatically:

1. **Run tests and linting** to validate code quality
2. **Validate the semantic version** tag
3. **Build and publish** the operator image to GitHub Container Registry
4. **Create a GitHub release** with assets and release notes
5. **Update manifests and Helm charts** (for stable releases only)
6. **Publish Helm chart** to OCI registry

## Release Workflow Details

### What Gets Built and Published

The release workflow creates and publishes:

- **Docker Images**: 
  - `ghcr.io/bbdsoftware/litellm-operator:{version}` (multi-arch: amd64, arm64)
  - `ghcr.io/bbdsoftware/litellm-operator:latest` (multi-arch: amd64, arm64)
  - For pre-releases: additional `{major}.{minor}-prerelease` tags
- **Binary Assets**: For Linux amd64 and arm64 platforms
- **Source Archives**: Complete source code archives
- **Helm Chart**: Published to `ghcr.io/bbdsoftware/charts/litellm-operator`

### Automated Manifest Updates

For stable releases (not pre-releases), the workflow automatically updates:

1. **Operator Image Tag**: `config/manager/kustomization.yaml`
2. **Helm Chart Version**: `deploy/charts/litellm-operator/Chart.yaml`
3. **Helm Chart AppVersion**: `deploy/charts/litellm-operator/Chart.yaml`

The workflow uses a GitHub App for authentication to commit changes directly to the main branch.

## Release Types

### Stable Releases

Stable releases (e.g., `v1.0.0`) trigger the full workflow including:
- ✅ Tests and validation
- ✅ Image building and publishing
- ✅ GitHub release creation
- ✅ Manifest updates
- ✅ Helm chart publication

### Pre-releases

Pre-releases (e.g., `v1.0.0-rc.1`, `v1.0.0-beta.1`, `v1.0.0-alpha.1`) skip manifest updates:
- ✅ Tests and validation
- ✅ Image building and publishing
- ✅ GitHub release creation (marked as pre-release)
- ❌ Manifest updates (skipped)
- ✅ Helm chart publication

## Release Workflow Jobs

### 1. Run Tests (`run-tests`)
- Runs unit tests with `make test`
- Performs linting with golangci-lint
- Ensures code quality before release
- Uses Ubuntu latest runner

### 2. Build and Release (`build-and-release`)
- Validates semantic versioning using `matt-usurp/validate-semver@v2`
- Determines release type (stable vs pre-release)
- Builds multi-architecture Docker images using Docker Buildx
- Generates CRDs and combines them into `dist/crds.yaml`
- Runs GoReleaser to create GitHub release
- Outputs release type information for downstream jobs
- Uses different GoReleaser configs for stable vs pre-releases

### 3. Update Manifests (`update-manifests`)
- **Only runs for stable releases**
- Uses GitHub App authentication for secure access
- Updates operator image tags in kustomization files
- Runs `make helm-gen` to regenerate Helm chart from Kustomize
- Updates Helm chart version and appVersion
- Commits changes directly to main branch using `EndBug/add-and-commit@v9`
- Publishes Helm chart to OCI registry using `appany/helm-oci-chart-releaser@v0.5.0`

## GoReleaser Configuration

The project uses different GoReleaser configurations:

- **`.goreleaser.yml`**: For stable releases
- **`.goreleaser.prerelease.yml`**: For pre-releases (alpha, beta, rc)

Key differences in pre-release configuration:
- Additional pre-release specific Docker tags
- Pre-release flag set to true
- Different release notes template
- Additional build labels

## Installation Instructions

### Using Helm (Recommended)

```bash
# Authenticate with GitHub Container Registry
helm registry login ghcr.io -u YOUR_GITHUB_USERNAME -p YOUR_GITHUB_TOKEN

# Install the operator from OCI registry
helm install litellm-operator oci://ghcr.io/bbdsoftware/charts/litellm-operator --version v1.0.0
```

### Using kubectl

```bash
# Install CRDs using kustomize
kubectl apply -k https://github.com/bbdsoftware/litellm-operator/config/crd

# Install the operator using kustomize
kubectl apply -k https://github.com/bbdsoftware/litellm-operator/config/default
```

## Troubleshooting

### Common Issues

#### Workflow Fails on Tag Push

**Problem**: Release workflow fails immediately after pushing a tag.

**Solutions**:
1. Check that the tag follows semantic versioning: `v1.0.0`
2. Ensure the tag doesn't already exist
3. Verify GitHub Actions are enabled for the repository
4. Check that the tag doesn't match path-ignore patterns (docs, README, etc.)
5. Verify the workflow has proper permissions (contents, packages, id-token, pull-requests)

#### Manifest Updates Fail

**Problem**: Release completes but manifest updates are incorrect.

**Solutions**:
1. Check GitHub App authentication is properly configured
2. Verify the workflow has proper permissions for the GitHub App
3. Review the workflow logs for sed/kustomize errors
4. Check that the version extraction is working correctly
5. Verify the file paths in the workflow
6. Check that `make helm-gen` completed successfully

#### Helm Chart Not Published

**Problem**: Release completes but Helm chart is not available in OCI registry.

**Solutions**:
1. Check the helm-publish workflow logs
2. Verify GitHub Container Registry permissions
3. Ensure the chart version is correctly extracted from Chart.yaml
4. Check that the OCI registry path is correct (`ghcr.io/bbdsoftware/charts/`)

#### Tests Fail

**Problem**: Release workflow fails during the test phase.

**Solutions**:
1. Run `make test` locally to reproduce the issue
2. Check for linting errors with golangci-lint
3. Ensure all dependencies are properly installed
4. Verify Go version compatibility (workflow uses Go 1.21)

### Rollback Process

If a release needs to be rolled back:

1. **Delete the release tag**:
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

2. **Delete the GitHub release** (via GitHub UI)

3. **Revert the manifest changes** if they were committed:
   ```bash
   git revert <commit-hash>
   git push origin main
   ```

4. **Create a new release** with the correct version

## Best Practices

### Versioning Strategy

- Use semantic versioning: `MAJOR.MINOR.PATCH`
- Increment patch for bug fixes
- Increment minor for new features
- Increment major for breaking changes
- Use pre-release suffixes for testing: `-alpha.1`, `-beta.1`, `-rc.1`

### Release Notes

- Update `CHANGELOG.md` before creating a release
- Include all significant changes
- Reference issues and pull requests
- Provide migration notes for breaking changes
- GoReleaser will automatically generate release notes from commits

### Testing

- Test the release process in a staging environment
- Verify the operator works with the new image
- Test Helm chart installation and upgrades
- Validate CRD compatibility
- Test pre-release versions before stable releases

### Communication

- Announce releases in your team communication channels
- Update documentation if needed
- Notify users of breaking changes well in advance
- Use pre-releases for major changes to gather feedback

## Configuration

### Workflow Configuration

The release workflow is configured in `.github/workflows/release.yml` and includes:

- **Permissions**: `contents: write`, `packages: write`, `id-token: write`, `pull-requests: write`
- **GoReleaser configuration**: `.goreleaser.yml` and `.goreleaser.prerelease.yml`
- **Docker build settings**: Multi-platform builds (amd64, arm64) using Docker Buildx
- **Helm chart updates**: Automatic version bumping and regeneration
- **GitHub App authentication**: For secure manifest updates

### Helm Chart Generation

The Helm chart is automatically generated from Kustomize output using:
- `make helm-gen`: Regenerates the Helm chart from Kustomize manifests
- Chart is published to OCI registry automatically using `appany/helm-oci-chart-releaser@v0.5.0`

### GitHub App Setup

The workflow uses a GitHub App for authentication to commit changes. Ensure the following secrets are configured:

- `LITELLM_OPERATOR_BOT_APP_ID`: The GitHub App ID
- `LITELLM_OPERATOR_BOT_PRIVATE_KEY`: The GitHub App private key

### Customization

To customize the release process:

1. **Modify `.goreleaser.yml`** for build configuration
2. **Update the workflow** for different release steps
3. **Adjust Helm chart settings** in `deploy/charts/litellm-operator/Chart.yaml`
4. **Modify manifest paths** in the workflow
5. **Update Helm chart generation** in the Makefile

## Support

If you encounter issues with the release process:

1. Check the [GitHub Actions logs](https://github.com/bbdsoftware/litellm-operator/actions)
2. Review the [GoReleaser documentation](https://goreleaser.com/)
3. Consult the [GitHub Actions documentation](https://docs.github.com/en/actions)
4. Check the [Helm OCI documentation](https://helm.sh/docs/topics/registries/)
5. Open an issue in the repository for persistent problems 