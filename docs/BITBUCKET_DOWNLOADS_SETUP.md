# Bitbucket Downloads Setup Guide

This guide explains how to automate artifact uploads to Bitbucket Downloads.

## Current Configuration

The pipeline requires **TWO repository variables**:
1. `BITBUCKET_USERNAME` - Your Bitbucket account username/email
2. `BITBUCKET_DOWNLOADS_TOKEN` - App password with "Repositories: Write" permission

**Important**: Both variables must be set for uploads to work. Bitbucket requires Basic Authentication with a username and app password for the Downloads API.

## How It Works

The pipeline uses Basic Authentication:
```bash
USERNAME="${BITBUCKET_USERNAME}"
TOKEN="${BITBUCKET_DOWNLOADS_TOKEN:-$BITBUCKET_API_TOKEN}"
curl -u "$USERNAME:$TOKEN" ...
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

## Required Setup

Both variables must be configured for automated uploads to work.

### Step 1: Get Your Bitbucket Username

Your Bitbucket username is usually your email address. To confirm:

1. Go to: https://bitbucket.org/account/settings/
2. Look for **"Bitbucket username"** or **"Email"**
3. Note this value (e.g., `jquinn@advita.us`)

### Step 2: Create App Password

1. Go to: https://bitbucket.org/account/settings/app-passwords/
2. Click **"Create app password"**
3. **Label**: "Pipeline Downloads Upload"
4. **Permissions**: Select **"Repositories: Write"** ✅
5. Click **"Create"**
6. **⚠️ Copy the password immediately** (it won't be shown again)

### Step 3: Add Both Variables to Repository

**Via Bitbucket UI:**

1. Go to: https://bitbucket.org/lexmata/go-mupdf/admin/addon/admin/pipelines/repository-variables

2. **Add first variable:**
   - Click **"Add variable"**
   - **Name**: `BITBUCKET_USERNAME`
   - **Value**: [your username from Step 1, e.g., `jquinn@advita.us`]
   - ✅ Check **"Secured"** (optional but recommended)
   - Click **"Add"**

3. **Add second variable:**
   - Click **"Add variable"**
   - **Name**: `BITBUCKET_DOWNLOADS_TOKEN`
   - **Value**: [paste the app password from Step 2]
   - ✅ Check **"Secured"** (required)
   - Click **"Add"**

**Via API (automated):**

```bash
# Set these variables
BB_USERNAME="jquinn@advita.us"  # Your Bitbucket username
BB_APP_PASSWORD="ATBBxyz..."    # The app password you just created
REPO_OWNER="lexmata"
REPO_SLUG="go-mupdf"

# Add BITBUCKET_USERNAME variable
curl -X POST \
  "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines_config/variables/" \
  -u "${BB_USERNAME}:${BB_APP_PASSWORD}" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "BITBUCKET_USERNAME",
    "value": "'${BB_USERNAME}'",
    "secured": false
  }'

# Add BITBUCKET_DOWNLOADS_TOKEN variable
curl -X POST \
  "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines_config/variables/" \
  -u "${BB_USERNAME}:${BB_APP_PASSWORD}" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "BITBUCKET_DOWNLOADS_TOKEN",
    "value": "'${BB_APP_PASSWORD}'",
    "secured": true
  }'
```

### Step 4: Verify

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

### Upload Fails with "BITBUCKET_USERNAME not set"

**Problem**: The USERNAME variable is missing.

**Solution**: Add `BITBUCKET_USERNAME` as shown in Step 3 above.

### Upload Fails with "Authentication Issue"

**Problem**: Either username or token is incorrect.

**Test your credentials:**
```bash
# Replace with your actual values
USERNAME="jquinn@advita.us"
TOKEN="ATBBxyz..."

# Test basic auth
curl -u "$USERNAME:$TOKEN" \
  "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf"

# Should return repository info (HTTP 200)
# If you get 401, the username or token is wrong
```

**Test file upload:**
```bash
# Create test file
echo "test" > test-upload.txt

# Try uploading
curl -X POST \
  "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/downloads" \
  -u "$USERNAME:$TOKEN" \
  -F "files=@test-upload.txt"

# Should return HTTP 201 with JSON response
# If you get 401, token lacks "Repositories: Write" permission
```

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

1. ✅ Changes committed - pipeline now uses Basic Auth
2. **Add both repository variables** as shown in "Required Setup" above
3. Push any tag to test: `git tag v1.2.6 && git push origin v1.2.6`
4. Verify files appear at: https://bitbucket.org/lexmata/go-mupdf/downloads/
5. Check pipeline logs for "✓ Uploaded ... successfully" messages

## Related Documentation

- [CI/CD Optimization](./CI_CD_OPTIMIZATION.md) - Pipeline architecture
- [Static Library Distribution](./STATIC_LIBRARY_DISTRIBUTION.md) - Build details
- [Pipeline Monitoring](./PIPELINE_MONITORING.md) - Monitoring tools

