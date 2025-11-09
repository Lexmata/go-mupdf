# Bitbucket Pipeline Monitoring Guide

This guide covers various methods to monitor Bitbucket Pipelines for the go-mupdf project.

## Quick Start

### Option 1: Web UI (Easiest)

Simply visit the Pipeline dashboard in your browser:
```
https://bitbucket.org/lexmata/go-mupdf/addon/pipelines/home
```

Features:
- ✅ Real-time log streaming
- ✅ Step-by-step progress
- ✅ Artifact downloads
- ✅ No setup required

### Option 2: Terminal Monitor Script (Recommended)

We've provided a bash script for terminal monitoring:

```bash
# Watch the latest pipeline
./scripts/watch-pipeline.sh

# Watch a specific pipeline by UUID
./scripts/watch-pipeline.sh {pipeline-uuid}
```

**Requirements:**
- `jq` (JSON processor)
- `curl`

Install on Ubuntu/Debian:
```bash
sudo apt-get install jq curl
```

**Features:**
- ✅ Auto-refreshes every 10 seconds
- ✅ Color-coded status
- ✅ Shows all pipeline steps
- ✅ Notifies when complete
- ✅ Works in tmux/screen

**Output Example:**
```
╔══════════════════════════════════════════════════════════════╗
║         Bitbucket Pipeline Monitor - go-mupdf               ║
╚══════════════════════════════════════════════════════════════╝

Pipeline Status:
{
  "build": 42,
  "state": "IN_PROGRESS",
  "target": "v1.2.4",
  "created": "2025-11-09T..."
}

Pipeline Steps:

  Build MuPDF Libraries                    ✅ SUCCESSFUL
  Lint Code                                ✅ SUCCESSFUL
  Run Tests for Release                    ⏳ IN_PROGRESS
  Build Release Artifacts                  ⏸  PENDING
  Build Static Library Distribution        ⏸  PENDING

───────────────────────────────────────────────────────────────
Refreshing in 10s... (Ctrl+C to stop) [Iteration: 5]
```

## Advanced Monitoring Options

### Option 3: Bitbucket API Direct Queries

Use the Bitbucket REST API directly:

#### Get Latest Pipeline
```bash
curl -s "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/?sort=-created_on&pagelen=1" | jq '.'
```

#### Get Pipeline by UUID
```bash
PIPELINE_UUID="{12345678-1234-1234-1234-123456789012}"
curl -s "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/${PIPELINE_UUID}" | jq '.'
```

#### Get Pipeline Steps
```bash
PIPELINE_UUID="{12345678-1234-1234-1234-123456789012}"
curl -s "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/${PIPELINE_UUID}/steps/" | jq '.values[] | {name: .name, state: .state.name}'
```

#### Get Step Logs
```bash
PIPELINE_UUID="{12345678-1234-1234-1234-123456789012}"
STEP_UUID="{step-uuid}"
curl -s "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/${PIPELINE_UUID}/steps/${STEP_UUID}/log" | jq -r '.log'
```

### Option 4: Watch with `watch` Command

Simple one-liner to check pipeline status:

```bash
watch -n 10 -c 'curl -s "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/?sort=-created_on&pagelen=1" | jq -r ".values[0] | \"Build: \(.build_number) | State: \(.state.name) | Result: \(.state.result.name // \"IN_PROGRESS\") | Target: \(.target.ref_name)\""'
```

### Option 5: Slack/Email Notifications

Configure notifications in Bitbucket:

1. Go to Repository Settings → Pipelines → Settings
2. Enable "Email notifications"
3. Or integrate with Slack:
   - Repository Settings → Integrations
   - Add Slack webhook

## Monitoring Release Pipelines

For release builds (tags), you can specifically monitor:

```bash
# Get pipelines for a specific tag
curl -s "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/?target.ref_name=v1.2.4&sort=-created_on" | jq '.values[0]'
```

## Pipeline States Reference

### Pipeline States
- `PENDING` - Queued, waiting to start
- `IN_PROGRESS` - Currently running
- `COMPLETED` - Finished (check result)

### Pipeline Results
- `SUCCESSFUL` ✅ - All steps passed
- `FAILED` ❌ - One or more steps failed
- `ERROR` ⚠️ - Pipeline error
- `STOPPED` ⏹ - Manually stopped

### Step States
- `PENDING` ⏸ - Waiting to run
- `IN_PROGRESS` ⏳ - Currently executing
- `COMPLETED` - Finished (check result)

## Troubleshooting

### Script Issues

**"jq: command not found"**
```bash
sudo apt-get update && sudo apt-get install -y jq
```

**API Rate Limiting**
If you hit API rate limits, wait a few minutes or authenticate:
```bash
# With authentication (requires app password)
curl -u "username:app_password" "https://api.bitbucket.org/2.0/..."
```

### Get Pipeline UUID

If you need to find a pipeline UUID:

1. From the web UI URL:
   ```
   https://bitbucket.org/lexmata/go-mupdf/addon/pipelines/results/{pipeline-uuid}
   ```

2. From API:
   ```bash
   curl -s "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/?sort=-created_on&pagelen=5" | jq -r '.values[] | "\(.build_number): {\(.uuid)}"'
   ```

## CI/CD Performance Monitoring

Track pipeline performance over time:

```bash
# Get last 10 pipeline durations
curl -s "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/?sort=-created_on&pagelen=10" | jq -r '.values[] | "\(.build_number) | \(.target.ref_name) | \(.duration_in_seconds)s | \(.state.result.name)"'
```

Expected durations:
- Development builds: 3-4 minutes (download pre-built libs)
- Release builds: 19-28 minutes (build MuPDF from source)

## Integration with CI/CD

### Pre-push Hook Integration

Add to `.githooks/pre-push` to wait for pipeline:

```bash
# Optional: Wait for pipeline to start
echo "Waiting for pipeline to start..."
sleep 10
./scripts/watch-pipeline.sh
```

### Makefile Integration

Add to `Makefile`:

```makefile
.PHONY: watch-pipeline
watch-pipeline:
	@echo "Monitoring latest pipeline..."
	@./scripts/watch-pipeline.sh
```

Usage:
```bash
make watch-pipeline
```

## Related Documentation

- [CI/CD Optimization](CI_CD_OPTIMIZATION.md)
- [Maintainer Guide](MAINTAINER_GUIDE.md)
- [Bitbucket Pipelines Documentation](https://support.atlassian.com/bitbucket-cloud/docs/get-started-with-bitbucket-pipelines/)
- [Bitbucket API Reference](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/)

## Tips

1. **Run in tmux/screen**: Start the monitor in a tmux pane to keep it running
   ```bash
   tmux new -s pipeline-watch
   ./scripts/watch-pipeline.sh
   # Detach with Ctrl+B, D
   ```

2. **Multiple pipelines**: Open multiple terminals for concurrent monitoring

3. **Pipeline artifacts**: Download from web UI or via API:
   ```bash
   # List artifacts
   curl "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/${PIPELINE_UUID}/steps/${STEP_UUID}/artifacts"
   ```

4. **Failed step logs**: Quickly get logs for failed steps:
   ```bash
   ./scripts/watch-pipeline.sh | tee pipeline-output.log
   ```

## Quick Reference

| Action | Command |
|--------|---------|
| Watch latest | `./scripts/watch-pipeline.sh` |
| Watch specific | `./scripts/watch-pipeline.sh {uuid}` |
| Web UI | https://bitbucket.org/lexmata/go-mupdf/addon/pipelines/home |
| API latest | `curl "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/pipelines/?sort=-created_on&pagelen=1"` |
| Help | `./scripts/watch-pipeline.sh --help` |

