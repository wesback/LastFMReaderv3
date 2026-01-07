#!/usr/bin/env bash
# ============================================================================
# dev-up.sh - Start LastFMReaderv3 Development Environment
# ============================================================================
# This script starts the Docker Compose development environment with proper
# validation and error handling.
#
# Usage:
#   ./scripts/dev-up.sh [options]
#
# Options:
#   --build, -b       Force rebuild of container image
#   --detach, -d      Run in background (detached mode)
#   --help, -h        Show this help message
#
# Examples:
#   ./scripts/dev-up.sh              # Start in foreground
#   ./scripts/dev-up.sh --build      # Rebuild and start
#   ./scripts/dev-up.sh --detach     # Start in background
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
BUILD_FLAG=""
DETACH_FLAG=""

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

# Check if Docker is installed and running
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        echo "Install Docker from: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        echo "Start Docker and try again"
        exit 1
    fi
    
    print_success "Docker is available"
}

# Check if Docker Compose is available
check_docker_compose() {
    if ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not available"
        echo "Install Docker Compose from: https://docs.docker.com/compose/install/"
        exit 1
    fi
    
    print_success "Docker Compose is available"
}

# Check if .env file exists
check_env_file() {
    if [[ ! -f "${PROJECT_ROOT}/.env" ]]; then
        print_warning ".env file not found"
        echo
        print_info "Creating .env from .env.example..."
        
        if [[ ! -f "${PROJECT_ROOT}/.env.example" ]]; then
            print_error ".env.example not found"
            echo "Cannot create .env file"
            exit 1
        fi
        
        cp "${PROJECT_ROOT}/.env.example" "${PROJECT_ROOT}/.env"
        print_success "Created .env file"
        echo
        print_warning "IMPORTANT: Edit .env and add your LASTFM_API_KEY before continuing"
        print_info "Run: nano .env"
        exit 1
    fi
    
    print_success ".env file exists"
}

# Validate required environment variables
validate_env_vars() {
    # Source .env file to check variables
    set -a
    # shellcheck source=/dev/null
    source "${PROJECT_ROOT}/.env"
    set +a
    
    if [[ -z "${LASTFM_API_KEY:-}" ]] || [[ "${LASTFM_API_KEY:-}" == "your-lastfm-api-key-here" ]]; then
        print_error "LASTFM_API_KEY is not set in .env"
        echo
        print_info "Get your API key from: https://www.last.fm/api/account/create"
        print_info "Then edit .env and set LASTFM_API_KEY=your-key-here"
        exit 1
    fi
    
    print_success "LASTFM_API_KEY is configured"
}

# Create data directory if it doesn't exist
create_data_dir() {
    if [[ ! -d "${PROJECT_ROOT}/data" ]]; then
        print_info "Creating data directory..."
        mkdir -p "${PROJECT_ROOT}/data/state"
        print_success "Created data directory"
    else
        print_success "Data directory exists"
    fi
}

# Start Docker Compose
start_compose() {
    print_info "Starting Docker Compose..."
    echo
    
    cd "${PROJECT_ROOT}"
    
    # Build compose command
    local compose_cmd="docker compose up"
    
    if [[ -n "${BUILD_FLAG}" ]]; then
        compose_cmd="${compose_cmd} --build"
    fi
    
    if [[ -n "${DETACH_FLAG}" ]]; then
        compose_cmd="${compose_cmd} -d"
    fi
    
    # Run compose
    if ${compose_cmd}; then
        echo
        print_success "Container started successfully"
        
        if [[ -n "${DETACH_FLAG}" ]]; then
            echo
            print_info "Container is running in background"
            print_info "View logs: docker compose logs -f"
            print_info "Stop container: docker compose down"
        fi
    else
        echo
        print_error "Failed to start container"
        exit 1
    fi
}

# ============================================================================
# Main
# ============================================================================

main() {
    # Parse command-line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --build|-b)
                BUILD_FLAG="--build"
                shift
                ;;
            --detach|-d)
                DETACH_FLAG="-d"
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
    print_info "LastFMReaderv3 Development Environment"
    echo
    
    # Pre-flight checks
    check_docker
    check_docker_compose
    check_env_file
    validate_env_vars
    create_data_dir
    
    echo
    
    # Start environment
    start_compose
    
    echo
    print_success "Development environment is ready"
    echo
}

# Run main function
main "$@"
