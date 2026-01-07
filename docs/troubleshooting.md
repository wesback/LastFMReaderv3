# Troubleshooting Guide

Common issues and solutions for LastFM Reader v3, covering local development, Docker, Docker Compose, and Azure Container Instances deployments.

---

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Configuration Issues](#configuration-issues)
3. [Docker Issues](#docker-issues)
4. [Docker Compose Issues](#docker-compose-issues)
5. [Azure Deployment Issues](#azure-deployment-issues)
6. [API & Network Issues](#api--network-issues)
7. [Data & Storage Issues](#data--storage-issues)
8. [Debugging Commands](#debugging-commands)
9. [Validation Checklist](#validation-checklist)
10. [FAQ](#faq)

---

## Quick Diagnostics

### System Health Check

```bash
# Check Docker installation
docker --version
docker compose version

# Check Go installation (for building from source)
go version

# Check Azure CLI (for Azure deployments)
az --version

# Check repository structure
ls -la .env.example Dockerfile docker-compose.yml
ls -la docs/ scripts/ azure/
```

### Common Quick Fixes

```bash
# Fix: Permission denied errors
sudo chmod +x scripts/*.sh azure/*.sh

# Fix: Stale Docker images
docker compose down --rmi all
docker system prune -af

# Fix: Corrupt .env file
cp .env.example .env
nano .env  # Re-add your LASTFM_API_KEY

# Fix: Missing directories
mkdir -p ~/.lastfm/state data/
```

---

## Configuration Issues

### Error: `LASTFM_API_KEY is required`

**Symptom:**
```
Error: validate config: LASTFM_API_KEY is required
```

**Cause:** API key not set in environment variable or `.env` file.

**Solution:**

1. **Check if .env exists:**
   ```bash
   ls -la .env
   # If missing: cp .env.example .env
   ```

2. **Verify LASTFM_API_KEY is set:**
   ```bash
   grep LASTFM_API_KEY .env
   # Should show: LASTFM_API_KEY=your-actual-key
   ```

3. **Get API key from Last.fm:**
   - Visit https://www.last.fm/api/account/create
   - Create new API application
   - Copy API key to `.env` file

4. **Test configuration:**
   ```bash
   # Local test
   export LASTFM_API_KEY="your-key"
   ./dist/lastfm-sync fetch --user testuser --dry-run
   
   # Docker Compose test
   docker compose run --rm lastfm-sync fetch --user testuser --dry-run
   ```

**Validation:**
```bash
# Verify API key works
docker run --rm -e LASTFM_API_KEY="your-key" lastfm-sync:latest \
  fetch --user testuser --limit 1 --dry-run
# Expected: Shows user's total scrobble count
```

---

### Error: `invalid value for --qps`

**Symptom:**
```
Error: invalid value "10" for flag --qps: must be between 1 and 5
```

**Cause:** QPS (queries per second) exceeds Last.fm API limit.

**Solution:**

```bash
# Use recommended QPS (default: 3)
lastfm-sync fetch --user alice --qps 3

# Or set via environment
export LASTFM_QPS=3
```

**Last.fm API limits:**
- **Maximum:** 5 QPS
- **Recommended:** 3 QPS (conservative, avoids throttling)
- **Minimum:** 1 QPS (for rate-limited scenarios)

---

### Error: `failed to parse timeout`

**Symptom:**
```
Error: failed to parse timeout "30": time: missing unit in duration
```

**Cause:** Timeout specified without time unit.

**Solution:**

```bash
# ✅ Correct: Include time unit
export LASTFM_TIMEOUT="30s"   # seconds
export LASTFM_TIMEOUT="1m"    # minutes
export LASTFM_TIMEOUT="500ms" # milliseconds

# ❌ Incorrect: Missing unit
export LASTFM_TIMEOUT="30"
```

**Valid time units:** `ns`, `us`, `ms`, `s`, `m`, `h`

---

## Docker Issues

### Error: `permission denied while trying to connect to Docker daemon`

**Symptom:**
```
permission denied while trying to connect to the Docker daemon socket
```

**Cause:** Current user not in `docker` group or Docker daemon not running.

**Solution:**

1. **Start Docker daemon:**
   ```bash
   # Linux (systemd)
   sudo systemctl start docker
   sudo systemctl enable docker
   
   # macOS/Windows
   # Start Docker Desktop application
   ```

2. **Add user to docker group (Linux):**
   ```bash
   sudo usermod -aG docker $USER
   newgrp docker  # Activate changes without logout
   
   # Verify
   docker ps
   ```

3. **Test Docker access:**
   ```bash
   docker run --rm hello-world
   ```

---

### Error: `Docker build fails with "failed to compute cache key"`

**Symptom:**
```
ERROR: failed to solve: failed to compute cache key: failed to calculate checksum
```

**Cause:** Missing files referenced in Dockerfile (e.g., `go.mod`, `go.sum`).

**Solution:**

1. **Verify required files exist:**
   ```bash
   ls -la go.mod go.sum cmd/ internal/
   # All should exist
   ```

2. **Check .dockerignore:**
   ```bash
   cat .dockerignore
   # Should NOT ignore: go.mod, go.sum, cmd/, internal/
   # Should ignore: .env, dist/, *.test
   ```

3. **Clean and rebuild:**
   ```bash
   docker buildx prune -af
   docker build --no-cache -t lastfm-sync .
   ```

---

### Error: `exec format error`

**Symptom:**
```
standard_init_linux.go:228: exec user process caused: exec format error
```

**Cause:** Container built for different architecture (e.g., ARM vs AMD64).

**Solution:**

```bash
# Build for your platform
docker build --platform linux/amd64 -t lastfm-sync .

# Or multi-platform build
docker buildx build --platform linux/amd64,linux/arm64 -t lastfm-sync .

# Verify binary architecture
docker run --rm --entrypoint /bin/sh lastfm-sync -c "file /app/lastfm-sync"
```

---

### Error: `Container won't start / exits immediately`

**Symptom:**
```
docker: Error response from daemon: OCI runtime create failed: container_linux.go:380: starting container process caused: exec: "/bin/sh": stat /bin/sh: no such file or directory: unknown.
```

**Cause:** Distroless image has no shell (by design for security).

**Solution:**

This is **expected behavior** with distroless images. The container is working correctly.

**Run application directly:**
```bash
# ✅ Correct: Run application
docker run --rm lastfm-sync fetch --help

# ❌ Incorrect: Try to open shell
docker run --rm -it lastfm-sync /bin/sh
# Error: /bin/sh doesn't exist in distroless
```

**Debugging without shell:**
```bash
# View logs
docker logs <container-id>

# Inspect container
docker inspect <container-id>

# Check exit code
docker inspect --format='{{.State.ExitCode}}' <container-id>
```

**Exit code meanings:**
- `0`: Success
- `1`: General error (check logs)
- `2`: Configuration error (missing API key, invalid flags)
- `137`: Killed by OOM (out of memory)
- `143`: Killed by SIGTERM (graceful shutdown)

---

## Docker Compose Issues

### Error: `.env file not found`

**Symptom:**
```bash
./scripts/dev-up.sh
Error: .env file not found
```

**Cause:** Missing `.env` file in project root.

**Solution:**

```bash
# Create .env from template
cp .env.example .env

# Edit and add your API key
nano .env

# Verify
cat .env | grep LASTFM_API_KEY
# Should show: LASTFM_API_KEY=your-key-here
```

**Helper script does this automatically:**
```bash
./scripts/dev-up.sh
# Prompts if .env is missing and guides you through setup
```

---

### Error: `data: permission denied`

**Symptom:**
```
Error: failed to open file: open /data/alice.ndjson: permission denied
```

**Cause:** Container user (UID 65532) cannot write to mounted volume.

**Solution:**

1. **Fix directory permissions:**
   ```bash
   # Create data directory with correct permissions
   mkdir -p data
   chmod 777 data  # Or set ownership to UID 65532
   
   # Alternative: Set specific ownership
   sudo chown -R 65532:65532 data/
   ```

2. **For Docker Compose:**
   ```yaml
   # Already configured in docker-compose.yml
   volumes:
     - ./data:/data  # Maps to local data/ directory
   ```

3. **Verify fix:**
   ```bash
   ls -ld data/
   # Should show: drwxrwxrwx or owned by UID 65532
   
   # Test write
   docker compose run --rm lastfm-sync \
     fetch --user testuser --out-path /data/test.ndjson --limit 1
   ```

---

### Error: `docker compose command not found`

**Symptom:**
```
bash: docker compose: command not found
```

**Cause:** Docker Compose V2 not installed or using old `docker-compose` command.

**Solution:**

```bash
# Try Docker Compose V1 (hyphenated)
docker-compose --version

# If V1 works but you want V2:
# Update Docker Desktop (includes Compose V2)
# Or install Compose V2 plugin:
sudo apt-get update
sudo apt-get install docker-compose-plugin

# Verify
docker compose version
# Expected: Docker Compose version v2.x.x
```

**Use helper scripts (work with both versions):**
```bash
./scripts/dev-up.sh    # Auto-detects V1/V2
./scripts/dev-down.sh
```

---

### Error: `Conflicting container name`

**Symptom:**
```
Error: Conflict. The container name "/lastfm-sync" is already in use
```

**Cause:** Previous container still exists.

**Solution:**

```bash
# Stop and remove existing container
docker compose down

# Or force removal
docker rm -f lastfm-sync

# Clean up everything
docker compose down --volumes --remove-orphans

# Restart
docker compose up
```

---

## Azure Deployment Issues

### Authentication Failures

#### Error: `az: command not found`

**Solution:**

```bash
# Install Azure CLI
# macOS
brew install azure-cli

# Linux (Debian/Ubuntu)
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Verify
az --version
```

#### Error: `Please run 'az login' to setup account`

**Solution:**

```bash
# Login to Azure
az login

# If browser doesn't open automatically, use device code
az login --use-device-code

# Verify authentication
az account show

# List available subscriptions
az account list --output table

# Set default subscription
az account set --subscription "your-subscription-id"
```

#### Error: `AADSTS700016: Application with identifier was not found`

**Cause:** Service principal or managed identity not properly configured.

**Solution:**

1. **For user authentication:**
   ```bash
   az login
   az account show  # Verify logged in
   ```

2. **For service principal:**
   ```bash
   # Verify service principal exists
   az ad sp show --id <service-principal-id>
   
   # Login with service principal
   az login --service-principal \
     --username <app-id> \
     --password <password> \
     --tenant <tenant-id>
   ```

3. **For managed identity (on Azure VM/ACI):**
   ```bash
   # Verify managed identity assigned
   az container show \
     --name lastfm-sync \
     --resource-group lastfm-rg \
     --query identity
   
   # Should show: "type": "UserAssigned" or "SystemAssigned"
   ```

---

### Missing Secrets / Key Vault Issues

#### Error: `The user, group or application does not have secrets get permission`

**Symptom:**
```
ErrorCode: Forbidden
Message: The user, group or application 'appid=...' does not have secrets get permission on key vault 'lastfm-kv'
```

**Cause:** Managed identity or user lacks Key Vault permissions.

**Solution:**

```bash
# Get managed identity principal ID
PRINCIPAL_ID=$(az identity show \
  --name lastfm-identity \
  --resource-group lastfm-rg \
  --query principalId -o tsv)

# Grant Key Vault Secrets User role
az role assignment create \
  --role "Key Vault Secrets User" \
  --assignee-object-id $PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv

# Verify role assignment
az role assignment list \
  --assignee $PRINCIPAL_ID \
  --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.KeyVault/vaults/lastfm-kv \
  -o table

# Test secret access
az keyvault secret show \
  --vault-name lastfm-kv \
  --name lastfm-api-key \
  --query value -o tsv
```

#### Error: `Secret not found`

**Symptom:**
```
ErrorCode: SecretNotFound
Message: A secret with (name/id) lastfm-api-key was not found in this key vault
```

**Solution:**

```bash
# List all secrets
az keyvault secret list --vault-name lastfm-kv -o table

# Create missing secret
az keyvault secret set \
  --vault-name lastfm-kv \
  --name lastfm-api-key \
  --value "your-api-key-here"

# Verify
az keyvault secret show \
  --vault-name lastfm-kv \
  --name lastfm-api-key \
  --query value -o tsv
```

#### Error: `Key Vault reference format incorrect`

**Symptom:**
```
Invalid secure environment variable format
```

**Solution:**

**Correct format:**
```bash
--secure-environment-variables \
  LASTFM_API_KEY=@Microsoft.KeyVault(SecretUri=https://lastfm-kv.vault.azure.net/secrets/lastfm-api-key/)
```

**Common mistakes:**
```bash
# ❌ Missing @
LASTFM_API_KEY=Microsoft.KeyVault(...)

# ❌ Missing trailing slash
SecretUri=https://lastfm-kv.vault.azure.net/secrets/lastfm-api-key

# ❌ Wrong secret name
SecretUri=https://lastfm-kv.vault.azure.net/secrets/api-key/
```

---

### Resource Quota Errors

#### Error: `Operation results in exceeding quota limits of Core`

**Symptom:**
```
Status: QuotaExceeded
Code: QuotaExceeded
Message: Operation results in exceeding quota limits of Core. Maximum allowed: 20, Current in use: 18, Additional requested: 4.
```

**Cause:** Subscription CPU quota exceeded.

**Solution:**

1. **Check current quota usage:**
   ```bash
   az vm list-usage \
     --location eastus \
     --query "[?name.value=='cores'].{Name:name.localName,Current:currentValue,Limit:limit}" \
     -o table
   ```

2. **Reduce container CPU allocation:**
   ```bash
   # Instead of --cpu 4, use --cpu 1 or 2
   az container create \
     --name lastfm-sync \
     --cpu 1 \
     --memory 0.5 \
     ...
   ```

3. **Delete unused resources:**
   ```bash
   # List all container instances
   az container list -o table
   
   # Delete unused containers
   az container delete \
     --name old-container \
     --resource-group lastfm-rg \
     --yes
   ```

4. **Request quota increase:**
   - Azure Portal → Subscriptions → Usage + quotas
   - Request increase for "Total Regional vCPUs"
   - Or use Azure CLI:
   ```bash
   az support tickets create \
     --title "Request vCPU quota increase" \
     --quota-change ...
   ```

#### Error: `The subscription is not registered to use namespace 'Microsoft.ContainerInstance'`

**Solution:**

```bash
# Register resource provider
az provider register --namespace Microsoft.ContainerInstance

# Wait for registration (takes 2-5 minutes)
az provider show \
  --namespace Microsoft.ContainerInstance \
  --query "registrationState"
# Expected: "Registered"

# List all registered providers
az provider list --query "[?registrationState=='Registered']" -o table
```

---

### Network Issues

#### Error: `connection timeout`

**Symptom:**
```
Error: failed to fetch scrobbles: Get "https://ws.audioscrobbler.com/2.0/": dial tcp: lookup ws.audioscrobbler.com: i/o timeout
```

**Cause:** Network connectivity issues or DNS resolution failure.

**Diagnostic steps:**

```bash
# 1. Check internet connectivity
ping -c 3 8.8.8.8

# 2. Check DNS resolution
nslookup ws.audioscrobbler.com
dig ws.audioscrobbler.com

# 3. Check Last.fm API accessibility
curl -v https://ws.audioscrobbler.com/2.0/?method=user.getinfo&user=rj&api_key=test&format=json

# 4. Check Azure container network
az container show \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --query "ipAddress"
```

**Solutions:**

1. **For VNet-integrated containers:**
   ```bash
   # Verify NSG rules allow outbound HTTPS
   az network nsg rule list \
     --nsg-name lastfm-nsg \
     --resource-group lastfm-rg \
     --query "[?direction=='Outbound']" \
     -o table
   
   # Add rule if missing
   az network nsg rule create \
     --name allow-https-outbound \
     --nsg-name lastfm-nsg \
     --resource-group lastfm-rg \
     --priority 100 \
     --direction Outbound \
     --destination-port-ranges 443 \
     --protocol Tcp \
     --access Allow
   ```

2. **For private endpoints:**
   ```bash
   # Verify private DNS zone linked to VNet
   az network private-dns link vnet list \
     --resource-group lastfm-rg \
     --zone-name privatelink.vaultcore.azure.net \
     -o table
   ```

3. **Test connectivity from container:**
   ```bash
   # Using Docker
   docker run --rm busybox ping -c 3 ws.audioscrobbler.com
   
   # Check container logs for network errors
   az container logs \
     --name lastfm-sync \
     --resource-group lastfm-rg
   ```

#### Error: `x509: certificate signed by unknown authority`

**Symptom:**
```
Error: x509: certificate signed by unknown authority
```

**Cause:** Corporate proxy or firewall intercepting SSL certificates.

**Solution:**

```bash
# Option 1: Use system CA certificates (if behind corporate proxy)
# Add corporate CA cert to Docker build
COPY corporate-ca.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates

# Option 2: Bypass proxy (not recommended for production)
export HTTP_PROXY=""
export HTTPS_PROXY=""

# Option 3: Configure Go to use system certificates
export SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
```

---

### Container Logs & Monitoring

#### Viewing Logs

```bash
# Real-time logs
az container logs \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --follow

# Logs from previous run
az container logs \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --previous

# Export logs to file
az container logs \
  --name lastfm-sync \
  --resource-group lastfm-rg > logs.txt
```

#### Log Analytics Queries

```kql
// Container logs (last 24 hours)
ContainerInstanceLog_CL
| where TimeGenerated > ago(24h)
| where ContainerGroup_s == "lastfm-sync"
| project TimeGenerated, Message, ContainerName_s
| order by TimeGenerated desc
| take 100

// Error logs only
ContainerInstanceLog_CL
| where Message contains "Error" or Message contains "error"
| project TimeGenerated, Message
| order by TimeGenerated desc

// Performance metrics
AzureMetrics
| where ResourceProvider == "MICROSOFT.CONTAINERINSTANCE"
| where MetricName in ("CpuUsage", "MemoryUsage")
| project TimeGenerated, MetricName, Average, Maximum
| order by TimeGenerated desc
```

#### Container Status

```bash
# Check container state
az container show \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --query "containers[0].instanceView.currentState"

# Possible states:
# - Waiting: Starting up
# - Running: Active
# - Terminated: Stopped (check exitCode)

# Check exit code
az container show \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --query "containers[0].instanceView.previousState.exitCode"
```

**Exit codes:**
- `0`: Success
- `1`: Application error (check logs)
- `2`: Configuration error (missing API key, invalid flags)
- `137`: Killed by OOM (out of memory - increase --memory)
- `143`: Killed by SIGTERM (graceful shutdown)

---

## API & Network Issues

### Error: `HTTP 429 Too Many Requests`

**Symptom:**
```
Error: Last.fm API returned 429: Rate limit exceeded
```

**Cause:** Exceeding Last.fm API rate limit (5 requests/second).

**Solution:**

```bash
# Reduce QPS (default: 3)
export LASTFM_QPS=2
lastfm-sync fetch --user alice --qps 2

# Or wait for rate limit to reset (usually 60 seconds)
sleep 60
lastfm-sync fetch --user alice
```

**Best practices:**
- Use default QPS of 3 (conservative)
- Don't run multiple instances for same user simultaneously
- Implement exponential backoff (already handled by application)

---

### Error: `HTTP 403 Forbidden`

**Symptom:**
```
Error: Last.fm API returned 403: Invalid API key
```

**Cause:** Incorrect or revoked API key.

**Solution:**

1. **Verify API key:**
   ```bash
   # Check API key is set
   echo $LASTFM_API_KEY
   
   # Test API key directly
   curl "https://ws.audioscrobbler.com/2.0/?method=user.getinfo&user=rj&api_key=$LASTFM_API_KEY&format=json"
   ```

2. **Get new API key:**
   - Visit https://www.last.fm/api/account/create
   - Create new application
   - Update `.env` with new key

3. **Update Key Vault (for Azure):**
   ```bash
   az keyvault secret set \
     --vault-name lastfm-kv \
     --name lastfm-api-key \
     --value "new-api-key"
   
   # Restart container to pick up new secret
   az container restart \
     --name lastfm-sync \
     --resource-group lastfm-rg
   ```

---

### Error: `context deadline exceeded`

**Symptom:**
```
Error: Get "https://ws.audioscrobbler.com/2.0/": context deadline exceeded
```

**Cause:** HTTP request timeout (default: 15s).

**Solution:**

```bash
# Increase timeout for slow networks
export LASTFM_TIMEOUT="30s"
lastfm-sync fetch --user alice --timeout 30s

# Or check network connectivity
ping ws.audioscrobbler.com
curl -w "Time: %{time_total}s\n" "https://ws.audioscrobbler.com/2.0/?method=user.getinfo&user=rj&api_key=test&format=json"
```

---

## Data & Storage Issues

### Error: `failed to create watermark file`

**Symptom:**
```
Error: failed to update watermark: failed to create watermark file: permission denied
```

**Cause:** No write permission to state directory.

**Solution:**

```bash
# Create state directory with write permissions
mkdir -p ~/.lastfm/state
chmod 755 ~/.lastfm/state

# Or use custom state path
export LASTFM_STATE="/tmp/lastfm-state"
mkdir -p /tmp/lastfm-state
```

---

### Error: `Azure storage: container not found`

**Symptom:**
```
Error: failed to create blob client: container "scrobbles" not found
```

**Cause:** Azure storage container doesn't exist.

**Solution:**

```bash
# Create container
az storage container create \
  --name scrobbles \
  --account-name lastfmstore \
  --auth-mode login

# Or using connection string
az storage container create \
  --name scrobbles \
  --connection-string "DefaultEndpointsProtocol=https;..."

# Verify
az storage container list \
  --account-name lastfmstore \
  --auth-mode login -o table
```

---

### Error: `Azure storage: unauthorized`

**Symptom:**
```
Error: failed to authenticate to Azure Storage: unauthorized
```

**Cause:** Missing storage permissions or invalid credentials.

**Solution:**

1. **For managed identity:**
   ```bash
   # Grant Storage Blob Data Contributor role
   PRINCIPAL_ID=$(az identity show \
     --name lastfm-identity \
     --resource-group lastfm-rg \
     --query principalId -o tsv)
   
   az role assignment create \
     --role "Storage Blob Data Contributor" \
     --assignee-object-id $PRINCIPAL_ID \
     --assignee-principal-type ServicePrincipal \
     --scope /subscriptions/{sub-id}/resourceGroups/lastfm-rg/providers/Microsoft.Storage/storageAccounts/lastfmstore
   
   # Verify
   az role assignment list \
     --assignee $PRINCIPAL_ID \
     -o table
   ```

2. **For SAS token:**
   ```bash
   # Generate new SAS token with correct permissions
   az storage container generate-sas \
     --account-name lastfmstore \
     --name scrobbles \
     --permissions rw \
     --expiry 2026-12-31T23:59:59Z \
     --https-only
   
   # Use full container URL with SAS
   export AZURE_CONTAINER_URL="https://lastfmstore.blob.core.windows.net/scrobbles?sv=...&sr=..."
   ```

---

## Debugging Commands

### Container Debugging

```bash
# Inspect Docker image
docker inspect lastfm-sync:latest

# Check image layers
docker history lastfm-sync:latest

# Verify binary exists
docker run --rm --entrypoint ls lastfm-sync:latest -la /app/

# Check environment variables
docker run --rm --entrypoint printenv lastfm-sync:latest

# Test with minimal config
docker run --rm \
  -e LASTFM_API_KEY="test-key" \
  lastfm-sync:latest \
  fetch --help
```

### Azure Debugging

```bash
# Container details
az container show \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --query "{State:instanceView.state,CPU:containers[0].resources.requests.cpu,Memory:containers[0].resources.requests.memoryInGb,ExitCode:containers[0].instanceView.currentState.exitCode}" \
  -o json

# Container events
az container show \
  --name lastfm-sync \
  --resource-group lastfm-rg \
  --query "instanceView.events" \
  -o table

# Resource group resources
az resource list \
  --resource-group lastfm-rg \
  --output table

# Activity logs
az monitor activity-log list \
  --resource-group lastfm-rg \
  --start-time 2026-01-06T00:00:00Z \
  --query "[?contains(resourceId, 'lastfm-sync')]" \
  -o table
```

### Network Debugging

```bash
# Test Last.fm API connectivity
curl -v "https://ws.audioscrobbler.com/2.0/?method=user.getinfo&user=rj&api_key=$LASTFM_API_KEY&format=json"

# Check DNS resolution
nslookup ws.audioscrobbler.com
dig ws.audioscrobbler.com +short

# Test HTTPS handshake
openssl s_client -connect ws.audioscrobbler.com:443 -servername ws.audioscrobbler.com

# Measure latency
time curl -s "https://ws.audioscrobbler.com/2.0/?method=user.getinfo&user=rj&api_key=test&format=json" > /dev/null
```

### Log Debugging

```bash
# Enable debug logging
export LASTFM_LOG_LEVEL=debug
lastfm-sync fetch --user alice --dry-run

# Structured logging output (JSON)
lastfm-sync fetch --user alice --dry-run 2>&1 | jq .

# Filter error logs
docker logs lastfm-sync 2>&1 | grep -i error

# Save logs with timestamps
docker logs lastfm-sync 2>&1 | while read line; do echo "$(date -Iseconds) $line"; done > debug.log
```

---

## Validation Checklist

### Pre-Deployment Checklist

- [ ] **Configuration**
  - [ ] `.env.example` exists and contains all variables
  - [ ] `.env` created and LASTFM_API_KEY set
  - [ ] `.env` not committed to git (`git check-ignore .env` passes)
  - [ ] API key valid (test with `curl`)

- [ ] **Docker**
  - [ ] Docker installed and running
  - [ ] Dockerfile builds successfully
  - [ ] Container starts without errors
  - [ ] Container runs as non-root (UID 65532)

- [ ] **Docker Compose**
  - [ ] `docker-compose.yml` present
  - [ ] Scripts executable (`chmod +x scripts/*.sh`)
  - [ ] `./scripts/dev-up.sh` runs without errors
  - [ ] Data directory has correct permissions

- [ ] **Azure (if deploying)**
  - [ ] Azure CLI installed
  - [ ] Logged in (`az account show`)
  - [ ] Resource group exists
  - [ ] Key Vault created with secrets
  - [ ] Managed identity created
  - [ ] RBAC roles assigned
  - [ ] Storage container created

### Post-Deployment Checklist

- [ ] **Functionality**
  - [ ] Container starts successfully
  - [ ] API key validated
  - [ ] Scrobbles fetched without errors
  - [ ] Output files created
  - [ ] Watermark updated correctly

- [ ] **Security**
  - [ ] No secrets in logs
  - [ ] No secrets in environment output
  - [ ] Container runs as non-root
  - [ ] Read-only root filesystem

- [ ] **Monitoring**
  - [ ] Logs visible in Azure Portal (if using ACI)
  - [ ] Log Analytics receiving data
  - [ ] Exit code is 0 on success
  - [ ] Metrics captured correctly

---

## FAQ

**Q: Why does `docker exec` fail with "no such file or directory"?**  
A: The container uses a distroless base image with no shell. This is **intentional for security**. Use `docker logs` instead.

**Q: How do I debug without a shell?**  
A: Use debug logging (`--log-level debug`), check container logs (`docker logs`), and inspect exit codes.

**Q: Can I run multiple sync instances simultaneously?**  
A: Yes, but use different output paths and usernames to avoid conflicts. Don't sync the same user simultaneously (rate limit issues).

**Q: How do I reset sync and start over?**  
A: Delete the watermark file:
```bash
# Local
rm ~/.lastfm/state/{username}.watermark

# Azure
az storage blob delete \
  --container-name scrobbles \
  --name lastfm/{username}.watermark \
  --account-name lastfmstore
```

**Q: What's the best way to test before production?**  
A: Use `--dry-run` flag:
```bash
lastfm-sync fetch --user alice --dry-run --log-level debug
```

**Q: How do I migrate from local to Azure storage?**  
A: 
1. Copy watermark to Azure
2. Change `--output local` to `--output azure`
3. Or delete watermark and do full resync

**Q: Why is my Azure container being throttled?**  
A: Azure Container Instances have CPU/memory limits. Increase allocation:
```bash
az container create --cpu 2 --memory 1 ...
```

**Q: How long does a full sync take?**  
A: At 3 QPS with 200 items/page: ~600 scrobbles/minute. For 100K scrobbles ≈ 3 hours.

**Q: Can I pause and resume a sync?**  
A: Yes! Watermarks track progress. Stop container and restart - it continues from last successful page.

**Q: How do I verify secrets are redacted in logs?**  
A: Check logs - API keys, connection strings, and SAS tokens automatically show as `****last4`.

---

## Additional Resources

- [Configuration Reference](configuration.md) - Complete environment variables and CLI flags
- [Docker Setup](docker.md) - Docker and Docker Compose documentation
- [Azure Deployment](azure-deployment.md) - Azure Container Instances deployment guide
- [Security Best Practices](security.md) - Secure configuration and secret management

---

**Need more help?** 
- Check application logs with `--log-level debug`
- Review [GitHub Issues](https://github.com/lastfm-reader/lastfm-sync/issues)
- Consult Last.fm API documentation: https://www.last.fm/api/
