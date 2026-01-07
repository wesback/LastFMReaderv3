# Azure Container Instances Deployment Guide

Complete guide for deploying LastFMReaderv3 to Azure Container Instances (ACI) with production-ready security and observability.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Step-by-Step Deployment](#step-by-step-deployment)
- [Azure Key Vault Integration](#azure-key-vault-integration)
- [Managed Identity Setup](#managed-identity-setup)
- [Networking Configuration](#networking-configuration)
- [Persistent Storage](#persistent-storage)
- [Logging and Monitoring](#logging-and-monitoring)
- [Structured Logging Format](#structured-logging-format)
- [Scaling Considerations](#scaling-considerations)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

---

## Overview

Azure Container Instances (ACI) provides serverless container execution without managing infrastructure. This guide covers production deployment with:

- ✅ Managed Identity authentication (no connection strings)
- ✅ Azure Key Vault for secrets management
- ✅ Log Analytics workspace integration
- ✅ Custom metrics and structured logging
- ✅ Automated deployment scripts
- ✅ Cost-effective resource allocation

**Architecture**:
```
┌─────────────────────────────────────────────────┐
│ Azure Container Instances                       │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │ lastfm-sync Container                    │  │
│  │  • Managed Identity enabled              │  │
│  │  • Fetches secrets from Key Vault        │  │
│  │  • Writes to Azure Blob Storage          │  │
│  │  • Sends logs to Log Analytics           │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
└─────────────────────────────────────────────────┘
           │           │              │
           ↓           ↓              ↓
    ┌──────────┐ ┌─────────────┐ ┌──────────────────┐
    │ Key Vault│ │Blob Storage │ │ Log Analytics    │
    │ (secrets)│ │  (output)   │ │ (monitoring)     │
    └──────────┘ └─────────────┘ └──────────────────┘
```

---

## Prerequisites

### Azure Subscription

- Active Azure subscription
- Contributor or Owner role on resource group

### Azure CLI

Install Azure CLI 2.50.0+:

```bash
# macOS
brew install azure-cli

# Linux
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Windows
# Download from https://aka.ms/installazurecliwindows

# Verify installation
az --version
```

### Authentication

```bash
# Login to Azure
az login

# Set default subscription (if you have multiple)
az account set --subscription "your-subscription-id"

# Verify current subscription
az account show
```

### Required Permissions

Your account needs these permissions:
- **Microsoft.ContainerInstance/containerGroups/write**: Create container instances
- **Microsoft.ManagedIdentity/userAssignedIdentities/assign/action**: Assign managed identity
- **Microsoft.Storage/storageAccounts/read**: Read storage account
- **Microsoft.KeyVault/vaults/secrets/read**: Read Key Vault secrets (via managed identity)

---

## Quick Start

### 1. Prepare Parameters

Copy and edit the deployment parameters:

```bash
cp azure/aci-params.json.example azure/aci-params.json
nano azure/aci-params.json
```

Edit values:
- `resourceGroup`: Your Azure resource group name
- `location`: Azure region (e.g., `eastus`, `westus2`)
- `containerName`: Unique name for container instance
- `storageAccount`: Your Azure Storage account name
- `keyVaultName`: Your Azure Key Vault name

### 2. Store Secrets in Key Vault

```bash
# Set your Last.fm API key in Key Vault
az keyvault secret set \
  --vault-name your-keyvault \
  --name lastfm-api-key \
  --value "your-lastfm-api-key"
```

### 3. Deploy

```bash
# Run deployment script
./azure/deploy-aci.sh -p azure/aci-params.json

# Or deploy manually (see below)
```

---

## Step-by-Step Deployment

### Step 1: Create Resource Group

```bash
# Create resource group
az group create \
  --name lastfm-rg \
  --location eastus

# Verify
az group show --name lastfm-rg
```

### Step 2: Create Storage Account

```bash
# Create storage account
az storage account create \
  --name lastfmstorage$(date +%s | tail -c 6) \
  --resource-group lastfm-rg \
  --location eastus \
  --sku Standard_LRS \
  --kind StorageV2

# Create container
az storage container create \
  --name scrobbles \
  --account-name lastfmstorage123456 \
  --auth-mode login
```

### Step 3: Create Key Vault

```bash
# Create Key Vault
az keyvault create \
  --name lastfm-kv-$(date +%s | tail -c 6) \
  --resource-group lastfm-rg \
  --location eastus \
  --enable-rbac-authorization

# Store Last.fm API key
az keyvault secret set \
  --vault-name lastfm-kv-123456 \
  --name lastfm-api-key \
  --value "your-api-key-here"
```

### Step 4: Create Managed Identity

```bash
# Create user-assigned managed identity
az identity create \
  --name lastfm-identity \
  --resource-group lastfm-rg \
  --location eastus

# Get identity details
IDENTITY_ID=$(az identity show \
  --name lastfm-identity \
  --resource-group lastfm-rg \
  --query id -o tsv)

PRINCIPAL_ID=$(az identity show \
  --name lastfm-identity \
  --resource-group lastfm-rg \
  --query principalId -o tsv)

echo "Identity ID: $IDENTITY_ID"
echo "Principal ID: $PRINCIPAL_ID"
```

### Step 5: Assign Permissions

```bash
# Storage Blob Data Contributor role
az role assignment create \
  --assignee $PRINCIPAL_ID \
  --role "Storage Blob Data Contributor" \
  --scope "/subscriptions/$(az account show --query id -o tsv)/resourceGroups/lastfm-rg/providers/Microsoft.Storage/storageAccounts/lastfmstorage123456"

# Key Vault Secrets User role
az role assignment create \
  --assignee $PRINCIPAL_ID \
  --role "Key Vault Secrets User" \
  --scope "/subscriptions/$(az account show --query id -o tsv)/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv-123456"
```

### Step 6: Create Log Analytics Workspace

```bash
# Create workspace
az monitor log-analytics workspace create \
  --resource-group lastfm-rg \
  --workspace-name lastfm-logs \
  --location eastus

# Get workspace credentials
WORKSPACE_ID=$(az monitor log-analytics workspace show \
  --resource-group lastfm-rg \
  --workspace-name lastfm-logs \
  --query customerId -o tsv)

WORKSPACE_KEY=$(az monitor log-analytics workspace get-shared-keys \
  --resource-group lastfm-rg \
  --workspace-name lastfm-logs \
  --query primarySharedKey -o tsv)

echo "Workspace ID: $WORKSPACE_ID"
echo "Workspace Key: $WORKSPACE_KEY"
```

### Step 7: Deploy Container Instance

```bash
# Fetch secret from Key Vault
LASTFM_API_KEY=$(az keyvault secret show \
  --vault-name lastfm-kv-123456 \
  --name lastfm-api-key \
  --query value -o tsv)

# Deploy container with managed identity
az container create \
  --resource-group lastfm-rg \
  --name lastfm-sync-alice \
  --image ghcr.io/lastfm-reader/lastfm-sync:latest \
  --cpu 0.5 \
  --memory 0.5 \
  --restart-policy Never \
  --assign-identity $IDENTITY_ID \
  --environment-variables \
    LASTFM_API_KEY="$LASTFM_API_KEY" \
    AZURE_STORAGE_ACCOUNT=lastfmstorage123456 \
  --command-line "/app/lastfm-sync fetch --user alice --output azure --azure-container scrobbles --azure-auth mi" \
  --log-analytics-workspace $WORKSPACE_ID \
  --log-analytics-workspace-key $WORKSPACE_KEY

# Verify deployment
az container show \
  --resource-group lastfm-rg \
  --name lastfm-sync-alice \
  --query "{FQDN:ipAddress.fqdn,ProvisioningState:provisioningState}" --output table
```

---

## Azure Key Vault Integration

### Why Key Vault?

- ✅ Centralized secret management
- ✅ Automatic secret rotation support
- ✅ Access logging and auditing
- ✅ No secrets in environment variables or code

### Creating and Managing Secrets

```bash
# Set secret
az keyvault secret set \
  --vault-name your-keyvault \
  --name lastfm-api-key \
  --value "your-secret-value"

# Get secret (for deployment)
az keyvault secret show \
  --vault-name your-keyvault \
  --name lastfm-api-key \
  --query value -o tsv

# List secrets
az keyvault secret list \
  --vault-name your-keyvault \
  --query "[].name" -o table

# Rotate secret (update value)
az keyvault secret set \
  --vault-name your-keyvault \
  --name lastfm-api-key \
  --value "new-secret-value"
```

### Using Key Vault References in Container

**Option 1**: Fetch at deployment time (recommended for one-off jobs)

```bash
LASTFM_API_KEY=$(az keyvault secret show --vault-name kv --name lastfm-api-key --query value -o tsv)
az container create ... --environment-variables LASTFM_API_KEY="$LASTFM_API_KEY"
```

**Option 2**: Use managed identity to fetch at runtime (requires code changes)

```go
// Modify application to fetch from Key Vault using DefaultAzureCredential
// Not implemented in current version - future enhancement
```

---

## Managed Identity Setup

Managed Identity eliminates the need for connection strings and access keys.

### User-Assigned vs System-Assigned

| Feature | User-Assigned | System-Assigned |
|---------|---------------|-----------------|
| Lifecycle | Independent of resource | Tied to resource |
| Reusability | Can be shared across resources | One per resource |
| Setup | More complex | Simpler |
| **Recommended** | ✅ Production | Development |

### Creating User-Assigned Identity

```bash
# Create identity
az identity create \
  --name lastfm-identity \
  --resource-group lastfm-rg

# Get ID and Principal ID
IDENTITY_ID=$(az identity show --name lastfm-identity --resource-group lastfm-rg --query id -o tsv)
PRINCIPAL_ID=$(az identity show --name lastfm-identity --resource-group lastfm-rg --query principalId -o tsv)
```

### Assigning Roles

```bash
# Storage Blob Data Contributor (read/write blobs)
az role assignment create \
  --assignee $PRINCIPAL_ID \
  --role "Storage Blob Data Contributor" \
  --scope /subscriptions/{subscription-id}/resourceGroups/lastfm-rg

# Key Vault Secrets User (read secrets)
az role assignment create \
  --assignee $PRINCIPAL_ID \
  --role "Key Vault Secrets User" \
  --scope /subscriptions/{subscription-id}/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv
```

### Using Managed Identity in Container

```bash
az container create \
  --name lastfm-sync \
  --assign-identity $IDENTITY_ID \
  --environment-variables AZURE_STORAGE_ACCOUNT=lastfmstorage \
  --command-line "/app/lastfm-sync fetch --user alice --output azure --azure-container scrobbles --azure-auth mi"
```

---

## Networking Configuration

### Virtual Network Integration

For private deployments:

```bash
# Create VNet and subnet
az network vnet create \
  --resource-group lastfm-rg \
  --name lastfm-vnet \
  --address-prefix 10.0.0.0/16 \
  --subnet-name container-subnet \
  --subnet-prefix 10.0.1.0/24

# Deploy container in VNet
az container create \
  --resource-group lastfm-rg \
  --name lastfm-sync \
  --image lastfm-sync:latest \
  --vnet lastfm-vnet \
  --subnet container-subnet
```

### Private Endpoints

Connect to storage and Key Vault via private endpoints:

```bash
# Create private endpoint for storage
az network private-endpoint create \
  --name storage-pe \
  --resource-group lastfm-rg \
  --vnet-name lastfm-vnet \
  --subnet container-subnet \
  --private-connection-resource-id "/subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.Storage/storageAccounts/lastfmstorage" \
  --group-id blob \
  --connection-name storage-connection
```

### Network Security Groups

Restrict outbound traffic:

```bash
az network nsg create \
  --resource-group lastfm-rg \
  --name lastfm-nsg

az network nsg rule create \
  --resource-group lastfm-rg \
  --nsg-name lastfm-nsg \
  --name allow-https-outbound \
  --priority 100 \
  --direction Outbound \
  --access Allow \
  --protocol Tcp \
  --destination-port-ranges 443
```

---

## Persistent Storage

### Azure File Share (for local output mode)

```bash
# Create file share
az storage share create \
  --name lastfm-data \
  --account-name lastfmstorage

# Get storage account key
STORAGE_KEY=$(az storage account keys list \
  --resource-group lastfm-rg \
  --account-name lastfmstorage \
  --query "[0].value" -o tsv)

# Mount file share in container
az container create \
  --resource-group lastfm-rg \
  --name lastfm-sync \
  --image lastfm-sync:latest \
  --azure-file-volume-account-name lastfmstorage \
  --azure-file-volume-account-key $STORAGE_KEY \
  --azure-file-volume-share-name lastfm-data \
  --azure-file-volume-mount-path /data \
  --command-line "/app/lastfm-sync fetch --user alice --output local"
```

---

## Logging and Monitoring

### Azure Portal Logs

**View logs in Azure Portal**:
1. Navigate to Container Instances
2. Select your container
3. Click "Containers" in left menu
4. Click "Logs" tab
5. View real-time output

**Download logs**:
```bash
az container logs \
  --resource-group lastfm-rg \
  --name lastfm-sync-alice \
  --follow
```

### Log Analytics Workspace Integration

Log Analytics provides advanced querying and alerting capabilities.

**Create workspace** (if not exists):
```bash
az monitor log-analytics workspace create \
  --resource-group lastfm-rg \
  --workspace-name lastfm-logs \
  --location eastus
```

**Get workspace credentials**:
```bash
WORKSPACE_ID=$(az monitor log-analytics workspace show \
  --resource-group lastfm-rg \
  --workspace-name lastfm-logs \
  --query customerId -o tsv)

WORKSPACE_KEY=$(az monitor log-analytics workspace get-shared-keys \
  --resource-group lastfm-rg \
  --workspace-name lastfm-logs \
  --query primarySharedKey -o tsv)
```

**Deploy container with Log Analytics**:
```bash
az container create \
  --resource-group lastfm-rg \
  --name lastfm-sync \
  --image lastfm-sync:latest \
  --log-analytics-workspace $WORKSPACE_ID \
  --log-analytics-workspace-key $WORKSPACE_KEY \
  ...
```

**Query logs in Log Analytics**:

```kusto
// All logs from lastfm-sync container
ContainerInstanceLog_CL
| where ContainerGroup_s == "lastfm-sync"
| project TimeGenerated, Message
| order by TimeGenerated desc

// Errors only
ContainerInstanceLog_CL
| where ContainerGroup_s == "lastfm-sync"
| where Message contains "error" or Message contains "Error"
| project TimeGenerated, Message
| order by TimeGenerated desc

// Fetch operations
ContainerInstanceLog_CL
| where ContainerGroup_s == "lastfm-sync"
| where Message contains "fetch.page"
| project TimeGenerated, Message
| order by TimeGenerated desc
```

### Container Exit Codes

Understanding container exit codes:

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| 0 | Success | Normal completion |
| 1 | General error | Check logs for error details |
| 2 | Misuse of shell command | Verify command syntax |
| 125 | Container failed to run | Check image and entrypoint |
| 126 | Command cannot execute | Check permissions |
| 127 | Command not found | Verify binary path |
| 128+n | Fatal error signal n | Signal interrupted (e.g., 137 = SIGKILL) |

**Check exit code**:
```bash
az container show \
  --resource-group lastfm-rg \
  --name lastfm-sync \
  --query "containers[0].instanceView.currentState.exitCode" -o tsv
```

### Custom Metrics

LastFMReaderv3 can export custom metrics for monitoring:

**Metrics Available**:
- `fetch_duration_seconds`: Total fetch duration
- `fetch_pages_total`: Number of pages fetched
- `fetch_scrobbles_total`: Number of scrobbles fetched
- `api_calls_total`: Total API calls made
- `api_errors_total`: Number of API errors

**Configure Application Insights** (future enhancement):
```bash
az monitor app-insights component create \
  --app lastfm-insights \
  --location eastus \
  --resource-group lastfm-rg \
  --application-type other
```

---

## Structured Logging Format

LastFMReaderv3 uses structured JSON logging for Log Analytics integration.

### Required Fields

All log entries must include:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `timestamp` | ISO 8601 | UTC timestamp | `2026-01-06T14:30:22Z` |
| `level` | string | Log level | `info`, `debug`, `error` |
| `message` | string | Human-readable message | `fetch.page.complete` |
| `context` | object | Structured context data | See below |

### Optional Fields

Additional fields for enriched logging:

| Field | Type | Description |
|-------|------|-------------|
| `user` | string | Last.fm username |
| `duration_ms` | integer | Operation duration in milliseconds |
| `api_calls` | integer | Number of API calls made |
| `error_details` | string | Error message and stack trace |
| `page` | integer | Current page number |
| `total_pages` | integer | Total pages to fetch |

### Example Log Entries

**Successful fetch**:
```json
{
  "timestamp": "2026-01-06T14:30:22Z",
  "level": "info",
  "message": "fetch.page.complete",
  "context": {
    "user": "alice",
    "page": 5,
    "total_pages": 10,
    "scrobbles_fetched": 200,
    "duration_ms": 1250
  }
}
```

**API error**:
```json
{
  "timestamp": "2026-01-06T14:30:25Z",
  "level": "error",
  "message": "fetch.api.error",
  "context": {
    "user": "alice",
    "error_details": "rate limited: 429 Too Many Requests",
    "retry_after": 60,
    "api_calls": 3
  }
}
```

**Watermark update**:
```json
{
  "timestamp": "2026-01-06T14:30:30Z",
  "level": "info",
  "message": "watermark.update.complete",
  "context": {
    "user": "alice",
    "max_uts": 1704067200,
    "duration_ms": 50
  }
}
```

### Querying Structured Logs

```kusto
// Parse JSON logs
ContainerInstanceLog_CL
| extend LogEntry = parse_json(Message)
| extend 
    Level = tostring(LogEntry.level),
    LogMessage = tostring(LogEntry.message),
    User = tostring(LogEntry.context.user),
    Duration = toint(LogEntry.context.duration_ms)
| where Level == "error"
| project TimeGenerated, User, LogMessage, Duration
| order by TimeGenerated desc

// Calculate average fetch duration
ContainerInstanceLog_CL
| extend LogEntry = parse_json(Message)
| where tostring(LogEntry.message) == "fetch.page.complete"
| extend Duration = toint(LogEntry.context.duration_ms)
| summarize AvgDuration = avg(Duration), Count = count() by bin(TimeGenerated, 1h)
| render timechart
```

---

## Scaling Considerations

### Horizontal Scaling

Run multiple container instances for different users:

```bash
# Deploy for user alice
az container create --name lastfm-sync-alice ...

# Deploy for user bob
az container create --name lastfm-sync-bob ...

# Deploy for user charlie
az container create --name lastfm-sync-charlie ...
```

### Scheduled Execution

Use Azure Logic Apps or Azure Functions to trigger container creation:

```bash
# Create container group with restart policy
az container create \
  --resource-group lastfm-rg \
  --name lastfm-sync-daily \
  --image lastfm-sync:latest \
  --restart-policy OnFailure \
  --schedule "0 0 * * *"  # Daily at midnight (not directly supported - use Logic Apps)
```

**Alternative: Azure Logic Apps**

1. Create Logic App with recurrence trigger (daily)
2. Add "Run Command" action: `az container create ...`
3. Add "Delete Container" action after completion

### Resource Allocation

Optimize CPU and memory based on workload:

| User Size | Scrobbles | CPU | Memory | Cost/Month |
|-----------|-----------|-----|--------|------------|
| Small | < 10K | 0.5 | 0.5 GB | ~$15 |
| Medium | 10K-100K | 1.0 | 1.0 GB | ~$30 |
| Large | > 100K | 2.0 | 2.0 GB | ~$60 |

```bash
az container create \
  --cpu 1.0 \
  --memory 1.0 \
  ...
```

---

## Cost Optimization

### Pay-Per-Use Model

ACI charges only for execution time (per second):

**Pricing** (as of 2026, US East):
- CPU: $0.0000125/vCPU/second (~$0.045/vCPU/hour)
- Memory: $0.0000014/GB/second (~$0.005/GB/hour)

**Example**:
- Container: 0.5 vCPU, 0.5 GB RAM
- Runtime: 10 minutes daily
- Cost: 0.5 × $0.0000125 × 600 × 30 = $0.11/month (CPU)
- Cost: 0.5 × $0.0000014 × 600 × 30 = $0.01/month (Memory)
- **Total: ~$0.12/month**

### Cost-Saving Tips

1. **Use restart policy "Never"** for one-off jobs
2. **Delete container after completion**:
   ```bash
   az container delete --name lastfm-sync --resource-group lastfm-rg --yes
   ```
3. **Use spot instances** (if available)
4. **Optimize fetch frequency** (weekly vs daily)
5. **Use incremental sync** (only fetch new scrobbles)

---

## Troubleshooting

### Authentication Failures

**Symptom**: `failed to get token: [credential authentication failed]`

**Diagnostic Commands**:
```bash
# Verify managed identity is assigned
az container show \
  --resource-group lastfm-rg \
  --name lastfm-sync \
  --query identity

# Check role assignments
az role assignment list \
  --assignee $PRINCIPAL_ID \
  --query "[].{Role:roleDefinitionName, Scope:scope}" -o table

# Test identity access to storage
az storage blob list \
  --account-name lastfmstorage \
  --container-name scrobbles \
  --auth-mode login \
  --only-show-errors
```

**Solutions**:
1. Verify managed identity is assigned to container
2. Check role assignments include "Storage Blob Data Contributor"
3. Wait 5-10 minutes for role propagation
4. Ensure storage account allows managed identity access

**Alternative**: Use connection string temporarily:
```bash
az container create \
  --environment-variables \
    AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=..." \
  --command-line "... --azure-auth connstr"
```

---

### Missing Secrets

**Symptom**: `LASTFM_API_KEY is required`

**Diagnostic Commands**:
```bash
# Verify secret exists in Key Vault
az keyvault secret show \
  --vault-name lastfm-kv \
  --name lastfm-api-key

# Check container environment variables
az container show \
  --resource-group lastfm-rg \
  --name lastfm-sync \
  --query "containers[0].environmentVariables" -o table
```

**Solutions**:
1. Verify secret name matches exactly (case-sensitive)
2. Check Key Vault access policy or RBAC roles
3. Ensure secret value is fetched before deployment:
   ```bash
   LASTFM_API_KEY=$(az keyvault secret show --vault-name kv --name lastfm-api-key --query value -o tsv)
   ```
4. Pass as environment variable:
   ```bash
   --environment-variables LASTFM_API_KEY="$LASTFM_API_KEY"
   ```

---

### Resource Quota Errors

**Symptom**: `Quota exceeded for container group count`

**Diagnostic Commands**:
```bash
# Check current usage
az container list \
  --resource-group lastfm-rg \
  --query "length(@)"

# Check subscription limits
az vm list-usage \
  --location eastus \
  --query "[?name.value=='containerGroups'].{Name:name.localizedValue, Current:currentValue, Limit:limit}" -o table
```

**Solutions**:
1. Delete unused container instances:
   ```bash
   az container delete --name old-container --resource-group lastfm-rg --yes
   ```
2. Request quota increase (Azure Portal → Support)
3. Deploy to different region with available capacity
4. Use Container Apps or AKS for higher scale

---

### Network Issues

**Symptom**: `connection timeout` or `name resolution failed`

**Diagnostic Commands**:
```bash
# Check container network profile
az container show \
  --resource-group lastfm-rg \
  --name lastfm-sync \
  --query "networkProfile"

# Test DNS resolution (from local machine)
nslookup api.last.fm
nslookup lastfmstorage.blob.core.windows.net

# Check NSG rules (if using VNet)
az network nsg rule list \
  --resource-group lastfm-rg \
  --nsg-name lastfm-nsg \
  --query "[].{Name:name, Priority:priority, Direction:direction, Access:access}" -o table
```

**Solutions**:
1. Verify outbound internet access (port 443 required)
2. Check NSG rules allow HTTPS to:
   - `api.last.fm` (Last.fm API)
   - `*.blob.core.windows.net` (Azure Storage)
   - `*.vault.azure.net` (Key Vault)
3. If using private endpoints, verify DNS resolution
4. Check firewall rules on storage account
5. Temporary workaround: Deploy without VNet:
   ```bash
   az container create --name lastfm-sync --image lastfm-sync:latest ...
   # (without --vnet or --subnet flags)
   ```

---

## Related Documentation

- [Configuration Reference](./configuration.md) - Environment variables and settings
- [Docker Documentation](./docker.md) - Building and running containers locally
- [Security Best Practices](./security.md) - Secure deployment patterns
- [Troubleshooting Guide](./troubleshooting.md) - Common issues and solutions

---

## External Resources

- [Azure Container Instances Documentation](https://docs.microsoft.com/en-us/azure/container-instances/)
- [Azure Managed Identities](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/)
- [Azure Key Vault](https://docs.microsoft.com/en-us/azure/key-vault/)
- [Log Analytics Query Language](https://docs.microsoft.com/en-us/azure/azure-monitor/logs/log-query-overview)

---

**Last Updated**: 2026-01-06  
**Feature**: 002-containerization-documentation
