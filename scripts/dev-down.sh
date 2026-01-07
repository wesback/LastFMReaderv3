#!/usr/bin/env bash
# ============================================================================
# dev-down.sh - Stop LastFMReaderv3 Development Environment
# ============================================================================
# This script stops and cleans up the Docker Compose development environment.
#
# Usage:
#   ./scripts/dev-down.sh [options]
#
# Options:
#   --volumes, -v     Remove volumes (deletes data)
#   --images, -i      Remove images
#   --all, -a         Remove everything (volumes + images)
#   --help, -h        Show this help message
#
# Examples:
#   ./scripts/dev-down.sh              # Stop containers only
#   ./scripts/dev-down.sh --volumes    # Stop and remove data
#   ./scripts/dev-down.sh --all        # Complete cleanup
# ============================================================================

set -euo pipefail  # Exit on error, undefined variable, or pipe failure

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory (for relative paths)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default options
VOLUMES_FLAG=""
IMAGES_FLAG=""

# ============================================================================
# Functions
# ============================================================================

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

show_help() {
    head -n 20 "$0" | grep "^#" | sed 's/^# \?//'
    exit 0
}

# Check if Docker Compose is available
check_docker_compose() {
    if ! command -v docker &> /dev/null || ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not available"
        exit 1
    fi
}

# Stop Docker Compose services
stop_compose() {
    print_info "Stopping Docker Compose services..."
    
    cd "${PROJECT_ROOT}"
    
    # Build compose down command
    local compose_cmd="docker compose down"
    
    if [[ -n "${VOLUMES_FLAG}" ]]; then
        compose_cmd="${compose_cmd} --volumes"
        print_warning "This will delete all data in volumes!"
    fi
    
    # Stop services
    if ${compose_cmd}; then
        print_success "Services stopped successfully"
    else
        print_error "Failed to stop services"
        exit 1
    fi
}

# Remove images
remove_images() {
    if [[ -n "${IMAGES_FLAG}" ]]; then
        print_info "Removing images..."
        
        cd "${PROJECT_ROOT}"
        
        if docker compose down --rmi local; then
            print_success "Images removed successfully"
        else
            print_warning "Failed to remove some images"
        fi
    fi
}

# Show cleanup summary
show_summary() {
    echo
    print_info "Cleanup Summary:"
    
    echo "  • Containers: Stopped and removed"
    
    if [[ -n "${VOLUMES_FLAG}" ]]; then
        echo "  • Volumes: Removed (data deleted)"
    else
        echo "  • Volumes: Preserved (data intact)"
    fi
    
    if [[ -n "${IMAGES_FLAG}" ]]; then
        echo "  • Images: Removed"
    else
        echo "  • Images: Preserved"
    fi
}

# Confirm destructive operations
confirm_destructive() {
    if [[ -n "${VOLUMES_FLAG}" ]]; then
        echo
        print_warning "WARNING: This will delete all data in ./data/ volume"
        echo
        read -p "Are you sure you want to continue? (yes/no): " -r
        echo
        
        if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            print_info "Operation cancelled"
            exit 0
        fi
    fi
}

# ============================================================================
# Main
# ============================================================================

main() {
    # Parse command-line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --volumes|-v)
                VOLUMES_FLAG="--volumes"
                shift
                ;;
            --images|-i)
                IMAGES_FLAG="--rmi"
                shift
                ;;
            --all|-a)
                VOLUMES_FLAG="--volumes"
                IMAGES_FLAG="--rmi"
                shift
                ;;
            --help|-h)
                show_help
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    echo
    print_info "LastFMReaderv3 Development Environment Cleanup"
    echo
    
    # Pre-flight checks
    check_docker_compose
    
    # Confirm if destructive
    confirm_destructive
    
    # Stop services
    stop_compose
    
    # Remove images if requested
    remove_images
    
    # Show summary
    show_summary
    
    echo
    print_success "Cleanup complete"
    
    if [[ -z "${VOLUMES_FLAG}" ]]; then
        echo
        print_info "To remove data volumes, run: ./scripts/dev-down.sh --volumes"
    fi
    
    echo
}

# Run main function
main "$@"
