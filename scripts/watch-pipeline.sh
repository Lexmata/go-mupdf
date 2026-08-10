#!/bin/bash
# Watch Bitbucket Pipeline Status
# Usage: ./scripts/watch-pipeline.sh [pipeline-uuid]
#
# Optional: set BITBUCKET_USERNAME and BITBUCKET_APP_PASSWORD to authenticate
# API requests (required for private repositories / higher rate limits).

set -e
set -o pipefail

REPO_OWNER="lexmata"
REPO_SLUG="go-mupdf"
REFRESH_INTERVAL=10

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
GRAY='\033[0;90m'
NC='\033[0m'

MAX_CONSECUTIVE_FAILURES=5

# Optional authentication: used when both BITBUCKET_USERNAME and
# BITBUCKET_APP_PASSWORD are set in the environment
CURL_AUTH=()
if [ -n "${BITBUCKET_USERNAME:-}" ] && [ -n "${BITBUCKET_APP_PASSWORD:-}" ]; then
    CURL_AUTH=(-u "${BITBUCKET_USERNAME}:${BITBUCKET_APP_PASSWORD}")
fi

# Curl wrapper: fail on HTTP errors (-f), silent (-s), no URL globbing (-g,
# so literal {braces} in pipeline UUIDs are passed through)
bb_curl() {
    curl -sfg "${CURL_AUTH[@]}" "$@"
}

# Function to get latest pipeline
get_latest_pipeline() {
    bb_curl "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines/?sort=-created_on&pagelen=1" \
        | jq -r '.values[0] | {uuid: .uuid, build: .build_number, state: .state.name, target: .target.ref_name, created: .created_on}'
}

# Function to get pipeline details (takes a bare UUID; braces are added here,
# exactly once, where the API path requires them)
get_pipeline_status() {
    local pipeline_uuid=$1
    bb_curl "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines/{${pipeline_uuid}}" \
        | jq -r '. | {
            build: .build_number,
            state: .state.name,
            result: .state.result.name,
            target: .target.ref_name,
            trigger: .trigger.name,
            created: .created_on,
            completed: .completed_on,
            duration_seconds: .duration_in_seconds
        }'
}

# Function to get pipeline steps (takes a bare UUID; braces are added here,
# exactly once, where the API path requires them)
get_pipeline_steps() {
    local pipeline_uuid=$1
    bb_curl "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines/{${pipeline_uuid}}/steps/" \
        | jq -r '.values[] | [.uuid, .name, .state.name, .state.result.name // "PENDING"] | @tsv'
}

# Function to display status with color
status_color() {
    local state=$1
    case "$state" in
        SUCCESSFUL) echo -e "${GREEN}✅ $state${NC}" ;;
        FAILED) echo -e "${RED}❌ $state${NC}" ;;
        IN_PROGRESS) echo -e "${YELLOW}⏳ $state${NC}" ;;
        PENDING) echo -e "${GRAY}⏸  $state${NC}" ;;
        STOPPED) echo -e "${YELLOW}⏹  $state${NC}" ;;
        *) echo -e "${BLUE}$state${NC}" ;;
    esac
}

# Main monitoring function
watch_pipeline() {
    local pipeline_uuid=$1

    echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║         Bitbucket Pipeline Monitor - go-mupdf               ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    if [ -z "$pipeline_uuid" ]; then
        echo "Getting latest pipeline..."
        if ! pipeline_info=$(get_latest_pipeline) || [ -z "$pipeline_info" ]; then
            echo -e "${RED}Error: failed to fetch the latest pipeline from the Bitbucket API${NC}"
            exit 1
        fi
        pipeline_uuid=$(echo "$pipeline_info" | jq -r '.uuid')
        echo "Monitoring pipeline: $pipeline_uuid"
        echo "$pipeline_info" | jq '.'
        echo ""
    fi

    # Normalize once at entry: strip any braces, whether the UUID came from
    # the API or was passed as {uuid} on the command line. The API helpers
    # re-add braces exactly once where needed.
    pipeline_uuid=$(echo "$pipeline_uuid" | tr -d '{}')

    local last_state=""
    local iteration=0
    local consecutive_failures=0

    while true; do
        # Get pipeline status (before clearing, so errors stay visible)
        if ! status=$(get_pipeline_status "$pipeline_uuid") || [ -z "$status" ]; then
            consecutive_failures=$((consecutive_failures + 1))
            echo -e "${RED}Failed to fetch pipeline status from the Bitbucket API (attempt ${consecutive_failures}/${MAX_CONSECUTIVE_FAILURES})${NC}"
            if [ "$consecutive_failures" -ge "$MAX_CONSECUTIVE_FAILURES" ]; then
                echo -e "${RED}Error: aborting after ${MAX_CONSECUTIVE_FAILURES} consecutive API failures. Check the pipeline UUID, your network connection, and BITBUCKET_USERNAME/BITBUCKET_APP_PASSWORD.${NC}"
                exit 1
            fi
            sleep $REFRESH_INTERVAL
            continue
        fi
        consecutive_failures=0

        clear
        echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${BLUE}║         Bitbucket Pipeline Monitor - go-mupdf               ║${NC}"
        echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
        echo ""

        state=$(echo "$status" | jq -r '.state')

        echo -e "${BLUE}Pipeline Status:${NC}"
        echo "$status" | jq '.'
        echo ""

        echo -e "${BLUE}Pipeline Steps:${NC}"
        echo ""

        # Get and display steps (a transient failure here is non-fatal)
        get_pipeline_steps "$pipeline_uuid" | while IFS=$'\t' read -r step_uuid step_name step_state step_result; do
            printf "  %-40s " "$step_name"
            if [ "$step_state" = "COMPLETED" ]; then
                status_color "$step_result"
            else
                status_color "$step_state"
            fi
        done || echo -e "${YELLOW}(failed to fetch pipeline steps)${NC}"

        echo ""
        echo -e "${GRAY}───────────────────────────────────────────────────────────────${NC}"

        # Check if completed
        if [ "$state" = "COMPLETED" ]; then
            result=$(echo "$status" | jq -r '.result')
            echo ""
            echo -e "Pipeline completed with result: $(status_color "$result")"
            echo ""
            echo "View full details at:"
            echo "https://bitbucket.org/${REPO_OWNER}/${REPO_SLUG}/addon/pipelines/results/$pipeline_uuid"
            break
        fi

        echo ""
        echo -e "${GRAY}Refreshing in ${REFRESH_INTERVAL}s... (Ctrl+C to stop) [Iteration: $((++iteration))]${NC}"
        sleep $REFRESH_INTERVAL
    done
}

# Check dependencies
if ! command -v jq >/dev/null 2>&1; then
    echo "Error: jq is required but not installed."
    echo "Install with: sudo apt-get install jq"
    exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
    echo "Error: curl is required but not installed."
    exit 1
fi

# Show usage
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << EOF
Bitbucket Pipeline Monitor

Usage:
  $0                 Watch latest pipeline
  $0 [pipeline-uuid] Watch specific pipeline
  $0 --help          Show this help

Examples:
  $0
  $0 {12345678-1234-1234-1234-123456789012}

EOF
    exit 0
fi

# Run monitor
watch_pipeline "$1"

