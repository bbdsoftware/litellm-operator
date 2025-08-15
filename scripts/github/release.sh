#!/bin/bash

# Script to checkout main branch, pull latest changes, create a tag and push it

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

# Check if tag name is provided as argument
if [ $# -eq 0 ]; then
    print_error "Usage: $0 <tag_name>"
    print_error "Example: $0 v1.0.0"
    exit 1
fi

TAG_NAME="$1"

# Validate tag name format (basic validation)
if [[ ! "$TAG_NAME" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    print_error "Tag name should follow semantic versioning format (e.g., v1.0.0)"
    exit 1
fi

print_info "Starting release process for tag: $TAG_NAME"

# Checkout main branch
print_info "Checking out main branch..."
git checkout main

# Pull latest changes
print_info "Pulling latest changes from remote..."
git pull origin main

# Check if tag already exists locally
if git show-ref --tags | grep -q "refs/tags/$TAG_NAME"; then
    print_error "Tag $TAG_NAME already exists locally. Please use a different tag name."
    exit 1
fi

# Check if tag already exists remotely
if git ls-remote --tags origin | grep -q "refs/tags/$TAG_NAME"; then
    print_error "Tag $TAG_NAME already exists remotely. Please use a different tag name."
    exit 1
fi

# Create the tag
print_info "Creating tag $TAG_NAME..."
LATEST_COMMIT=$(git rev-parse HEAD)
git tag "$TAG_NAME" "$LATEST_COMMIT"

# Push the tag
print_info "Pushing tag $TAG_NAME to remote..."
git push origin "$TAG_NAME"

print_success "Successfully created and pushed tag $TAG_NAME"
print_info "Tag $TAG_NAME points to commit: $LATEST_COMMIT"
