#!/bin/bash

# Script to delete completed and old GitHub workflow runs
# This script will delete workflow runs that are:
# - Completed (success, failure, cancelled, skipped)
# - Older than a specified number of days (default: 30 days)

set -e

# Configuration
DAYS_OLD=${1:-0}  # Default to 0 days (from now) if no argument provided
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
if [ "$DAYS_OLD" -eq 0 ]; then
    print_info "Deleting ALL completed workflow runs (from now)"
else
    print_info "Deleting workflow runs older than or equal to $DAYS_OLD days (including today)"
fi

# Calculate the date threshold (including today)
if [ "$DAYS_OLD" -eq 0 ]; then
    # If DAYS_OLD is 0, we want all runs (from now), so set a very old date
    DATE_THRESHOLD="1970-01-01"
    print_info "Date threshold: ALL runs (from now)"
else
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        DATE_THRESHOLD=$(date -v-${DAYS_OLD}d +%Y-%m-%d)
    else
        # Linux
        DATE_THRESHOLD=$(date -d "$DAYS_OLD days ago" +%Y-%m-%d)
    fi
    print_info "Date threshold: $DATE_THRESHOLD (including today)"
fi

# Get completed workflow runs older than or equal to the threshold (including today)
print_info "Fetching workflow runs..."

# First, let's see what workflow runs exist
print_info "Checking available workflow runs..."
TOTAL_RUNS=$(gh run list --limit 10 --json databaseId,status,conclusion,createdAt,workflowName,headBranch --jq "length")
print_info "Total workflow runs found: $TOTAL_RUNS"

COMPLETED_RUNS=$(gh run list --status completed --limit 10 --json databaseId,status,conclusion,createdAt,workflowName,headBranch --jq "length")
print_info "Completed workflow runs found: $COMPLETED_RUNS"

# Get workflow runs that are completed and older than or equal to threshold (including today)
if [ "$DAYS_OLD" -eq 0 ]; then
    # Get all completed workflow runs
    WORKFLOW_RUNS=$(gh run list \
        --status completed \
        --limit 1000 \
        --json databaseId,status,conclusion,createdAt,workflowName,headBranch \
        --jq ".[] | .databaseId")
else
    # Get workflow runs that are completed and older than or equal to threshold
    WORKFLOW_RUNS=$(gh run list \
        --status completed \
        --limit 1000 \
        --json databaseId,status,conclusion,createdAt,workflowName,headBranch \
        --jq ".[] | select(.createdAt <= \"${DATE_THRESHOLD}T23:59:59Z\") | .databaseId")
fi

if [ -z "$WORKFLOW_RUNS" ]; then
    if [ "$DAYS_OLD" -eq 0 ]; then
        print_info "No completed workflow runs found."
    else
        print_info "No completed workflow runs older than or equal to $DAYS_OLD days found."
    fi
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
FAILED_RUNS=()
CURRENT_RUN=0

for RUN_ID in $WORKFLOW_RUNS; do
    ((CURRENT_RUN++))
    print_info "Deleting workflow run $RUN_ID ($CURRENT_RUN/$RUN_COUNT)..."
    
    # Capture error output for logging with timeout
    ERROR_OUTPUT=$(timeout 30s gh run delete "$RUN_ID"  2>&1)
    DELETE_EXIT_CODE=$?
    
    if [ $DELETE_EXIT_CODE -eq 0 ]; then
        ((DELETED_COUNT++))
        echo -n "."
    elif [ $DELETE_EXIT_CODE -eq 124 ]; then
        ((FAILED_COUNT++))
        echo -n "T"
        FAILED_RUNS+=("$RUN_ID: Timeout (30s) - command took too long")
    else
        ((FAILED_COUNT++))
        echo -n "x"
        FAILED_RUNS+=("$RUN_ID: $ERROR_OUTPUT")
    fi
done

echo ""

# Display failed deletions with error details
if [ $FAILED_COUNT -gt 0 ]; then
    print_error "Failed to delete $FAILED_COUNT workflow runs:"
    for failed_run in "${FAILED_RUNS[@]}"; do
        print_error "  - $failed_run"
    done
fi

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
