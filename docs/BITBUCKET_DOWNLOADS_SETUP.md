# Bitbucket Downloads Setup Guide

This guide explains how to automate artifact uploads to Bitbucket Downloads.

## Current Configuration

The pipeline is configured to use **either**:
1. `BITBUCKET_DOWNLOADS_TOKEN` (if set) - dedicated token for downloads
2. `BITBUCKET_API_TOKEN` (fallback) - your existing workspace token

**Status**: ✅ The pipeline will now use your existing `BITBUCKET_API_TOKEN` automatically!

## How It Works

The pipeline uses this token priority:
```bash
TOKEN="${BITBUCKET_DOWNLOADS_TOKEN:-$BITBUCKET_API_TOKEN}"
```

This means:
- If `BITBUCKET_DOWNLOADS_TOKEN` exists, use it
- Otherwise, fall back to `BITBUCKET_API_TOKEN`
- If neither exists, skip upload

## What Gets Uploaded

When a new version tag is pushed (e.g., `v1.2.5`), the following files are automatically uploaded to Bitbucket Downloads:

### Static Library Distribution
- `go-mupdf-<version>-linux-amd64.tar.gz` - Pre-built MuPDF libraries
- `go-mupdf-<version>-linux-amd64.tar.gz.sha256` - Checksum file

### Release Artifacts
- `go-mupdf-<version>-source.tar.gz` - Source code archive
- `go-mupdf-<version>-docs.tar.gz` - Documentation package
- Any binary tools (if cmd/ directory exists)

## Testing the Upload

To test if uploads are working:

1. **Check the pipeline logs** after a tag push:
   ```
   Uploading go-mupdf-1.2.4-linux-amd64.tar.gz...
   ✓ Uploaded go-mupdf-1.2.4-linux-amd64.tar.gz successfully
   ```

2. **Verify on Downloads page**:
   - Go to: https://bitbucket.org/lexmata/go-mupdf/downloads/
   - Files should appear within seconds of upload

3. **Test download URL**:
   ```bash
   curl -LO https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-1.2.4-linux-amd64.tar.gz
   ```

## Optional: Create Dedicated Downloads Token

If you prefer to use a separate token specifically for downloads:

### Step 1: Create App Password

1. Go to: https://bitbucket.org/account/settings/app-passwords/
2. Click **"Create app password"**
3. **Label**: "Pipeline Downloads Upload"
4. **Permissions**: Select **"Repositories: Write"**
5. Click **"Create"**
6. **⚠️ Copy the password immediately** (it won't be shown again)

### Step 2: Add to Repository Variables

**Option A: Via Bitbucket UI**
1. Go to: https://bitbucket.org/lexmata/go-mupdf/admin/addon/admin/pipelines/repository-variables
2. Click **"Add variable"**
3. **Name**: `BITBUCKET_DOWNLOADS_TOKEN`
4. **Value**: [paste the app password]
5. ✅ Check **"Secured"** (hides from logs)
6. Click **"Add"**

**Option B: Via API** (automated)
```bash
# Set these variables
BITBUCKET_USERNAME="your-username"
BITBUCKET_APP_PASSWORD="your-app-password"  # The one you just created
REPO_OWNER="lexmata"
REPO_SLUG="go-mupdf"

# Create the repository variable
curl -X POST \
  "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines_config/variables/" \
  -u "${BITBUCKET_USERNAME}:${BITBUCKET_APP_PASSWORD}" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "BITBUCKET_DOWNLOADS_TOKEN",
    "value": "'${BITBUCKET_APP_PASSWORD}'",
    "secured": true
  }'
```

### Step 3: Verify

Push a new tag or re-run a tag pipeline:
```bash
git tag v1.2.5
git push origin v1.2.5
```

Check the "Build Static Library Distribution" step logs for:
```
Uploading distribution packages to Bitbucket Downloads...
✓ Uploaded go-mupdf-1.2.5-linux-amd64.tar.gz successfully
```

## Troubleshooting

### Upload Fails with "Failed to upload"

**Check token permissions:**
```bash
# Test your token
curl -u "username:token" \
  "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf"
```

If you get a 403, the token needs "Repositories: Write" permission.

### Upload Succeeds but Files Don't Appear

- Check the Downloads page after 30 seconds (there may be a delay)
- Verify the repository slug and owner are correct
- Check for file size limits (Bitbucket has a 2GB limit per file)

### Token Not Found

If you see "No authentication token available":
- The workspace-level `BITBUCKET_API_TOKEN` might not be accessible
- Create a repository-level `BITBUCKET_DOWNLOADS_TOKEN` instead

## Manual Upload (Emergency)

If automated upload fails, download artifacts from the pipeline and upload manually:

1. Go to the failed pipeline
2. Click on "Build Static Library Distribution" step
3. Click **"Artifacts"** tab
4. Download files
5. Go to: https://bitbucket.org/lexmata/go-mupdf/downloads/
6. Click **"Upload files"**
7. Select and upload the downloaded files

## Security Notes

- ✅ Always mark tokens as "Secured" in repository variables
- ✅ Use app passwords with minimal required permissions
- ✅ Never commit tokens to the repository
- ✅ Rotate tokens periodically (every 90 days recommended)
- ✅ Revoke tokens that are no longer needed

## Integration with setup-mupdf.sh

The `scripts/setup-mupdf.sh` script automatically downloads pre-built libraries from Bitbucket Downloads:

```bash
# Detects platform
PLATFORM="linux-amd64"
VERSION=$(cat VERSION)

# Downloads from Bitbucket
DOWNLOAD_URL="https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-${VERSION}-${PLATFORM}.tar.gz"
curl -fsSL "$DOWNLOAD_URL" -o /tmp/go-mupdf-libs.tar.gz

# Extracts to third_party/mupdf/build/release/
tar -xzf /tmp/go-mupdf-libs.tar.gz -C third_party/mupdf/
```

This makes `go get` installations much faster (seconds vs minutes).

## Next Steps

1. ✅ Changes committed - pipeline now uses existing token
2. Push any tag to test: `git tag v1.2.5 && git push origin v1.2.5`
3. Verify files appear at: https://bitbucket.org/lexmata/go-mupdf/downloads/
4. (Optional) Create dedicated token for better security isolation

## Related Documentation

- [CI/CD Optimization](./CI_CD_OPTIMIZATION.md) - Pipeline architecture
- [Static Library Distribution](./STATIC_LIBRARY_DISTRIBUTION.md) - Build details
- [Pipeline Monitoring](./PIPELINE_MONITORING.md) - Monitoring tools

