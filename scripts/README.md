# Scripts

This directory contains utility scripts for the LiteLLM Operator project.

## sync-helm-chart.sh

A comprehensive script to synchronise the Helm chart with operator-sdk generated resources.

### Overview

The `sync-helm-chart.sh` script automatically copies CRDs and RBAC resources from the operator-sdk generated directories to the Helm chart directories, ensuring they stay in sync. It also handles version updates and provides validation capabilities.

### Features

- **CRD Synchronisation**: Copies Custom Resource Definitions from `config/crd/bases/` to `helm/crds/`
- **RBAC Synchronisation**: Copies RBAC resources from `config/rbac/` to `helm/templates/rbac/`
- **Version Management**: Automatically updates Helm chart version based on project version
- **Backup Creation**: Backups are disabled by default (can be re-enabled if needed)
- **Dry Run Mode**: Shows differences without making changes
- **Validation**: Optionally validates the Helm chart after syncing
- **Cross-platform Support**: Works on both macOS and Linux

### Usage

#### Basic Usage

```bash
# Synchronise everything
./scripts/sync-helm-chart.sh

# Show what would be changed (dry run)
./scripts/sync-helm-chart.sh --dry-run

# Synchronise and validate
./scripts/sync-helm-chart.sh --validate

# Synchronise without updating version
./scripts/sync-helm-chart.sh --no-version
```

#### Using Make Targets

The script is integrated into the Makefile for convenience:

```bash
# Generate manifests and sync Helm chart
make manifests-and-sync

# Sync Helm chart only
make helm-sync

# Show differences without changes
make helm-sync-dry-run

# Sync and validate
make helm-sync-validate
```

### Command Line Options

| Option | Description |
|--------|-------------|
| `-h, --help` | Show help message |
| `-d, --dry-run` | Show differences without making changes |
| `-v, --validate` | Validate the Helm chart after syncing |
| `--no-version` | Skip version update |

### What Gets Synchronised

#### CRDs
- Source: `config/crd/bases/*.yaml`
- Target: `helm/crds/*.yaml`
- Files: All Custom Resource Definition files

#### RBAC Resources
- Source: `config/rbac/*.yaml`
- Target: `helm/templates/rbac/*.yaml`
- Files: All RBAC files except `kustomization.yaml`

#### Version Updates
- Updates `version` and `appVersion` in `helm/Chart.yaml`
- Sources version from `VERSION` file or `Makefile`

### Backup Strategy

Backups are currently disabled in the script. The `backup_file()` function is a no-op that can be re-enabled if needed in the future.

### Error Handling

- Script exits on first error (`set -e`)
- Comprehensive error messages with colour coding
- Validation of required directories and files
- Graceful handling of missing tools (e.g., Helm)

### Integration with CI/CD

The script is designed to be used in CI/CD pipelines:

```yaml
# Example GitHub Actions step
- name: Sync Helm Chart
  run: |
    make manifests-and-sync
```

### Troubleshooting

#### Common Issues

1. **Permission Denied**: Ensure the script is executable
   ```bash
   chmod +x scripts/sync-helm-chart.sh
   ```

2. **Missing Directories**: Ensure you're in the project root directory

3. **Helm Not Found**: Install Helm or skip validation with `--no-validate`

4. **Version Issues**: Check that version is properly set in `VERSION` file or `Makefile`

#### Debug Mode

For debugging, you can run with bash debugging:
```bash
bash -x ./scripts/sync-helm-chart.sh
```

### Contributing

When modifying the script:
1. Test on both macOS and Linux
2. Update this README if adding new features
3. Ensure backward compatibility
4. Add appropriate error handling 