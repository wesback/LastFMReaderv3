# Security Best Practices

Comprehensive security guide for LastFM Reader v3, covering secret management, Azure security features, container security, and operational best practices.

---

## Table of Contents

1. [Overview](#overview)
2. [Secret Management](#secret-management)
3. [Azure Key Vault Integration](#azure-key-vault-integration)
4. [Managed Identity & RBAC](#managed-identity--rbac)
5. [Container Security](#container-security)
6. [Network Security](#network-security)
7. [Storage Security](#storage-security)
8. [Secret Rotation](#secret-rotation)
9. [Audit & Compliance](#audit--compliance)
10. [Security Checklist](#security-checklist)

---

## Overview

### Security Principles

LastFM Reader v3 follows these security principles:

1. **Defense in Depth**: Multiple layers of security controls
2. **Least Privilege**: Minimal permissions for each component
3. **Secret Zero Trust**: Never store secrets in code or plaintext
4. **Audit Everything**: Comprehensive logging of security events
5. **Encryption Everywhere**: Data encrypted at rest and in transit

### Threat Model

| Threat | Mitigation |
|--------|------------|
| API key exposure | Key Vault, managed identity, no hardcoded secrets |
| Unauthorized access | Azure RBAC, managed identity, network policies |
| Container escape | Distroless base, non-root user, read-only filesystem |
| Data exfiltration | Private endpoints, storage firewall, audit logs |
| Supply chain attacks | Multi-stage build, verified base images, minimal dependencies |

---

## Secret Management

### Never Commit Secrets

**CRITICAL**: Never commit these to version control:

```bash
# ❌ NEVER do this:
export LASTFM_API_KEY="abc123..."  # Don't commit shell history
echo "LASTFM_API_KEY=abc123" > .env  # Don't commit .env files
```

### .gitignore Verification

Verify `.gitignore` contains:

```gitignore
# Environment and configuration
.env
.env.*
*.local

# Runtime data
*.log
*.watermark

# Azure-specific (if present)
azure.yaml
.azure/
```

Current `.gitignore` status:

```bash
# Check if .env is ignored
git check-ignore .env
# Expected output: .env

# Verify no secrets are tracked
git ls-files | grep -E '\.(env|local|secret)$'
# Expected output: (empty)
```

### Local Development

For local development, use `.env` files:

```bash
# Copy example template
cp .env.example .env

# Edit with your secrets (never commit this file)
nano .env
```

**Best practices**:
- Store `.env` outside the repository directory (e.g., `~/.config/lastfm-reader/.env`)
- Use a password manager to store API keys
- Rotate API keys regularly (every 90 days)
- Use different API keys for dev/staging/prod

### Environment Variable Security

When passing secrets as environment variables:

```bash
# ✅ Good: Read from secure storage
export LASTFM_API_KEY=$(security find-generic-password -s lastfm-api-key -w)

# ✅ Good: Read from password manager
export LASTFM_API_KEY=$(pass show lastfm/api-key)

# ❌ Bad: Hardcoded in shell
export LASTFM_API_KEY="abc123..."

# ❌ Bad: Passed as command argument (visible in ps)
./lastfm-sync --api-key abc123...
```

---

## Azure Key Vault Integration

### Why Key Vault?

Azure Key Vault provides:
- **Centralized Secret Management**: Single source of truth for secrets
- **Access Control**: Fine-grained RBAC permissions
- **Audit Logs**: Complete history of secret access
- **Encryption**: Hardware-backed encryption (HSM)
- **Versioning**: Track secret changes over time
- **Soft Delete**: Recover accidentally deleted secrets

### Setting Up Key Vault

#### 1. Create Key Vault

```bash
# Create Key Vault
az keyvault create \
  --name lastfm-kv \
  --resource-group lastfm-rg \
  --location eastus \
  --enable-rbac-authorization true \
  --retention-days 90 \
  --enable-purge-protection true

# Enable diagnostic logging
az monitor diagnostic-settings create \
  --name kv-diagnostics \
  --resource $(az keyvault show --name lastfm-kv --query id -o tsv) \
  --workspace /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.OperationalInsights/workspaces/lastfm-logs \
  --logs '[{"category": "AuditEvent", "enabled": true}]'
```

#### 2. Store Secrets

```bash
# Store Last.fm API key
az keyvault secret set \
  --vault-name lastfm-kv \
  --name lastfm-api-key \
  --value "your-api-key-here" \
  --description "Last.fm API key for production" \
  --tags environment=production app=lastfm-reader

# Store storage account connection string (if not using managed identity)
az keyvault secret set \
  --vault-name lastfm-kv \
  --name storage-connection-string \
  --value "DefaultEndpointsProtocol=https;AccountName=..." \
  --description "Azure Storage connection string"

# Verify secrets
az keyvault secret list --vault-name lastfm-kv -o table
```

#### 3. Configure Access Policies

```bash
# Grant your user account access (for testing)
az keyvault set-policy \
  --name lastfm-kv \
  --upn your-email@domain.com \
  --secret-permissions get list set delete

# For RBAC-enabled Key Vault (preferred)
az role assignment create \
  --role "Key Vault Secrets User" \
  --assignee your-email@domain.com \
  --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv
```

### Using Key Vault with Container Instances

#### Option 1: Managed Identity (Recommended)

```bash
# Create managed identity
az identity create \
  --name lastfm-identity \
  --resource-group lastfm-rg

# Get identity details
IDENTITY_ID=$(az identity show --name lastfm-identity --resource-group lastfm-rg --query id -o tsv)
PRINCIPAL_ID=$(az identity show --name lastfm-identity --resource-group lastfm-rg --query principalId -o tsv)

# Grant Key Vault access to managed identity
az role assignment create \
  --role "Key Vault Secrets User" \
  --assignee-object-id $PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv

# Deploy container with managed identity
az container create \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --image ghcr.io/yourusername/lastfm-reader:latest \
  --assign-identity $IDENTITY_ID \
  --secure-environment-variables \
    LASTFM_API_KEY=@Microsoft.KeyVault(SecretUri=https://lastfm-kv.vault.azure.net/secrets/lastfm-api-key/)
```

**How it works**:
1. Container starts with managed identity assigned
2. Azure injects Key Vault references as environment variables
3. Container runtime fetches secrets from Key Vault using managed identity
4. Secrets never stored in container definition or logs

#### Option 2: Service Principal (Alternative)

```bash
# Create service principal
az ad sp create-for-rbac \
  --name lastfm-sp \
  --role "Key Vault Secrets User" \
  --scopes /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv

# Store service principal credentials in Key Vault
az keyvault secret set \
  --vault-name lastfm-kv \
  --name sp-client-id \
  --value "sp-app-id"

az keyvault secret set \
  --vault-name lastfm-kv \
  --name sp-client-secret \
  --value "sp-password"
```

**Note**: Managed identity is preferred over service principal as it eliminates the need to manage credentials.

### Key Vault Best Practices

1. **Enable RBAC**: Use RBAC instead of access policies for better governance
2. **Enable Purge Protection**: Prevent permanent deletion of secrets
3. **Enable Soft Delete**: Recover accidentally deleted secrets (90-day retention)
4. **Use Separate Key Vaults**: One per environment (dev/staging/prod)
5. **Tag Secrets**: Add metadata for organization and compliance
6. **Monitor Access**: Enable diagnostic logs to Log Analytics
7. **Rotate Secrets**: Implement automated rotation (see [Secret Rotation](#secret-rotation))

---

## Managed Identity & RBAC

### Why Managed Identity?

Managed identity eliminates:
- Storing credentials in code or configuration
- Rotating service principal passwords
- Managing certificate expiration
- Risk of credential leakage

### Types of Managed Identity

| Type | Use Case | Assignment |
|------|----------|------------|
| **System-assigned** | Single resource (e.g., one container instance) | Automatically created with resource |
| **User-assigned** | Multiple resources (e.g., multiple container instances) | Created separately, assigned to resources |

### User-Assigned Managed Identity (Recommended)

```bash
# Create managed identity
az identity create \
  --name lastfm-identity \
  --resource-group lastfm-rg \
  --location eastus

# Get identity IDs
IDENTITY_ID=$(az identity show --name lastfm-identity --resource-group lastfm-rg --query id -o tsv)
PRINCIPAL_ID=$(az identity show --name lastfm-identity --resource-group lastfm-rg --query principalId -o tsv)
CLIENT_ID=$(az identity show --name lastfm-identity --resource-group lastfm-rg --query clientId -o tsv)

echo "Identity ID: $IDENTITY_ID"
echo "Principal ID: $PRINCIPAL_ID"
echo "Client ID: $CLIENT_ID"
```

### RBAC Role Assignments

#### Key Vault Access

```bash
# Grant Key Vault Secrets User role (read-only)
az role assignment create \
  --role "Key Vault Secrets User" \
  --assignee-object-id $PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv
```

#### Storage Account Access

```bash
# Grant Storage Blob Data Contributor role
az role assignment create \
  --role "Storage Blob Data Contributor" \
  --assignee-object-id $PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.Storage/storageAccounts/lastfmstore
```

#### Log Analytics Access (Optional)

```bash
# Grant Log Analytics Reader role (for custom queries)
az role assignment create \
  --role "Log Analytics Reader" \
  --assignee-object-id $PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.OperationalInsights/workspaces/lastfm-logs
```

### Least Privilege Principle

**Only grant the minimum required permissions**:

| Resource | Required Role | Justification |
|----------|---------------|---------------|
| Key Vault | Key Vault Secrets User | Read API key (not write/delete) |
| Storage Account | Storage Blob Data Contributor | Write scrobble data (not account-level access) |
| Log Analytics | (none) | Container logs auto-forwarded by ACI |

**Never grant**:
- `Owner` or `Contributor` at subscription/resource group level
- `Key Vault Administrator` (allows deleting vault)
- `Storage Account Contributor` (allows deleting storage account)

### Verifying RBAC Assignments

```bash
# List all role assignments for managed identity
az role assignment list \
  --assignee $PRINCIPAL_ID \
  --all \
  -o table

# Check specific role on Key Vault
az role assignment list \
  --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv \
  --query "[?principalId=='$PRINCIPAL_ID'].{Role:roleDefinitionName,Scope:scope}" \
  -o table
```

---

## Container Security

### Distroless Base Image

LastFM Reader uses Google's `distroless/static:nonroot` base image:

```dockerfile
# Final stage: minimal runtime
FROM gcr.io/distroless/static:nonroot

# Benefits:
# - No shell (prevents reverse shell attacks)
# - No package manager (prevents installing malicious packages)
# - Minimal attack surface (~2MB vs ~100MB Alpine)
# - Non-root user (UID 65532)
```

**Security advantages**:
- **No Shell**: `docker exec` with `/bin/sh` fails (prevents container escape)
- **No Tools**: No `curl`, `wget`, `nc` for data exfiltration
- **Minimal CVEs**: Fewer packages = fewer vulnerabilities
- **Read-Only**: Application can't modify container filesystem

### Non-Root User

Container runs as non-root user (UID 65532):

```bash
# Verify non-root
docker run --rm ghcr.io/yourusername/lastfm-reader:latest /lastfm-sync version
# No permission errors

# Try to write to root filesystem (should fail)
docker run --rm ghcr.io/yourusername/lastfm-reader:latest sh -c "touch /test"
# Error: cannot execute binary file
```

### Read-Only Root Filesystem

Deploy container with read-only filesystem:

```bash
# Docker
docker run --rm --read-only \
  -v lastfm-data:/data \
  ghcr.io/yourusername/lastfm-reader:latest

# Azure Container Instances
az container create \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --image ghcr.io/yourusername/lastfm-reader:latest \
  --azure-file-volume-account-name lastfmstore \
  --azure-file-volume-account-key "..." \
  --azure-file-volume-share-name scrobbles \
  --azure-file-volume-mount-path /data \
  --command-line "/lastfm-sync fetch -u username -o /data -f ndjson" \
  --secure-environment-variables LASTFM_API_KEY=...
```

**Note**: Application only writes to `/data` (mounted volume), not root filesystem.

### Container Image Scanning

Scan images for vulnerabilities:

```bash
# Trivy (recommended)
trivy image ghcr.io/yourusername/lastfm-reader:latest

# Expected output:
# Total: 0 (UNKNOWN: 0, LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0)

# Grype (alternative)
grype ghcr.io/yourusername/lastfm-reader:latest

# Docker Scout (if using Docker Desktop)
docker scout cves ghcr.io/yourusername/lastfm-reader:latest
```

### Secure Build Practices

```dockerfile
# Multi-stage build: separate build and runtime
FROM golang:1.25-alpine AS builder
# Build with static linking
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o lastfm-sync

# Minimal runtime: no build tools
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/lastfm-sync /lastfm-sync
```

**Benefits**:
- Build tools not in runtime image
- Smaller image size (faster pulls, less storage)
- Static binary (no libc dependencies)

---

## Network Security

### Private Endpoints

For production, use private endpoints to avoid public internet exposure:

```bash
# Create virtual network
az network vnet create \
  --name lastfm-vnet \
  --resource-group lastfm-rg \
  --address-prefix 10.0.0.0/16 \
  --subnet-name containers \
  --subnet-prefix 10.0.1.0/24

# Create private endpoint for Key Vault
az network private-endpoint create \
  --name lastfm-kv-pe \
  --resource-group lastfm-rg \
  --vnet-name lastfm-vnet \
  --subnet containers \
  --private-connection-resource-id $(az keyvault show --name lastfm-kv --query id -o tsv) \
  --group-id vault \
  --connection-name lastfm-kv-connection

# Create private endpoint for Storage Account
az network private-endpoint create \
  --name lastfm-storage-pe \
  --resource-group lastfm-rg \
  --vnet-name lastfm-vnet \
  --subnet containers \
  --private-connection-resource-id $(az storage account show --name lastfmstore --query id -o tsv) \
  --group-id blob \
  --connection-name lastfm-storage-connection

# Deploy container in VNet
az container create \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --image ghcr.io/yourusername/lastfm-reader:latest \
  --vnet lastfm-vnet \
  --subnet containers \
  --assign-identity $IDENTITY_ID \
  --secure-environment-variables LASTFM_API_KEY=@Microsoft.KeyVault(...)
```

### Network Policies

Restrict outbound traffic using Azure Firewall or Network Security Groups:

```bash
# Create NSG
az network nsg create \
  --name lastfm-nsg \
  --resource-group lastfm-rg

# Allow outbound to Last.fm API
az network nsg rule create \
  --name allow-lastfm-api \
  --nsg-name lastfm-nsg \
  --resource-group lastfm-rg \
  --priority 100 \
  --direction Outbound \
  --destination-address-prefixes "*.last.fm" \
  --destination-port-ranges 443 \
  --protocol Tcp \
  --access Allow

# Allow outbound to Azure services
az network nsg rule create \
  --name allow-azure-services \
  --nsg-name lastfm-nsg \
  --resource-group lastfm-rg \
  --priority 110 \
  --direction Outbound \
  --destination-address-prefixes AzureCloud \
  --destination-port-ranges 443 \
  --protocol Tcp \
  --access Allow

# Deny all other outbound traffic
az network nsg rule create \
  --name deny-all-outbound \
  --nsg-name lastfm-nsg \
  --resource-group lastfm-rg \
  --priority 1000 \
  --direction Outbound \
  --destination-address-prefixes "*" \
  --destination-port-ranges "*" \
  --protocol "*" \
  --access Deny

# Associate NSG with subnet
az network vnet subnet update \
  --name containers \
  --vnet-name lastfm-vnet \
  --resource-group lastfm-rg \
  --network-security-group lastfm-nsg
```

### TLS/SSL

All Azure services use TLS 1.2+ by default:

- **Last.fm API**: HTTPS only (`https://ws.audioscrobbler.com`)
- **Azure Key Vault**: TLS 1.2+ required
- **Azure Storage**: TLS 1.2+ enforced

Verify TLS version:

```bash
# Check Last.fm API
curl -v https://ws.audioscrobbler.com/2.0/ 2>&1 | grep "TLS"
# Expected: TLSv1.2 or TLSv1.3

# Check Azure Key Vault
az keyvault show --name lastfm-kv --query "properties.networkAcls.minTlsVersion"
# Expected: "1.2"
```

---

## Storage Security

### Storage Account Firewall

Restrict storage account access:

```bash
# Disable public access
az storage account update \
  --name lastfmstore \
  --resource-group lastfm-rg \
  --allow-blob-public-access false

# Enable firewall (deny all by default)
az storage account update \
  --name lastfmstore \
  --resource-group lastfm-rg \
  --default-action Deny

# Allow specific VNet
az storage account network-rule add \
  --account-name lastfmstore \
  --resource-group lastfm-rg \
  --vnet-name lastfm-vnet \
  --subnet containers

# Allow trusted Azure services (for managed identity)
az storage account update \
  --name lastfmstore \
  --resource-group lastfm-rg \
  --bypass AzureServices
```

### Shared Access Signature (SAS) Security

If using SAS tokens instead of managed identity:

```bash
# Generate SAS token with minimal permissions
az storage container generate-sas \
  --account-name lastfmstore \
  --name scrobbles \
  --permissions rw \
  --expiry 2024-12-31T23:59:59Z \
  --https-only \
  --ip 203.0.113.0/24  # Restrict to specific IP range

# Best practices:
# - Use HTTPS only (--https-only)
# - Set short expiry (hours/days, not years)
# - Minimal permissions (rw, not rwdl)
# - Restrict to specific IP ranges (--ip)
# - Use account SAS, not service SAS (more secure)
```

**Prefer managed identity over SAS tokens** to avoid credential management.

### Data Encryption

Data encrypted at rest and in transit:

```bash
# Verify encryption at rest
az storage account show \
  --name lastfmstore \
  --resource-group lastfm-rg \
  --query "encryption.services"

# Expected output:
# {
#   "blob": { "enabled": true },
#   "file": { "enabled": true }
# }

# Enable infrastructure encryption (double encryption)
az storage account create \
  --name lastfmstore \
  --resource-group lastfm-rg \
  --require-infrastructure-encryption true

# Use customer-managed keys (optional)
az storage account update \
  --name lastfmstore \
  --resource-group lastfm-rg \
  --encryption-key-source Microsoft.Keyvault \
  --encryption-key-vault https://lastfm-kv.vault.azure.net \
  --encryption-key-name storage-encryption-key
```

---

## Secret Rotation

### Last.fm API Key Rotation

Rotate API keys every 90 days:

```bash
# 1. Generate new API key on Last.fm website
# https://www.last.fm/api/account/create

# 2. Store new key in Key Vault with version
az keyvault secret set \
  --vault-name lastfm-kv \
  --name lastfm-api-key \
  --value "new-api-key-here" \
  --description "Rotated on $(date +%Y-%m-%d)" \
  --tags rotation-date=$(date +%Y-%m-%d)

# 3. Test new key
docker run --rm \
  -e LASTFM_API_KEY="new-api-key" \
  ghcr.io/yourusername/lastfm-reader:latest \
  /lastfm-sync fetch -u testuser -o /dev/null --limit 1

# 4. Update container instances
az container restart --name lastfm-sync --resource-group lastfm-rg

# 5. Revoke old API key on Last.fm website
# 6. Delete old Key Vault secret version (optional)
```

### Storage Account Key Rotation

If using storage account keys (not recommended, use managed identity):

```bash
# Rotate key1
az storage account keys renew \
  --account-name lastfmstore \
  --resource-group lastfm-rg \
  --key key1

# Update Key Vault with new key
az keyvault secret set \
  --vault-name lastfm-kv \
  --name storage-account-key \
  --value $(az storage account keys list --account-name lastfmstore --resource-group lastfm-rg --query "[0].value" -o tsv)

# Restart containers
az container restart --name lastfm-sync --resource-group lastfm-rg

# Wait 24 hours, then rotate key2
az storage account keys renew \
  --account-name lastfmstore \
  --resource-group lastfm-rg \
  --key key2
```

### Automated Rotation with Azure Functions

```bash
# Create Azure Function for automated rotation
# (requires Azure Functions deployment, see Azure Functions documentation)

# Example: Rotate secrets every 90 days
# - Azure Function triggered by Timer (0 0 1 * * * - 1st of month)
# - Function generates new API key (if API supports programmatic rotation)
# - Function updates Key Vault
# - Function restarts container instances
# - Function sends notification to admins
```

### Rotation Monitoring

```bash
# List Key Vault secret versions with dates
az keyvault secret list-versions \
  --vault-name lastfm-kv \
  --name lastfm-api-key \
  --query "[].{Version:id, Created:attributes.created, Updated:attributes.updated}" \
  -o table

# Alert on secrets older than 90 days
# (Configure Azure Monitor alert rule)
az monitor metrics alert create \
  --name secret-rotation-alert \
  --resource-group lastfm-rg \
  --scopes $(az keyvault show --name lastfm-kv --query id -o tsv) \
  --condition "count secrets > 0 where age > 90 days" \
  --description "Alert when secrets are older than 90 days"
```

---

## Audit & Compliance

### Enable Diagnostic Logs

```bash
# Create Log Analytics workspace
az monitor log-analytics workspace create \
  --name lastfm-logs \
  --resource-group lastfm-rg \
  --location eastus

# Enable Key Vault audit logs
az monitor diagnostic-settings create \
  --name kv-audit-logs \
  --resource $(az keyvault show --name lastfm-kv --query id -o tsv) \
  --workspace $(az monitor log-analytics workspace show --name lastfm-logs --resource-group lastfm-rg --query id -o tsv) \
  --logs '[
    {
      "category": "AuditEvent",
      "enabled": true,
      "retentionPolicy": {
        "enabled": true,
        "days": 90
      }
    }
  ]'

# Enable Storage Account audit logs
az monitor diagnostic-settings create \
  --name storage-audit-logs \
  --resource $(az storage account show --name lastfmstore --query id -o tsv) \
  --workspace $(az monitor log-analytics workspace show --name lastfm-logs --resource-group lastfm-rg --query id -o tsv) \
  --logs '[
    {
      "category": "StorageRead",
      "enabled": true
    },
    {
      "category": "StorageWrite",
      "enabled": true
    }
  ]'
```

### Query Audit Logs

```kql
// Key Vault secret access
AzureDiagnostics
| where ResourceProvider == "MICROSOFT.KEYVAULT"
| where OperationName == "SecretGet"
| project TimeGenerated, CallerIdentity=identity_claim_upn_s, SecretName=id_s, ResultDescription
| order by TimeGenerated desc
| take 100

// Failed Key Vault access attempts
AzureDiagnostics
| where ResourceProvider == "MICROSOFT.KEYVAULT"
| where OperationName == "SecretGet"
| where ResultSignature == "Forbidden"
| project TimeGenerated, CallerIdentity=identity_claim_upn_s, SecretName=id_s, ResultDescription
| order by TimeGenerated desc

// Storage Account access
StorageBlobLogs
| where OperationName in ("PutBlob", "GetBlob", "DeleteBlob")
| project TimeGenerated, AccountName, ContainerName, Uri, CallerIpAddress, AuthenticationType
| order by TimeGenerated desc
| take 100
```

### Compliance Reports

Generate compliance reports:

```bash
# List all RBAC assignments
az role assignment list \
  --all \
  --query "[?scope contains(@, 'lastfm-rg')].{Principal:principalName, Role:roleDefinitionName, Scope:scope}" \
  -o table > rbac-report.txt

# List all Key Vault secrets
az keyvault secret list \
  --vault-name lastfm-kv \
  --query "[].{Name:name, Enabled:attributes.enabled, Created:attributes.created, Updated:attributes.updated}" \
  -o table > secrets-report.txt

# List all storage account configurations
az storage account show \
  --name lastfmstore \
  --resource-group lastfm-rg \
  --query "{Name:name, Encryption:encryption, NetworkRules:networkRuleSet, AllowBlobPublicAccess:allowBlobPublicAccess}" \
  -o json > storage-report.json
```

### Security Alerts

Configure security alerts:

```bash
# Alert on Key Vault secret access by unknown identity
az monitor metrics alert create \
  --name unknown-secret-access \
  --resource-group lastfm-rg \
  --scopes $(az keyvault show --name lastfm-kv --query id -o tsv) \
  --condition "count operations where identity != 'known-identity'" \
  --description "Alert when Key Vault is accessed by unknown identity" \
  --action email admin@example.com

# Alert on storage account public access enabled
az monitor activity-log alert create \
  --name storage-public-access-enabled \
  --resource-group lastfm-rg \
  --scope $(az storage account show --name lastfmstore --query id -o tsv) \
  --condition category=Administrative and operationName=Microsoft.Storage/storageAccounts/write \
  --action email admin@example.com
```

---

## Security Checklist

### Pre-Deployment

- [ ] **Secrets**: All secrets stored in Azure Key Vault (not .env or code)
- [ ] **.gitignore**: Verify `.env`, `.env.*`, `*.local` are ignored
- [ ] **RBAC**: Managed identity has only required permissions (least privilege)
- [ ] **Network**: Private endpoints or VNet integration configured
- [ ] **Storage**: Public blob access disabled, firewall enabled
- [ ] **Encryption**: Storage account encryption at rest enabled
- [ ] **Audit Logs**: Diagnostic settings enabled for Key Vault and Storage

### Deployment

- [ ] **Container Image**: Scanned for vulnerabilities (Trivy/Grype)
- [ ] **Base Image**: Using distroless/static:nonroot (no shell, non-root)
- [ ] **Read-Only FS**: Container deployed with read-only root filesystem
- [ ] **Secrets**: Using Key Vault references (not plaintext env vars)
- [ ] **Identity**: Managed identity assigned to container instance
- [ ] **Network**: Container deployed in VNet with NSG rules

### Post-Deployment

- [ ] **Access Logs**: Review Key Vault access logs for anomalies
- [ ] **Rotation**: Set calendar reminder for API key rotation (90 days)
- [ ] **Monitoring**: Alerts configured for failed authentication attempts
- [ ] **Compliance**: RBAC and storage reports generated and reviewed
- [ ] **Testing**: Verify container cannot write to root filesystem
- [ ] **Testing**: Verify container cannot access unauthorized resources

### Ongoing

- [ ] **Monthly**: Review Key Vault access logs
- [ ] **Quarterly**: Rotate Last.fm API key
- [ ] **Quarterly**: Review and update RBAC assignments
- [ ] **Semi-Annually**: Run container image vulnerability scan
- [ ] **Annually**: Review and update network security policies

---

## Quick Reference

### Secure Deployment Command

```bash
# Fully secure Azure Container Instances deployment
az container create \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --image ghcr.io/yourusername/lastfm-reader:latest \
  --assign-identity /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/lastfm-identity \
  --vnet lastfm-vnet \
  --subnet containers \
  --azure-file-volume-account-name lastfmstore \
  --azure-file-volume-share-name scrobbles \
  --azure-file-volume-mount-path /data \
  --secure-environment-variables \
    LASTFM_API_KEY=@Microsoft.KeyVault(SecretUri=https://lastfm-kv.vault.azure.net/secrets/lastfm-api-key/) \
  --log-analytics-workspace /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.OperationalInsights/workspaces/lastfm-logs \
  --log-analytics-workspace-key "workspace-key" \
  --restart-policy Never \
  --cpu 1 \
  --memory 0.5
```

### Security Validation

```bash
# Verify container security
docker run --rm ghcr.io/yourusername/lastfm-reader:latest /lastfm-sync version
trivy image ghcr.io/yourusername/lastfm-reader:latest

# Verify Azure RBAC
az role assignment list --assignee $PRINCIPAL_ID --all -o table

# Verify Key Vault access
az keyvault secret show --vault-name lastfm-kv --name lastfm-api-key

# Verify storage account security
az storage account show --name lastfmstore --query "{PublicAccess:allowBlobPublicAccess, Encryption:encryption.services.blob.enabled}"
```

---

## Additional Resources

- [Azure Key Vault Best Practices](https://docs.microsoft.com/azure/key-vault/general/best-practices)
- [Azure Storage Security Guide](https://docs.microsoft.com/azure/storage/common/storage-security-guide)
- [Azure Container Security](https://docs.microsoft.com/azure/container-instances/container-instances-image-security)
- [Azure RBAC Documentation](https://docs.microsoft.com/azure/role-based-access-control/)
- [OWASP Container Security](https://owasp.org/www-project-docker-top-10/)

---

*For deployment instructions, see [docs/azure-deployment.md](azure-deployment.md).*  
*For configuration reference, see [docs/configuration.md](configuration.md).*
