#!/bin/bash

# Script to delete a local and remote tag, then re-tag and push

set -e

# Function to print colored output
print_info() {
    echo "[INFO] $1"
}

print_success() {
    echo "[SUCCESS] $1"
}

print_error() {
    echo "[ERROR] $1"
}

print_usage() {
    echo "Usage: $0 <tag_name>"
    echo "  tag_name: The name of the tag to create/recreate (e.g., v1.0.0)"
    echo ""
    echo "Examples:"
    echo "  $0 v1.0.0"
    echo "  $0 v0.2.1"
    exit 1
}

# Check if tag name is provided
if [ $# -eq 0 ]; then
    print_error "Tag name is required"
    print_usage
fi

TAG_NAME="$1"

# Validate tag name format (basic validation)
if [[ ! "$TAG_NAME" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$ ]]; then
    print_error "Invalid tag format. Expected format: v<major>.<minor>.<patch>[-<prerelease>][+<build>]"
    print_error "Examples: v1.0.0, v0.2.1, v1.0.0-alpha.1, v1.0.0+20231201"
    exit 1
fi

# Check if git is available
if ! command -v git &> /dev/null; then
    print_error "Git is not installed."
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository."
    exit 1
fi

print_info "Processing tag: $TAG_NAME"

# Delete the local tag if it exists
if git show-ref --tags | grep -q "refs/tags/$TAG_NAME"; then
    git tag -d "$TAG_NAME"
    print_info "Deleted local tag $TAG_NAME"
else
    print_info "Local tag $TAG_NAME does not exist"
fi

# Delete the remote tag
if git ls-remote --tags origin | grep -q "refs/tags/$TAG_NAME"; then
    git push --delete origin "$TAG_NAME"
    print_info "Deleted remote tag $TAG_NAME"
else
    print_info "Remote tag $TAG_NAME does not exist"
fi

# Create and push the tag
LATEST_COMMIT=$(git rev-parse HEAD)
git tag "$TAG_NAME" "$LATEST_COMMIT"
git push origin "$TAG_NAME"

print_success "Successfully created and pushed tag $TAG_NAME"
print_info "Tag points to commit: $LATEST_COMMIT"
