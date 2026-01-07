# Quickstart Guide: LastFM Sync with Docker

**Goal**: Get from zero to running lastfm-sync container in 5 minutes

---

## Prerequisites (30 seconds)

✅ **Docker** installed and running
- Check: `docker --version` (should show Docker 20.10+ or newer)
- Install: [Get Docker](https://docs.docker.com/get-docker/)

✅ **Git** installed
- Check: `git --version`

✅ **Last.fm API Key**
- Get one (free): [Last.fm API Account](https://www.last.fm/api/account/create)
- You'll receive a 32-character hex key

---

## Step 1: Clone Repository (30 seconds)

```bash
git clone https://github.com/lastfm-reader/lastfm-sync.git
cd lastfm-sync
```

---

## Step 2: Configure Environment (1 minute)

Create your environment file from the template:

```bash
cp .env.example .env
```

Edit `.env` and set your credentials:

```bash
# Required: Your Last.fm API key and username
LASTFM_API_KEY=your_32_character_api_key_here
LASTFM_USER=your_lastfm_username

# Optional: Adjust these if needed
# LASTFM_QPS=3
# LASTFM_LOG=info
```

💡 **Tip**: On macOS/Linux, use `nano .env` or your favorite editor.

---

## Step 3: Run with Docker Compose (2 minutes)

Start the container:

```bash
docker compose up --build
```

**What happens**:
1. Docker builds the image (~1-2 minutes first time)
2. Container starts and runs with default `--help` command
3. You'll see usage information in logs

**First build output** (example):
```
[+] Building 120.3s (12/12) FINISHED
 => [build 1/5] FROM golang:1.25-alpine
 => [build 2/5] COPY go.mod go.sum ./
 => [build 3/5] RUN go mod download
 => [build 4/5] COPY . .
 => [build 5/5] RUN go build ...
 => [stage-1 1/1] COPY --from=build /out/lastfm-sync /app/lastfm-sync
 => exporting to image
```

---

## Step 4: Fetch Your Scrobbles (1 minute)

Run a real fetch command:

```bash
docker compose run --rm lastfm-sync fetch --user $LASTFM_USER
```

**Expected output**:
```
INFO  Starting lastfm-sync v1.0.0
INFO  Fetching scrobbles for user: alice
INFO  Rate limit: 3 QPS
INFO  Fetched 200 scrobbles (page 1/5)
INFO  Fetched 200 scrobbles (page 2/5)
...
INFO  Sync complete: 1000 scrobbles written to /data/alice.ndjson
INFO  Watermark updated: 1735689600
```

---

## Step 5: Verify Output (30 seconds)

Check the generated file:

```bash
ls -lh data/
# Should show: alice.ndjson

head -n 1 data/alice.ndjson | jq
# Should show JSON scrobble record
```

**Example scrobble record**:
```json
{
  "username": "alice",
  "artist": "Radiohead",
  "track": "Idioteque",
  "album": "Kid A",
  "uts": 1735689600,
  "mbid": "12345-...",
  "source": "lastfm",
  "ingested_at": "2026-01-06T18:00:00Z",
  "raw": { ... }
}
```

---

## ✅ Success! You're Running LastFM Sync

### What You've Accomplished

- ✅ Built a production-ready container image
- ✅ Fetched your Last.fm scrobble history
- ✅ Stored data in NDJSON format
- ✅ Set up persistent local storage

---

## Next Steps

### 🚀 Advanced Usage

**Incremental Sync** (only fetch new scrobbles):
```bash
docker compose run --rm lastfm-sync fetch --user $LASTFM_USER
```
The watermark is stored automatically. Second run only fetches new records!

**Custom Date Range**:
```bash
docker compose run --rm lastfm-sync fetch \
  --user $LASTFM_USER \
  --since 1704067200 \
  --until 1735689600
```

**Debug Mode**:
```bash
# Edit .env: LASTFM_LOG=debug
docker compose run --rm lastfm-sync fetch --user $LASTFM_USER
```

### 📚 Read the Documentation

- **[Configuration Guide](../docs/configuration.md)**: All environment variables and CLI flags
- **[Docker Guide](../docs/docker.md)**: Advanced Docker and Compose usage
- **[Azure Deployment](../docs/azure-deployment.md)**: Deploy to Azure Container Instances
- **[Security Best Practices](../docs/security.md)**: Secure secret management
- **[Troubleshooting](../docs/troubleshooting.md)**: Common issues and solutions

### 🛠️ Helper Scripts

For convenience, use the provided scripts:

```bash
# Start development environment
./scripts/dev-up.sh

# Stop and clean up
./scripts/dev-down.sh
```

### ☁️ Deploy to Azure

Ready for production? Deploy to Azure Container Instances:

```bash
# Follow the Azure deployment guide
cat docs/azure-deployment.md

# Quick deploy (requires Azure CLI)
./azure/deploy-aci.sh
```

---

## Common Issues

### "Permission denied" on data directory

**Problem**: Container can't write to `./data/`

**Solution**:
```bash
mkdir -p data
chmod 777 data  # Or chown to your user
```

### "LASTFM_API_KEY is required"

**Problem**: Environment variable not set

**Solution**: Verify `.env` file exists and has `LASTFM_API_KEY=...`

### "Rate limit exceeded"

**Problem**: Last.fm API rate limit hit

**Solution**: Reduce QPS in `.env`:
```bash
LASTFM_QPS=1  # Slower but safer
```

---

## Cleanup

Stop all containers and remove volumes:

```bash
docker compose down -v
```

Remove images (optional):

```bash
docker rmi lastfm-sync:dev
```

---

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/lastfm-reader/lastfm-sync/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lastfm-reader/lastfm-sync/discussions)
- **Documentation**: [Full docs](../docs/)

---

**Total Time**: ⏱️ 5 minutes (build included)

**You're now ready to sync your Last.fm history!** 🎵
