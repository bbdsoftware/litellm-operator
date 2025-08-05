#!/bin/bash

# sync-helm-chart.sh
# Script to reorganise the deploy chart structure by moving RBAC resources
# into an rbac folder for better organisation.

set -euo pipefail

# Colours for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Colour

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOY_CHART_DIR="$PROJECT_ROOT/deploy/charts/litellm-operator"
DEPLOY_CHART_TEMPLATES_DIR="$DEPLOY_CHART_DIR/templates"
DEPLOY_CHART_RBAC_DIR="$DEPLOY_CHART_DIR/templates/rbac"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to clean up duplicate files
cleanup_duplicates() {
    log_info "Cleaning up duplicate RBAC files..."
    
    local removed_files=()
    local rbac_patterns=(
        "*-rbac.yaml"
        "manager-rbac.yaml"
        "leader-election-rbac.yaml"
    )
    
    # Remove RBAC files from main templates directory if they exist in rbac directory
    for pattern in "${rbac_patterns[@]}"; do
        for rbac_file in "$DEPLOY_CHART_TEMPLATES_DIR"/$pattern; do
            if [[ -f "$rbac_file" ]]; then
                local filename=$(basename "$rbac_file")
                local target_file="$DEPLOY_CHART_RBAC_DIR/$filename"
                
                if [[ -f "$target_file" ]]; then
                    # Check if files are identical
                    if cmp -s "$rbac_file" "$target_file"; then
                        rm "$rbac_file"
                        removed_files+=("$filename")
                        log_success "Removed duplicate RBAC file: $filename"
                    else
                        log_warning "RBAC file differs from target: $filename"
                    fi
                fi
            fi
        done
    done
    
    if [[ ${#removed_files[@]} -gt 0 ]]; then
        log_success "Removed ${#removed_files[@]} duplicate RBAC file(s): ${removed_files[*]}"
    else
        log_info "No duplicate files found"
    fi
}

# Function to reorganise deploy chart structure
reorganise_deploy_chart() {
    log_info "Reorganising deploy chart structure..."
    
    if [[ ! -d "$DEPLOY_CHART_TEMPLATES_DIR" ]]; then
        log_error "Deploy chart templates directory not found: $DEPLOY_CHART_TEMPLATES_DIR"
        return 1
    fi
    
    # Create rbac directory if it doesn't exist
    if [[ ! -d "$DEPLOY_CHART_RBAC_DIR" ]]; then
        mkdir -p "$DEPLOY_CHART_RBAC_DIR"
        log_success "Created RBAC directory: $DEPLOY_CHART_RBAC_DIR"
    fi
    
    # First, clean up any existing duplicates
    cleanup_duplicates
    
    local moved_files=()
    local rbac_patterns=(
        "*-rbac.yaml"
        "manager-rbac.yaml"
        "leader-election-rbac.yaml"
    )
    
    # Move RBAC files from templates to templates/rbac
    for pattern in "${rbac_patterns[@]}"; do
        for rbac_file in "$DEPLOY_CHART_TEMPLATES_DIR"/$pattern; do
            if [[ -f "$rbac_file" ]]; then
                local filename=$(basename "$rbac_file")
                local target_file="$DEPLOY_CHART_RBAC_DIR/$filename"
                
                if [[ ! -f "$target_file" ]]; then
                    mv "$rbac_file" "$target_file"
                    moved_files+=("$filename")
                    log_success "Moved RBAC file: $filename"
                else
                    log_warning "RBAC file already exists in target: $filename"
                fi
            fi
        done
    done
    
    if [[ ${#moved_files[@]} -gt 0 ]]; then
        log_success "Moved ${#moved_files[@]} RBAC file(s) to rbac directory: ${moved_files[*]}"
    else
        log_info "No RBAC files needed to be moved"
    fi
}

# Function to show help
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Reorganise the deploy chart structure by moving RBAC resources into an rbac folder.

OPTIONS:
    -h, --help          Show this help message
    -d, --dry-run       Show what would be moved without making changes
    -f, --force         Force overwrite of existing files in target directory

EXAMPLES:
    $0                    # Reorganise the deploy chart structure
    $0 --dry-run         # Show what would be moved
    $0 --force           # Force reorganisation even if files exist

EOF
}

# Main function
main() {
    local dry_run=false
    local force=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -d|--dry-run)
                dry_run=true
                shift
                ;;
            -f|--force)
                force=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    log_info "Starting deploy chart reorganisation..."
    log_info "Project root: $PROJECT_ROOT"
    log_info "Deploy chart directory: $DEPLOY_CHART_DIR"
    
    # Check if we're in the right directory
    if [[ ! -f "$PROJECT_ROOT/Makefile" ]]; then
        log_error "Makefile not found. Are you in the correct project directory?"
        exit 1
    fi
    
    if [[ ! -d "$DEPLOY_CHART_DIR" ]]; then
        log_error "Deploy chart directory not found: $DEPLOY_CHART_DIR"
        exit 1
    fi
    
    if [[ "$dry_run" == true ]]; then
        log_info "DRY RUN MODE - No changes will be made"
        log_info "Would create directory: $DEPLOY_CHART_RBAC_DIR"
        
        # Show what files would be moved
        local rbac_patterns=(
            "*-rbac.yaml"
            "manager-rbac.yaml"
            "leader-election-rbac.yaml"
        )
        
        echo ""
        echo "Files that would be moved:"
        for pattern in "${rbac_patterns[@]}"; do
            for rbac_file in "$DEPLOY_CHART_TEMPLATES_DIR"/$pattern; do
                if [[ -f "$rbac_file" ]]; then
                    local filename=$(basename "$rbac_file")
                    local target_file="$DEPLOY_CHART_RBAC_DIR/$filename"
                    
                    if [[ -f "$target_file" ]]; then
                        echo "  $filename -> rbac/$filename (would overwrite)"
                    else
                        echo "  $filename -> rbac/$filename"
                    fi
                fi
            done
        done
        
        echo ""
        echo "Duplicate files that would be cleaned up:"
        for pattern in "${rbac_patterns[@]}"; do
            for rbac_file in "$DEPLOY_CHART_TEMPLATES_DIR"/$pattern; do
                if [[ -f "$rbac_file" ]]; then
                    local filename=$(basename "$rbac_file")
                    local target_file="$DEPLOY_CHART_RBAC_DIR/$filename"
                    
                    if [[ -f "$target_file" ]]; then
                        echo "  $filename (duplicate in templates/)"
                    fi
                fi
            done
        done
        exit 0
    fi
    
    # Perform reorganisation
    reorganise_deploy_chart
    
    log_success "Deploy chart reorganisation completed successfully!"
}

# Run main function with all arguments
main "$@" 