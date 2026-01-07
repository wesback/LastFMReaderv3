#!/usr/bin/env bash
# ============================================================================
# deploy-aci.sh - Deploy LastFMReaderv3 to Azure Container Instances
# ============================================================================
# This script automates deployment of lastfm-sync to Azure Container Instances
# with managed identity, Key Vault integration, and Log Analytics monitoring.
#
# Usage:
#   ./azure/deploy-aci.sh [options]
#
# Options:
#   -p, --params FILE     Path to parameters JSON file (required)
#   -u, --user USER       Last.fm username to fetch (required)
#   -i, --image IMAGE     Container image (default: from params file)
#   --dry-run             Show what would be deployed without deploying
#   -h, --help            Show this help message
#
# Examples:
#   ./azure/deploy-aci.sh -p azure/aci-params.json -u alice
#   ./azure/deploy-aci.sh -p azure/aci-params.json -u bob --dry-run
#
# Prerequisites:
#   - Azure CLI installed and authenticated (az login)
#   - jq installed (for JSON parsing)
#   - Secrets stored in Azure Key Vault
#   - Managed identity created with appropriate roles
#
# See docs/azure-deployment.md for complete documentation.
# ============================================================================

set -euo pipefail  # Exit on error, undefined variable, or pipe failure

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default values
PARAMS_FILE=""
LASTFM_USER=""
CONTAINER_IMAGE=""
DRY_RUN=false

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
    head -n 30 "$0" | grep "^#" | sed 's/^# \?//'
    exit 0
}

# Check if required tools are installed
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check Azure CLI
    if ! command -v az &> /dev/null; then
        print_error "Azure CLI is not installed"
        echo "Install from: https://aka.ms/installazurecli"
        exit 1
    fi
    print_success "Azure CLI is available"
    
    # Check jq
    if ! command -v jq &> /dev/null; then
        print_error "jq is not installed"
        echo "Install with: apt-get install jq (Linux) or brew install jq (macOS)"
        exit 1
    fi
    print_success "jq is available"
    
    # Check Azure login status
    if ! az account show &> /dev/null; then
        print_error "Not logged in to Azure"
        echo "Run: az login"
        exit 1
    fi
    print_success "Azure authentication is valid"
}

# Load and validate parameters file
load_parameters() {
    print_info "Loading parameters from $PARAMS_FILE..."
    
    if [[ ! -f "$PARAMS_FILE" ]]; then
        print_error "Parameters file not found: $PARAMS_FILE"
        exit 1
    fi
    
    # Parse JSON parameters
    RESOURCE_GROUP=$(jq -r '.resourceGroup' "$PARAMS_FILE")
    LOCATION=$(jq -r '.location' "$PARAMS_FILE")
    CONTAINER_NAME_PREFIX=$(jq -r '.containerNamePrefix' "$PARAMS_FILE")
    STORAGE_ACCOUNT=$(jq -r '.storageAccount' "$PARAMS_FILE")
    STORAGE_CONTAINER=$(jq -r '.storageContainer' "$PARAMS_FILE")
    KEY_VAULT_NAME=$(jq -r '.keyVaultName' "$PARAMS_FILE")
    MANAGED_IDENTITY=$(jq -r '.managedIdentity' "$PARAMS_FILE")
    LOG_ANALYTICS_WORKSPACE=$(jq -r '.logAnalyticsWorkspace' "$PARAMS_FILE")
    CPU=$(jq -r '.cpu // 0.5' "$PARAMS_FILE")
    MEMORY=$(jq -r '.memory // 0.5' "$PARAMS_FILE")
    
    # Use provided image or default from params
    if [[ -z "$CONTAINER_IMAGE" ]]; then
        CONTAINER_IMAGE=$(jq -r '.containerImage // "ghcr.io/lastfm-reader/lastfm-sync:latest"' "$PARAMS_FILE")
    fi
    
    # Validate required parameters
    if [[ "$RESOURCE_GROUP" == "null" ]] || [[ -z "$RESOURCE_GROUP" ]]; then
        print_error "resourceGroup is required in parameters file"
        exit 1
    fi
    
    if [[ "$LOCATION" == "null" ]] || [[ -z "$LOCATION" ]]; then
        print_error "location is required in parameters file"
        exit 1
    fi
    
    if [[ "$STORAGE_ACCOUNT" == "null" ]] || [[ -z "$STORAGE_ACCOUNT" ]]; then
        print_error "storageAccount is required in parameters file"
        exit 1
    fi
    
    if [[ "$KEY_VAULT_NAME" == "null" ]] || [[ -z "$KEY_VAULT_NAME" ]]; then
        print_error "keyVaultName is required in parameters file"
        exit 1
    fi
    
    if [[ "$MANAGED_IDENTITY" == "null" ]] || [[ -z "$MANAGED_IDENTITY" ]]; then
        print_error "managedIdentity is required in parameters file"
        exit 1
    fi
    
    print_success "Parameters loaded successfully"
}

# Fetch secrets from Key Vault
fetch_secrets() {
    print_info "Fetching secrets from Key Vault: $KEY_VAULT_NAME..."
    
    LASTFM_API_KEY=$(az keyvault secret show \
        --vault-name "$KEY_VAULT_NAME" \
        --name lastfm-api-key \
        --query value -o tsv 2>/dev/null)
    
    if [[ -z "$LASTFM_API_KEY" ]]; then
        print_error "Failed to fetch LASTFM_API_KEY from Key Vault"
        echo "Ensure the secret 'lastfm-api-key' exists in vault '$KEY_VAULT_NAME'"
        exit 1
    fi
    
    print_success "Secrets fetched successfully"
}

# Get Log Analytics workspace credentials
get_log_analytics_credentials() {
    if [[ "$LOG_ANALYTICS_WORKSPACE" != "null" ]] && [[ -n "$LOG_ANALYTICS_WORKSPACE" ]]; then
        print_info "Getting Log Analytics credentials..."
        
        WORKSPACE_ID=$(az monitor log-analytics workspace show \
            --resource-group "$RESOURCE_GROUP" \
            --workspace-name "$LOG_ANALYTICS_WORKSPACE" \
            --query customerId -o tsv 2>/dev/null)
        
        WORKSPACE_KEY=$(az monitor log-analytics workspace get-shared-keys \
            --resource-group "$RESOURCE_GROUP" \
            --workspace-name "$LOG_ANALYTICS_WORKSPACE" \
            --query primarySharedKey -o tsv 2>/dev/null)
        
        if [[ -z "$WORKSPACE_ID" ]] || [[ -z "$WORKSPACE_KEY" ]]; then
            print_warning "Failed to get Log Analytics credentials"
            print_warning "Deploying without Log Analytics integration"
            LOG_ANALYTICS_WORKSPACE=""
        else
            print_success "Log Analytics credentials retrieved"
        fi
    fi
}

# Create resource group if it doesn't exist
ensure_resource_group() {
    print_info "Ensuring resource group exists: $RESOURCE_GROUP..."
    
    if ! az group show --name "$RESOURCE_GROUP" &> /dev/null; then
        if [[ "$DRY_RUN" == true ]]; then
            print_info "[DRY-RUN] Would create resource group: $RESOURCE_GROUP"
        else
            az group create \
                --name "$RESOURCE_GROUP" \
                --location "$LOCATION" \
                --output none
            print_success "Resource group created"
        fi
    else
        print_success "Resource group already exists"
    fi
}

# Deploy container instance
deploy_container() {
    CONTAINER_NAME="${CONTAINER_NAME_PREFIX}-${LASTFM_USER}"
    
    print_info "Deploying container: $CONTAINER_NAME..."
    
    # Build command
    local deploy_cmd="az container create"
    deploy_cmd="$deploy_cmd --resource-group '$RESOURCE_GROUP'"
    deploy_cmd="$deploy_cmd --name '$CONTAINER_NAME'"
    deploy_cmd="$deploy_cmd --image '$CONTAINER_IMAGE'"
    deploy_cmd="$deploy_cmd --cpu $CPU"
    deploy_cmd="$deploy_cmd --memory $MEMORY"
    deploy_cmd="$deploy_cmd --restart-policy Never"
    deploy_cmd="$deploy_cmd --assign-identity '$MANAGED_IDENTITY'"
    deploy_cmd="$deploy_cmd --environment-variables"
    deploy_cmd="$deploy_cmd   LASTFM_API_KEY='$LASTFM_API_KEY'"
    deploy_cmd="$deploy_cmd   AZURE_STORAGE_ACCOUNT='$STORAGE_ACCOUNT'"
    deploy_cmd="$deploy_cmd --command-line '/app/lastfm-sync fetch --user $LASTFM_USER --output azure --azure-container $STORAGE_CONTAINER --azure-auth mi'"
    
    # Add Log Analytics if available
    if [[ -n "$LOG_ANALYTICS_WORKSPACE" ]] && [[ -n "$WORKSPACE_ID" ]]; then
        deploy_cmd="$deploy_cmd --log-analytics-workspace '$WORKSPACE_ID'"
        deploy_cmd="$deploy_cmd --log-analytics-workspace-key '$WORKSPACE_KEY'"
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        print_info "[DRY-RUN] Would execute:"
        echo "$deploy_cmd"
        return
    fi
    
    # Execute deployment
    if eval "$deploy_cmd --output none" 2>&1 | tee /tmp/aci-deploy.log; then
        print_success "Container deployed successfully"
    else
        print_error "Deployment failed"
        cat /tmp/aci-deploy.log
        exit 1
    fi
}

# Show deployment summary
show_summary() {
    echo
    print_success "Deployment Summary"
    echo "─────────────────────────────────────────────"
    echo "  Resource Group: $RESOURCE_GROUP"
    echo "  Location: $LOCATION"
    echo "  Container Name: ${CONTAINER_NAME_PREFIX}-${LASTFM_USER}"
    echo "  Image: $CONTAINER_IMAGE"
    echo "  CPU: $CPU vCPU"
    echo "  Memory: $MEMORY GB"
    echo "  User: $LASTFM_USER"
    echo "  Storage Account: $STORAGE_ACCOUNT"
    echo "  Storage Container: $STORAGE_CONTAINER"
    if [[ -n "$LOG_ANALYTICS_WORKSPACE" ]]; then
        echo "  Log Analytics: $LOG_ANALYTICS_WORKSPACE"
    fi
    echo "─────────────────────────────────────────────"
    
    if [[ "$DRY_RUN" == false ]]; then
        echo
        print_info "View logs with:"
        echo "  az container logs --resource-group $RESOURCE_GROUP --name ${CONTAINER_NAME_PREFIX}-${LASTFM_USER} --follow"
        echo
        print_info "Check status with:"
        echo "  az container show --resource-group $RESOURCE_GROUP --name ${CONTAINER_NAME_PREFIX}-${LASTFM_USER}"
        echo
    fi
}

# Cleanup on error
cleanup_on_error() {
    if [[ $? -ne 0 ]]; then
        print_error "Deployment failed"
        
        if [[ "$DRY_RUN" == false ]] && [[ -n "${CONTAINER_NAME:-}" ]]; then
            print_info "Cleaning up failed deployment..."
            az container delete \
                --resource-group "$RESOURCE_GROUP" \
                --name "$CONTAINER_NAME" \
                --yes \
                --output none 2>/dev/null || true
        fi
    fi
}

trap cleanup_on_error EXIT

# ============================================================================
# Main
# ============================================================================

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -p|--params)
                PARAMS_FILE="$2"
                shift 2
                ;;
            -u|--user)
                LASTFM_USER="$2"
                shift 2
                ;;
            -i|--image)
                CONTAINER_IMAGE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            -h|--help)
                show_help
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    # Validate required arguments
    if [[ -z "$PARAMS_FILE" ]]; then
        print_error "Parameters file is required"
        echo "Usage: $0 -p azure/aci-params.json -u alice"
        exit 1
    fi
    
    if [[ -z "$LASTFM_USER" ]]; then
        print_error "Last.fm username is required"
        echo "Usage: $0 -p azure/aci-params.json -u alice"
        exit 1
    fi
    
    echo
    print_info "LastFMReaderv3 Azure Container Instances Deployment"
    echo
    
    if [[ "$DRY_RUN" == true ]]; then
        print_warning "DRY-RUN MODE: No changes will be made"
        echo
    fi
    
    # Execute deployment
    check_prerequisites
    load_parameters
    fetch_secrets
    get_log_analytics_credentials
    ensure_resource_group
    deploy_container
    show_summary
    
    echo
    print_success "Deployment complete!"
    echo
}

# Run main function
main "$@"
