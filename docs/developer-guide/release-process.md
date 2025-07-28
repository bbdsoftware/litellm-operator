# Release Process

This document describes the release process for the LiteLLM Operator, including how to create releases, what happens during the release workflow, and how to handle the automated updates.

## Overview

The LiteLLM Operator uses a fully automated release process that:

1. **Builds and publishes** the operator image and artifacts
2. **Creates a GitHub release** with release notes
3. **Automatically updates** manifests and Helm charts
4. **Creates a pull request** for review and merging

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

### Step 3: Monitor the Release Workflow

The release workflow will automatically:

1. **Validate the semantic version** tag
2. **Build and publish** the operator image to GitHub Container Registry
3. **Create a GitHub release** with assets
4. **Update manifests and Helm charts** (for stable releases only)
5. **Create a pull request** with the updates

## Release Workflow Details

### What Gets Built and Published

The release workflow creates and publishes:

- **Docker Image**: `ghcr.io/bbdsoftware/litellm-operator:{version}`
- **Helm Chart**: Updated with new version and appVersion
- **CRDs**: Generated and included in the release
- **Binary Assets**: For multiple platforms (Linux, macOS, Windows)

### Automated Manifest Updates

For stable releases (not pre-releases), the workflow automatically updates:

1. **Operator Image Tag**: `config/manager/kustomization.yaml`
2. **Helm Chart Version**: `helm/Chart.yaml`
3. **Helm Chart AppVersion**: `helm/Chart.yaml`
4. **Default Image Tag**: `helm/values.yaml`

### Pull Request Creation

The workflow creates a pull request with:

- **Branch**: `update-manifests-{version}`
- **Title**: `chore: update operator image and helm chart to {version}`
- **Description**: Detailed list of changes made
- **Labels**: `automated`, `release`

## Handling the Release Pull Request

### Step 1: Review the Changes

When the release workflow completes, you'll see a new pull request. Review the changes to ensure:

- [ ] Image tags are correctly updated
- [ ] Helm chart versions are appropriate
- [ ] No unintended changes were made

### Step 2: Merge the Pull Request

Once reviewed, merge the pull request:

```bash
# Option 1: Merge via GitHub UI
# Click "Merge pull request" in the GitHub interface

# Option 2: Merge via command line
git fetch origin
git checkout main
git merge origin/update-manifests-v1.0.0
git push origin main
```

### Step 3: Verify the Release

After merging, verify the release:

1. **Check the GitHub release** page for all assets
2. **Verify the Docker image** is available in the registry
3. **Test the Helm chart** installation
4. **Confirm manifests** are updated in main branch

## Release Types

### Stable Releases

Stable releases (e.g., `v1.0.0`) trigger the full workflow including:
- ✅ Image building and publishing
- ✅ GitHub release creation
- ✅ Manifest updates
- ✅ Pull request creation

### Pre-releases

Pre-releases (e.g., `v1.0.0-rc.1`, `v1.0.0-beta.1`, `v1.0.0-alpha.1`) skip manifest updates:
- ✅ Image building and publishing
- ✅ GitHub release creation
- ❌ Manifest updates (skipped)
- ❌ Pull request creation (skipped)

## Troubleshooting

### Common Issues

#### Workflow Fails on Tag Push

**Problem**: Release workflow fails immediately after pushing a tag.

**Solutions**:
1. Check that the tag follows semantic versioning: `v1.0.0`
2. Ensure the tag doesn't already exist
3. Verify GitHub Actions are enabled for the repository

#### Pull Request Not Created

**Problem**: Release completes but no pull request is created.

**Solutions**:
1. Check if it's a pre-release (rc, beta, alpha tags skip PR creation)
2. Verify the workflow has proper permissions
3. Check workflow logs for errors

#### Manifest Updates Fail

**Problem**: Pull request is created but manifest updates are incorrect.

**Solutions**:
1. Review the workflow logs for sed/kustomize errors
2. Check that the version extraction is working correctly
3. Verify the file paths in the workflow

### Rollback Process

If a release needs to be rolled back:

1. **Delete the release tag**:
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

2. **Delete the GitHub release** (via GitHub UI)

3. **Revert the manifest changes** if the PR was merged:
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

### Release Notes

- Update `CHANGELOG.md` before creating a release
- Include all significant changes
- Reference issues and pull requests
- Provide migration notes for breaking changes

### Testing

- Test the release process in a staging environment
- Verify the operator works with the new image
- Test Helm chart installation and upgrades
- Validate CRD compatibility

### Communication

- Announce releases in your team communication channels
- Update documentation if needed
- Notify users of breaking changes well in advance

## Configuration

### Workflow Configuration

The release workflow is configured in `.github/workflows/release.yml` and includes:

- **GoReleaser configuration**: `.goreleaser.yml`
- **Docker build settings**: Multi-platform builds
- **Helm chart updates**: Automatic version bumping
- **Pull request creation**: Automated manifest updates

### Customization

To customize the release process:

1. **Modify `.goreleaser.yml`** for build configuration
2. **Update the workflow** for different release steps
3. **Adjust Helm chart settings** in `helm/Chart.yaml`
4. **Modify manifest paths** in the workflow

## Support

If you encounter issues with the release process:

1. Check the [GitHub Actions logs](https://github.com/bbdsoftware/litellm-operator/actions)
2. Review the [GoReleaser documentation](https://goreleaser.com/)
3. Consult the [GitHub Actions documentation](https://docs.github.com/en/actions)
4. Open an issue in the repository for persistent problems 