#!/bin/bash

# GitHub Packages Management Script
# Usage: ./manage-packages.sh [command] [options]

set -e

ORG="bbdsoftware"
PACKAGE_TYPE="container"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# List all packages
list_packages() {
    local package_type=${1:-$PACKAGE_TYPE}
    print_info "Listing litellm $package_type packages for organization: $ORG"
    
    {
        echo -e "ID\tName\tUpdated"
        gh api "/orgs/$ORG/packages?package_type=$package_type" | \
            jq -r '.[] | select(.name | test("litellm-operator")) | "\(.id)\t\(.name)\t\(.updated_at)"'
    } | column -t -s $'\t'
}

# List packages with details
list_packages_detailed() {
    local package_type=${1:-$PACKAGE_TYPE}
    print_info "Listing detailed litellm $package_type packages for organization: $ORG"
    
    {
        echo -e "ID\tName\tVisibility\tUpdated\tRepository"
        gh api "/orgs/$ORG/packages?package_type=$package_type" | \
            jq -r '.[] | select(.name | test("litellm-operator")) | "\(.id)\t\(.name)\t\(.visibility)\t\(.updated_at)\t\(.repository.name // "N/A")"'
    } | column -t -s $'\t'
}

# Delete a specific package
delete_package() {
    local package_name="$1"
    local package_type=${2:-$PACKAGE_TYPE}
    
    if [ -z "$package_name" ]; then
        print_error "Package name is required"
        exit 1
    fi
    
    print_warning "Are you sure you want to delete package '$package_name'? (y/N)"
    read -r confirm
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        if gh api -X DELETE "/orgs/$ORG/packages/$package_type/$package_name" 2>/dev/null; then
            print_success "Deleted package: $package_name"
        else
            print_error "Failed to delete package: $package_name"
        fi
    else
        print_info "Deletion cancelled"
    fi
}

# Delete packages matching a pattern
delete_packages_pattern() {
    local pattern="$1"
    local package_type=${2:-$PACKAGE_TYPE}
    
    if [ -z "$pattern" ]; then
        print_error "Pattern is required"
        exit 1
    fi
    
    print_info "Finding packages matching pattern: $pattern"
    
    local packages
    packages=$(gh api "/orgs/$ORG/packages?package_type=$package_type" | \
        jq -r ".[] | select(.name | test(\"litellm\")) | select(.name | contains(\"$pattern\")) | .name")
    
    if [ -z "$packages" ]; then
        print_info "No packages found matching pattern: $pattern"
        return
    fi
    
    print_warning "Found packages to delete:"
    echo "$packages"
    echo ""
    print_warning "Are you sure you want to delete these packages? (y/N)"
    read -r confirm
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        echo "$packages" | while read -r package_name; do
            if gh api -X DELETE "/orgs/$ORG/packages/$package_type/$package_name" 2>/dev/null; then
                print_success "Deleted package: $package_name"
            else
                print_error "Failed to delete package: $package_name"
            fi
        done
    else
        print_info "Deletion cancelled"
    fi
}

# Delete all packages (DANGEROUS!)
delete_all_packages() {
    local package_type=${1:-$PACKAGE_TYPE}
    
    print_error "WARNING: This will delete ALL litellm $package_type packages in organization $ORG"
    print_warning "Type 'DELETE ALL' to confirm:"
    read -r confirm
    
    if [ "$confirm" = "DELETE ALL" ]; then
        gh api "/orgs/$ORG/packages?package_type=$package_type" | \
            jq -r '.[] | select(.name | test("litellm")) | .name' | \
            while read -r package_name; do
                if gh api -X DELETE "/orgs/$ORG/packages/$package_type/$package_name" 2>/dev/null; then
                    print_success "Deleted package: $package_name"
                else
                    print_error "Failed to delete package: $package_name"
                fi
            done
    else
        print_info "Deletion cancelled"
    fi
}

# Get package details
package_info() {
    local package_name="$1"
    local package_type=${2:-$PACKAGE_TYPE}
    
    if [ -z "$package_name" ]; then
        print_error "Package name is required"
        exit 1
    fi
    
    print_info "Package details for: $package_name"
    gh api "/orgs/$ORG/packages/$package_type/$package_name" | jq .
}

# List package versions
list_versions() {
    local package_name="$1"
    local package_type=${2:-$PACKAGE_TYPE}
    
    if [ -z "$package_name" ]; then
        print_error "Package name is required"
        exit 1
    fi
    
    print_info "Versions for package: $package_name"
    {
        echo -e "ID\tVersion\tCreated"
        gh api "/orgs/$ORG/packages/$package_type/$package_name/versions" | \
            jq -r '.[] | "\(.id)\t\(.name)\t\(.created_at)"'
    } | column -t -s $'\t'
}

# Show usage
usage() {
    echo "GitHub Packages Management Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  list [type]                    - List packages (default: container)"
    echo "  list-detailed [type]           - List packages with details"
    echo "  delete <name> [type]           - Delete a specific package"
    echo "  delete-pattern <pattern> [type] - Delete packages matching pattern"
    echo "  delete-all [type]              - Delete ALL packages (DANGEROUS!)"
    echo "  info <name> [type]             - Show package details"
    echo "  versions <name> [type]         - List package versions"
    echo ""
    echo "Package types: container, npm, maven, nuget, rubygems, docker"
    echo "Default organization: $ORG"
    echo "Default package type: $PACKAGE_TYPE"
    echo ""
    echo "Examples:"
    echo "  $0 list"
    echo "  $0 delete litellm-operator"
    echo "  $0 delete-pattern litellm"
    echo "  $0 info my-package"
}

# Main script logic
case "${1:-}" in
    "list")
        list_packages "$2"
        ;;
    "list-detailed")
        list_packages_detailed "$2"
        ;;
    "delete")
        delete_package "$2" "$3"
        ;;
    "delete-pattern")
        delete_packages_pattern "$2" "$3"
        ;;
    "delete-all")
        delete_all_packages "$2"
        ;;
    "info")
        package_info "$2" "$3"
        ;;
    "versions")
        list_versions "$2" "$3"
        ;;
    "help"|"-h"|"--help")
        usage
        ;;
    *)
        usage
        exit 1
        ;;
esac
