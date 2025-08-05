#!/bin/bash

# Script to delete a local and remote tag, then re-tag and push

TAG_NAME="v0.0.1"

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

# Check if git is available
if ! command -v git &> /dev/null; then
    print_error "Git is not installed."
    exit 1
fi

# Delete the local tag if it exists
if git show-ref --tags | grep -q "refs/tags/$TAG_NAME"; then
    git tag -d "$TAG_NAME"
    print_info "Deleted local tag $TAG_NAME"
else
    print_info "Local tag $TAG_NAME does not exist"
fi

# Delete the remote tag
git push --delete origin "$TAG_NAME" || print_info "Remote tag $TAG_NAME does not exist"

# Create and push the tag
LATEST_COMMIT=$(git rev-parse HEAD)
git tag "$TAG_NAME" "$LATEST_COMMIT"
git push origin "$TAG_NAME"

print_success "Re-tagged and pushed $TAG_NAME"
