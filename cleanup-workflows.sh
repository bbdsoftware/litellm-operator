#!/bin/bash

# Script to delete completed and old GitHub workflow runs
# This script will delete workflow runs that are:
# - Completed (success, failure, cancelled, skipped)
# - Older than a specified number of days (default: 30 days)

set -e

# Configuration
DAYS_OLD=${1:-30}  # Default to 30 days if no argument provided
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || echo "")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    print_error "GitHub CLI (gh) is not installed. Please install it first."
    exit 1
fi

# Check if we're in a git repository
if [ -z "$REPO" ]; then
    print_error "Not in a GitHub repository or unable to detect repository."
    exit 1
fi

print_info "Repository: $REPO"
print_info "Deleting workflow runs older than $DAYS_OLD days"

# Calculate the date threshold
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    DATE_THRESHOLD=$(date -v-${DAYS_OLD}d +%Y-%m-%d)
else
    # Linux
    DATE_THRESHOLD=$(date -d "$DAYS_OLD days ago" +%Y-%m-%d)
fi

print_info "Date threshold: $DATE_THRESHOLD"

# Get completed workflow runs older than the threshold
print_info "Fetching workflow runs..."

# Get workflow runs that are completed and older than threshold
WORKFLOW_RUNS=$(gh run list \
    --status completed \
    --limit 1000 \
    --json databaseId,status,conclusion,createdAt,workflowName,headBranch \
    --jq ".[] | select(.createdAt < \"${DATE_THRESHOLD}T00:00:00Z\") | .databaseId")

if [ -z "$WORKFLOW_RUNS" ]; then
    print_info "No completed workflow runs older than $DAYS_OLD days found."
    exit 0
fi

# Count the runs to be deleted
RUN_COUNT=$(echo "$WORKFLOW_RUNS" | wc -l | tr -d ' ')
print_warning "Found $RUN_COUNT workflow runs to delete."

# Ask for confirmation
echo -n "Do you want to proceed with deletion? (y/N): "
read -r CONFIRM

if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
    print_info "Deletion cancelled."
    exit 0
fi

print_info "Starting deletion process..."

# Delete workflow runs
DELETED_COUNT=0
FAILED_COUNT=0

for RUN_ID in $WORKFLOW_RUNS; do
    if gh run delete "$RUN_ID" --confirm 2>/dev/null; then
        ((DELETED_COUNT++))
        echo -n "."
    else
        ((FAILED_COUNT++))
        echo -n "x"
    fi
done

echo ""
print_success "Deleted $DELETED_COUNT workflow runs"

if [ $FAILED_COUNT -gt 0 ]; then
    print_warning "Failed to delete $FAILED_COUNT workflow runs"
fi

print_info "Cleanup completed!"

# Optional: Show remaining workflow runs
echo ""
echo -n "Show remaining workflow runs? (y/N): "
read -r SHOW_REMAINING

if [[ "$SHOW_REMAINING" =~ ^[Yy]$ ]]; then
    print_info "Remaining workflow runs:"
    gh run list --limit 10
fi
